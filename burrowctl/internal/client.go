package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"burrow/protocol"
	"burrow/protocol/telemetry"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

type Client struct {
	server   string
	token    string
	domain   string
	secure   bool
	tunnels  map[uint16]*TunnelConn
	mu       sync.RWMutex
	observer telemetry.Observer
}

var httpClient = NewManagementHTTPClient()

type connectionRejected struct {
	status  int
	message string
}

func (e *connectionRejected) Error() string { return e.message }

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
	forwarded  sync.Once
}

func NewClient(server, token, domain string, secure bool, observers ...telemetry.Observer) *Client {
	var observer telemetry.Observer = (*telemetry.Reporter)(nil)
	if len(observers) > 0 && observers[0] != nil {
		observer = observers[0]
	}
	return &Client{
		server:   server,
		token:    token,
		domain:   domain,
		secure:   secure,
		tunnels:  make(map[uint16]*TunnelConn),
		observer: observer,
	}
}

func (c *Client) CreateTunnel(ctx context.Context, port uint16) (string, error) {
	started := time.Now()
	outcome := telemetry.Unavailable
	defer func() { c.observer.Operation(telemetry.Connect, started, outcome) }()
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

	tunnelURL := tc.URL
	go c.trackLatency(tc)
	go c.handleTunnel(tc, port)

	outcome = telemetry.Success
	c.observer.Track(telemetry.TunnelOpened)
	return tunnelURL, nil
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
			req, err := http.NewRequestWithContext(tc.ctx, "GET", fmt.Sprintf("%s://%s/health", scheme, c.server), nil)
			if err != nil {
				continue
			}
			resp, err := httpClient.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()
			c.mu.Lock()
			tc.latency = time.Since(start)
			c.mu.Unlock()
		}
	}
}

func (c *Client) createTunnelSession(ctx context.Context, tunnelID string, port uint16) (*TunnelConn, error) {
	header := http.Header{"X-Tunnel-Token": []string{c.token}}
	scheme := "ws"
	if c.secure {
		scheme = "wss"
	}
	wsURL := fmt.Sprintf("%s://%s/tunnels/connect?tunnel_id=%s&port=%d", scheme, c.server, tunnelID, port)

	start := time.Now()
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
			resp.Body.Close()
			msg := strings.TrimSpace(string(body))
			if msg != "" {
				return nil, &connectionRejected{resp.StatusCode, fmt.Sprintf("server rejected connection: %s: %s", resp.Status, msg)}
			}
			return nil, &connectionRejected{resp.StatusCode, fmt.Sprintf("server rejected connection: %s", resp.Status)}
		}
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	conn.SetReadLimit(protocol.MaxMessageSize)
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
	// Expected disconnects and remote closure must not corrupt the TUI output.
	cfg.LogOutput = io.Discard
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
			reconnectDelay = min(reconnectDelay*2, 30*time.Second)

			// Persistent ownership makes the original ID safe to retry. Switching
			// IDs after failures would bypass a closure received while offline.
			started := time.Now()
			newTC, err := c.createTunnelSession(tc.ctx, tc.ID, port)
			if err != nil {
				c.observer.Operation(telemetry.Reconnect, started, telemetry.Unavailable)
				var rejected *connectionRejected
				if errors.As(err, &rejected) && (rejected.status == http.StatusUnauthorized || rejected.status == http.StatusForbidden || rejected.status == http.StatusConflict || rejected.status == http.StatusGone) {
					return
				}
				if tc.ctx.Err() != nil {
					return
				}
				continue
			}

			c.mu.Lock()
			if tc.ctx.Err() != nil {
				c.mu.Unlock()
				newTC.session.Close()
				newTC.wsConn.Close()
				return
			}
			tc.wsConn.Close()
			tc.wsConn, tc.session = newTC.wsConn, newTC.session
			tc.latency = newTC.latency
			tc.reconnects++
			tc.URL = newTC.URL
			c.mu.Unlock()
			c.observer.Track(telemetry.TunnelReconnected)
			c.observer.Operation(telemetry.Reconnect, started, telemetry.Success)
			reconnectDelay = time.Second
			continue
		}

		reconnectDelay = time.Second
		go func(s *yamux.Stream) {
			defer s.Close()
			started := time.Now()
			outcome := telemetry.Unavailable
			defer func() { c.observer.Operation(telemetry.Forward, started, outcome) }()

			conn, err := net.DialTimeout("tcp", localAddr, 5*time.Second)
			if err != nil {
				return
			}
			defer conn.Close()

			done := make(chan struct{}, 2)
			var responseBytes int64
			go func() {
				n, _ := io.Copy(conn, s)
				atomic.AddInt64(&tc.totalSize, n)
				conn.Close()
				done <- struct{}{}
			}()
			go func() {
				n, _ := io.Copy(s, conn)
				responseBytes = n
				atomic.AddInt64(&tc.totalSize, n)
				s.Close()
				done <- struct{}{}
			}()
			<-done
			<-done
			if responseBytes > 0 {
				outcome = telemetry.Success
				tc.forwarded.Do(func() { c.observer.Track(telemetry.TunnelForwarded) })
			}
		}(stream)
	}
}

func (c *Client) CloseTunnel(port uint16) error {
	c.mu.RLock()
	tc := c.tunnels[port]
	id := ""
	if tc != nil {
		id = tc.ID
	}
	c.mu.RUnlock()
	if tc == nil {
		return nil
	}
	if err := c.DeleteTunnel(id); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tunnels[port] == tc {
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
