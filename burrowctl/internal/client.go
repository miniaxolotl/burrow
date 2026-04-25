package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"burrow/protocol"

	"github.com/gorilla/websocket"
	"github.com/xtaci/yamux"
)

type countWriter struct {
	w io.Writer
	n *int64
}

func (cw *countWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	atomic.AddInt64(cw.n, int64(n))
	return n, err
}


type Client struct {
	server  string
	token   string
	domain  string
	secure  bool
	tunnels map[uint16]*TunnelConn
	mu      sync.RWMutex
}

type TunnelConn struct {
	ID         string
	Port       uint16
	URL        string
	wsConn     *websocket.Conn
	session    *yamux.Session
	ctx        context.Context
	cancel     context.CancelFunc
	latency    time.Duration
	reconnects int
	totalSize  int64
}

func NewClient(server, token, domain string, secure bool) *Client {
	return &Client{
		server:  server,
		token:   token,
		domain:  domain,
		secure:  secure,
		tunnels: make(map[uint16]*TunnelConn),
	}
}

func (c *Client) CreateTunnel(ctx context.Context, port uint16) (string, error) {
	tunnelID := protocol.RandomTunnelID()
	tc, err := c.createTunnelSession(ctx, tunnelID, port)
	if err != nil {
		return "", err
	}

	tctx, cancel := context.WithCancel(context.Background())
	tc.ctx = tctx
	tc.cancel = cancel

	c.mu.Lock()
	c.tunnels[port] = tc
	c.mu.Unlock()

	go c.trackLatency(tc)
	go c.handleTunnel(tc, port)

	return tc.URL, nil
}

func (c *Client) trackLatency(tc *TunnelConn) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-tc.ctx.Done():
			return
		case <-ticker.C:
			start := time.Now()
			scheme := "http"
			if c.secure {
				scheme = "https"
			}
			req, err := http.NewRequest("GET", fmt.Sprintf("%s://%s/health", scheme, c.server), nil)
			if err != nil {
				continue
			}
			req.Header.Set("X-Tunnel-Token", c.token)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()
			tc.latency = time.Since(start)
		}
	}
}

func (c *Client) createTunnelSession(ctx context.Context, tunnelID string, port uint16) (*TunnelConn, error) {
	header := http.Header{"X-Tunnel-Token": []string{c.token}}
	scheme := "ws"
	if c.secure {
		scheme = "wss"
	}
	wsURL := fmt.Sprintf("%s://%s/tunnel/ws?tunnel_id=%s&port=%d", scheme, c.server, tunnelID, port)

	start := time.Now()
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	latency := time.Since(start)

	tunnelURL := ""
	if resp != nil {
		tunnelURL = resp.Header.Get("X-Tunnel-URL")
	}
	if tunnelURL == "" {
		tunnelURL = fmt.Sprintf("https://%s.%s", tunnelID, c.domain)
	}

	cfg := yamux.DefaultConfig()
	cfg.KeepAliveInterval = 30 * time.Second
	wrappedConn := &protocol.WsConn{Conn: conn}
	session, err := yamux.Client(wrappedConn, cfg)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create yamux session: %w", err)
	}

	return &TunnelConn{
		ID:      tunnelID,
		Port:    port,
		URL:     tunnelURL,
		wsConn:  conn,
		session: session,
		latency: latency,
	}, nil
}

// maxReconnectAttempts is the number of consecutive failures before abandoning
// the current tunnel ID and creating a fresh one. The backoff reaches its 30s
// cap after ~5 attempts (~62s total), so this is a reasonable switchover point.
const maxReconnectAttempts = 5

func (c *Client) handleTunnel(tc *TunnelConn, port uint16) {
	defer tc.cancel()
	defer func() {
		c.mu.Lock()
		if c.tunnels[port] == tc {
			delete(c.tunnels, port)
		}
		c.mu.Unlock()
	}()

	localAddr := fmt.Sprintf("localhost:%d", port)
	reconnectDelay := time.Second
	failCount := 0

	for {
		stream, err := tc.session.AcceptStream()
		if err != nil {
			if tc.ctx.Err() != nil {
				return
			}

		select {
			case <-tc.ctx.Done():
				return
			case <-time.After(reconnectDelay):
			}
			if reconnectDelay < 30*time.Second {
				reconnectDelay *= 2
			}

			tunnelID := tc.ID
			if failCount >= maxReconnectAttempts {
				tunnelID = protocol.RandomTunnelID()
			}

			newTC, err := c.createTunnelSession(tc.ctx, tunnelID, port)
			if err != nil {
				if tc.ctx.Err() != nil {
					return
				}
				failCount++
				continue
			}

			tc.wsConn.Close()
			tc.wsConn, tc.session = newTC.wsConn, newTC.session
			tc.latency = newTC.latency
			tc.reconnects++
			if tunnelID != tc.ID {
				tc.ID, tc.URL = tunnelID, newTC.URL
			} else {
				tc.URL = newTC.URL
			}
			reconnectDelay = time.Second
			failCount = 0
			continue
		}

		reconnectDelay = time.Second
		failCount = 0
		go func(s *yamux.Stream) {
			defer s.Close()

			conn, err := net.DialTimeout("tcp", localAddr, 5*time.Second)
			if err != nil {
				return
			}
			defer conn.Close()

			done := make(chan struct{}, 2)
			go func() { io.Copy(&countWriter{conn, &tc.totalSize}, s); conn.Close(); done <- struct{}{} }()
			go func() { io.Copy(&countWriter{s, &tc.totalSize}, conn); s.Close(); done <- struct{}{} }()
			<-done
			<-done
		}(stream)
	}
}

func (c *Client) CloseTunnel(port uint16) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if tc, ok := c.tunnels[port]; ok {
		tc.cancel()
		tc.session.Close()
		tc.wsConn.Close()
		delete(c.tunnels, port)
	}
	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, tc := range c.tunnels {
		tc.cancel()
		tc.session.Close()
		tc.wsConn.Close()
	}
	c.tunnels = make(map[uint16]*TunnelConn)
	return nil
}

func (c *Client) ListTunnels() []*protocol.TunnelInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tunnels := make([]*protocol.TunnelInfo, 0, len(c.tunnels))
	for port, tc := range c.tunnels {
		tunnels = append(tunnels, &protocol.TunnelInfo{
			TunnelID:   tc.ID,
			Port:       port,
			URL:        tc.URL,
			Status:     "active",
			Latency:    tc.latency.Round(time.Millisecond).String(),
			Reconnects: tc.reconnects,
			TotalSize:  atomic.LoadInt64(&tc.totalSize),
		})
	}
	sort.Slice(tunnels, func(i, j int) bool {
		return tunnels[i].Port < tunnels[j].Port
	})
	return tunnels
}

func (c *Client) GetLogs(tunnelID string) ([]*protocol.TunnelLog, error) {
	scheme := "http"
	if c.secure {
		scheme = "https"
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s://%s/logs/%s", scheme, c.server, tunnelID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("X-Tunnel-Token", c.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to contact server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}

	var logs []*protocol.TunnelLog
	if err := json.NewDecoder(resp.Body).Decode(&logs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return logs, nil
}
