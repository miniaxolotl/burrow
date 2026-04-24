package internal

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

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

type Server struct {
	registry    *TunnelRegistry
	redis       *RedisClient
	upgrader    websocket.Upgrader
	mux         *http.ServeMux
	hostname    string
	domain      string
	secret      string
	startTime   time.Time
	mu          sync.RWMutex
	connections map[string]*yamux.Session
	httpServer  *http.Server
}

func NewServer(registry *TunnelRegistry, redis *RedisClient, hostname, domain, secret string) *Server {
	s := &Server{
		registry:    registry,
		redis:       redis,
		hostname:    hostname,
		domain:      domain,
		secret:      secret,
		startTime:   time.Now(),
		connections: make(map[string]*yamux.Session),
	}

	// CheckOrigin is intentionally permissive: tunnel clients connect from
	// arbitrary hosts and browser-based CSRF is not a concern here.
	s.upgrader = websocket.Upgrader{
		CheckOrigin:     func(r *http.Request) bool { return true },
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/tunnel/", s.handleTunnel)
	s.mux.HandleFunc("/", s.handleHTTP)

	return s
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	tunnels := len(s.connections)
	s.mu.RUnlock()

	uptime := time.Since(s.startTime).Round(time.Second)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","tunnels":%d,"uptime":"%s"}`, tunnels, uptime)
}

func (s *Server) handleTunnel(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" && r.URL.Path == "/tunnel/ws" {
		s.handleWebSocket(w, r)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := r.Header.Get("X-Tunnel-Token")
	if token == "" {
		token = r.URL.Query().Get("token")
	}

	if !ValidateToken(token, s.secret) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tunnelID := strings.TrimPrefix(r.URL.Path, "/tunnel/")
	if tunnelID == "" || tunnelID == "-" {
		var err error
		tunnelID, err = s.registry.HandleWildcardTunnelID(r.Context())
		if err != nil {
			http.Error(w, "Failed to generate tunnel ID", http.StatusInternalServerError)
			return
		}
	}

	portStr := r.URL.Query().Get("port")
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil || port == 0 {
		http.Error(w, "Valid port required: specify ?port=N", http.StatusBadRequest)
		return
	}

	url, err := s.registry.Register(r.Context(), tunnelID, uint16(port))
	if err != nil {
		http.Error(w, "Failed to register tunnel", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"tunnel_id":"%s","url":"%s"}`, tunnelID, url)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Tunnel-Token")
	if token == "" {
		token = r.URL.Query().Get("token")
	}

	if !ValidateToken(token, s.secret) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tunnelID := r.URL.Query().Get("tunnel_id")
	if tunnelID == "" {
		http.Error(w, "Tunnel ID required", http.StatusBadRequest)
		return
	}

	portStr := r.URL.Query().Get("port")
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil || port == 0 {
		http.Error(w, "Valid port required: specify ?port=N", http.StatusBadRequest)
		return
	}

	tunnelURL := fmt.Sprintf("https://%s.%s", tunnelID, s.domain)
	responseHeader := http.Header{"X-Tunnel-URL": []string{tunnelURL}}

	conn, err := s.upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	wrappedConn := &wsConn{Conn: conn}
	session, err := yamux.Server(wrappedConn, nil)
	if err != nil {
		log.Printf("Yamux server failed: %v", err)
		return
	}
	defer session.Close()

	if _, err := s.registry.Register(r.Context(), tunnelID, uint16(port)); err != nil {
		log.Printf("Failed to register tunnel %s: %v", tunnelID, err)
		return
	}
	defer s.registry.Remove(tunnelID)

	s.mu.Lock()
	s.connections[tunnelID] = session
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.connections, tunnelID)
		s.mu.Unlock()
	}()

	log.Printf("Tunnel %s connected (port %d)", tunnelID, port)

	// Block until the session closes. The client should not open streams;
	// close any unexpected client-initiated streams immediately.
	for {
		stream, err := session.AcceptStream()
		if err != nil {
			log.Printf("Tunnel %s disconnected: %v", tunnelID, err)
			return
		}
		stream.Close()
	}
}

// handleHTTP routes an incoming HTTP request through the yamux tunnel to the
// client's local service, then relays the response back to the caller.
func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	tunnelID := ParseTunnelID(host)

	if tunnelID == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	s.mu.RLock()
	session, ok := s.connections[tunnelID]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, "Tunnel not connected", http.StatusServiceUnavailable)
		return
	}

	conn, err := session.Open()
	if err != nil {
		http.Error(w, "Failed to open tunnel stream", http.StatusBadGateway)
		return
	}
	defer conn.Close()

	if err := r.Write(conn); err != nil {
		http.Error(w, "Failed to forward request", http.StatusBadGateway)
		return
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), r)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (s *Server) Start(addr string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	s.mu.Lock()
	s.httpServer = srv
	s.mu.Unlock()

	log.Printf("Starting server on %s", addr)
	return srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.RLock()
	srv := s.httpServer
	s.mu.RUnlock()
	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}
