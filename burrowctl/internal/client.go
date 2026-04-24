package internal

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"burrow/protocol"

	"github.com/gorilla/websocket"
	"github.com/xtaci/yamux"
)

type wsConn struct {
	*websocket.Conn
	buf []byte
}

func (c *wsConn) Read(b []byte) (int, error) {
	// Drain any leftover bytes from the previous message before reading a new one.
	if len(c.buf) > 0 {
		n := copy(b, c.buf)
		c.buf = c.buf[n:]
		return n, nil
	}
	msgType, msg, err := c.Conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	if msgType != websocket.BinaryMessage {
		return 0, fmt.Errorf("expected binary message")
	}
	n := copy(b, msg)
	if n < len(msg) {
		c.buf = msg[n:]
	}
	return n, nil
}

func (c *wsConn) Write(b []byte) (int, error) {
	if err := c.Conn.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return 0, err
	}
	return len(b), nil
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
	ID       string
	Port     uint16
	URL      string
	wsConn   *websocket.Conn
	session  *yamux.Session
	closed   bool
	closeMu  sync.Mutex
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

	c.mu.Lock()
	c.tunnels[port] = tc
	c.mu.Unlock()

	go c.handleTunnel(tc, port)

	return tc.URL, nil
}

func (c *Client) createTunnelSession(ctx context.Context, tunnelID string, port uint16) (*TunnelConn, error) {
	header := http.Header{"X-Tunnel-Token": []string{c.token}}
	scheme := "ws"
	if c.secure {
		scheme = "wss"
	}
	wsURL := fmt.Sprintf("%s://%s/tunnel/ws?tunnel_id=%s&port=%d", scheme, c.server, tunnelID, port)

	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	tunnelURL := ""
	if resp != nil {
		tunnelURL = resp.Header.Get("X-Tunnel-URL")
	}
	if tunnelURL == "" {
		tunnelURL = fmt.Sprintf("https://%s.%s", tunnelID, c.domain)
	}

	wrappedConn := &wsConn{Conn: conn}
	session, err := yamux.Client(wrappedConn, nil)
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
	}, nil
}

func (c *Client) handleTunnel(tc *TunnelConn, port uint16) {
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
			tc.closeMu.Lock()
			if tc.closed {
				tc.closeMu.Unlock()
				return
			}
			tc.closeMu.Unlock()

			log.Printf("Tunnel %s lost, reconnecting in %v...", tc.ID, reconnectDelay)
			time.Sleep(reconnectDelay)
			if reconnectDelay < 30*time.Second {
				reconnectDelay *= 2
			}

			newURL, err := c.createTunnelSession(context.Background(), tc.ID, port)
			if err != nil {
				log.Printf("Reconnect failed for tunnel %s: %v", tc.ID, err)
				continue
			}

			tc.wsConn, tc.session, tc.URL = newURL.wsConn, newURL.session, newURL.URL
			continue
		}

		go func(s *yamux.Stream) {
			defer s.Close()

			conn, err := net.DialTimeout("tcp", localAddr, 5*time.Second)
			if err != nil {
				log.Printf("Failed to dial %s: %v", localAddr, err)
				return
			}
			defer conn.Close()

			done := make(chan struct{}, 2)
			go func() { io.Copy(conn, s); conn.Close(); done <- struct{}{} }()
			go func() { io.Copy(s, conn); s.Close(); done <- struct{}{} }()
			<-done
			<-done
		}(stream)
	}
}

func (c *Client) CloseTunnel(port uint16) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if tc, ok := c.tunnels[port]; ok {
		tc.closeMu.Lock()
		tc.closed = true
		tc.closeMu.Unlock()
		if tc.session != nil {
			tc.session.Close()
		}
		if tc.wsConn != nil {
			tc.wsConn.Close()
		}
		delete(c.tunnels, port)
	}
	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, tc := range c.tunnels {
		tc.closeMu.Lock()
		tc.closed = true
		tc.closeMu.Unlock()
		if tc.session != nil {
			tc.session.Close()
		}
		if tc.wsConn != nil {
			tc.wsConn.Close()
		}
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
			TunnelID: tc.ID,
			Port:     port,
			URL:      tc.URL,
			Status:   "active",
		})
	}
	return tunnels
}
