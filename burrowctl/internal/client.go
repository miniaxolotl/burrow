package internal

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
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
	server    string
	token     string
	domain    string
	tunnels   map[uint16]*TunnelConn
	mu        sync.RWMutex
	reconnect bool
}

type TunnelConn struct {
	ID        string
	Port      uint16
	URL       string
	wsConn    *websocket.Conn
	session   *yamux.Session
	localConn net.Conn
}

func NewClient(server, token, domain string) *Client {
	return &Client{
		server:  server,
		token:   token,
		domain:  domain,
		tunnels: make(map[uint16]*TunnelConn),
	}
}

func (c *Client) CreateTunnel(ctx context.Context, port uint16) (string, error) {
	tunnelID := generateTunnelID()

	// Send token as a header, not a query parameter, to avoid it appearing in logs.
	header := http.Header{"X-Tunnel-Token": []string{c.token}}
	wsURL := fmt.Sprintf("ws://%s/tunnel/ws?tunnel_id=%s&port=%d", c.server, tunnelID, port)

	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return "", fmt.Errorf("failed to connect to server: %w", err)
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
		return "", fmt.Errorf("failed to create yamux session: %w", err)
	}

	tc := &TunnelConn{
		ID:      tunnelID,
		Port:    port,
		URL:     tunnelURL,
		wsConn:  conn,
		session: session,
	}

	c.mu.Lock()
	c.tunnels[port] = tc
	c.mu.Unlock()

	go c.handleTunnel(tc, port)

	return tc.URL, nil
}

// handleTunnel accepts yamux streams opened by the server and pipes each one
// to the client's local service at the given port.
func (c *Client) handleTunnel(tc *TunnelConn, port uint16) {
	localAddr := fmt.Sprintf("localhost:%d", port)

	for {
		stream, err := tc.session.AcceptStream()
		if err != nil {
			log.Printf("Session closed for tunnel %s: %v", tc.ID, err)
			return
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
		}(stream)
	}
}

func (c *Client) CloseTunnel(port uint16) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if tc, ok := c.tunnels[port]; ok {
		if tc.session != nil {
			tc.session.Close()
		}
		if tc.wsConn != nil {
			tc.wsConn.Close()
		}
		if tc.localConn != nil {
			tc.localConn.Close()
		}
		delete(c.tunnels, port)
	}
	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, tc := range c.tunnels {
		if tc.session != nil {
			tc.session.Close()
		}
		if tc.wsConn != nil {
			tc.wsConn.Close()
		}
		if tc.localConn != nil {
			tc.localConn.Close()
		}
	}
	c.tunnels = make(map[uint16]*TunnelConn)
	return nil
}

func (c *Client) ListTunnels() []*TunnelInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tunnels := make([]*TunnelInfo, 0, len(c.tunnels))
	for port, tc := range c.tunnels {
		tunnels = append(tunnels, &TunnelInfo{
			TunnelID: tc.ID,
			Port:     port,
			URL:      tc.URL,
			Status:   "active",
		})
	}
	return tunnels
}

type TunnelInfo struct {
	TunnelID string
	Port     uint16
	URL      string
	Status   string
}

var tunnelIDAdjectives = []string{
	"arcane", "ancient", "astral", "bold", "brave", "chaotic", "cryptic", "dark", "elder",
	"ethereal", "fierce", "frozen", "hidden", "icy", "jade", "keen", "liquid", "mystic",
	"noble", "obscure", "potent", "quick", "radiant", "shadow", "swift", "twilight",
	"uncanny", "vivid", "wandering", "wild",
}

var tunnelIDNouns = []string{
	"amulet", "basilisk", "cipher", "dragon", "ember", "fortress", "gargoyle", "helm",
	"illusion", "kraken", "lich", "mithril", "nymph", "oracle", "phoenix", "quest",
	"rune", "specter", "talisman", "umbral", "void", "wyrm", "zephyr", "amethyst",
	"bramble", "crypt", "druid", "forge", "grimoire", "haven", "isle", "knave",
	"lava", "moon", "nexus", "obsidian", "prism", "quill", "shadow", "tome", "umbra",
	"vestige", "warden", "xorn", "zinc",
}

func generateTunnelID() string {
	adj := tunnelIDAdjectives[rand.Intn(len(tunnelIDAdjectives))]
	noun := tunnelIDNouns[rand.Intn(len(tunnelIDNouns))]
	return fmt.Sprintf("%s-%s-%d", adj, noun, rand.Intn(100))
}

// GenerateToken creates a time-stamped HMAC token delegating to the shared protocol package.
func GenerateToken(secret string) string      { return protocol.GenerateToken(secret) }
func ValidateToken(token, secret string) bool { return protocol.ValidateToken(token, secret) }
