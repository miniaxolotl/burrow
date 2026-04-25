package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
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
	// Drain leftover bytes from the previous WebSocket message before reading a new one.
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

const tunnelGracePeriod = 20 * time.Second
const maxLogsPerTunnel = 200

type Server struct {
	registry       *TunnelRegistry
	upgrader       websocket.Upgrader
	mux            *http.ServeMux
	domain         string
	secret         string
	secure         bool
	startTime      time.Time
	logger         *Logger
	mu             sync.RWMutex
	connections    map[string]*yamux.Session
	pendingRemoves map[string]chan struct{}
	httpServer     *http.Server
	closeOnce      sync.Once
	logMu          sync.RWMutex
	logs           map[string][]*protocol.TunnelLog
}

func NewServer(registry *TunnelRegistry, domain, secret string, secure bool, logger *Logger) *Server {
	s := &Server{
		registry:    registry,
		domain:      domain,
		secret:      secret,
		secure:      secure,
		startTime:   time.Now(),
		logger:      logger,
		connections:    make(map[string]*yamux.Session),
		pendingRemoves: make(map[string]chan struct{}),
		logs:           make(map[string][]*protocol.TunnelLog),
	}

	// CheckOrigin is intentionally permissive: clients connect from arbitrary
	// locations and browser-based CSRF is not a concern for a tunnel service.
	s.upgrader = websocket.Upgrader{
		CheckOrigin:     func(r *http.Request) bool { return true },
		ReadBufferSize:  32 * 1024,
		WriteBufferSize: 32 * 1024,
	}

	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/", s.handleRoot)
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/tunnels", s.handleTunnelList)
	s.mux.HandleFunc("/tunnel/", s.handleTunnel)
	s.mux.HandleFunc("/logs/", s.handleLogs)

	return s
}

// ServeHTTP dispatches tunnel-subdomain traffic to the proxy handler before
// any path-based routing. This prevents admin endpoints from intercepting
// requests destined for a client's local service.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if ParseTunnelID(r.Host, s.domain) != "" {
		s.handleTCP(w, r)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) auth(r *http.Request) bool {
	token := r.Header.Get("X-Tunnel-Token")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	return protocol.ValidateToken(token, s.secret)
}

func (s *Server) addLog(tunnelID string, entry *protocol.TunnelLog) {
	s.logMu.Lock()
	defer s.logMu.Unlock()
	entries := s.logs[tunnelID]
	if len(entries) >= maxLogsPerTunnel {
		entries = entries[1:]
	}
	s.logs[tunnelID] = append(entries, entry)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.auth(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tunnelID := strings.TrimPrefix(r.URL.Path, "/logs/")
	if tunnelID == "" {
		http.Error(w, "Tunnel ID required", http.StatusBadRequest)
		return
	}
	s.logMu.RLock()
	entries := s.logs[tunnelID]
	s.logMu.RUnlock()
	if entries == nil {
		entries = []*protocol.TunnelLog{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "404 not found")
		return
	}
	scheme := "http"
	if s.secure {
		scheme = "https"
	}
	fmt.Fprintf(w, "burrow - network tunneling service\n%s://{tunnel-id}.%s\n", scheme, s.domain)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	tunnels := len(s.connections)
	s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","tunnels":%d,"uptime":"%s"}`,
		tunnels, time.Since(s.startTime).Round(time.Second))
}

func (s *Server) handleTunnelList(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.auth(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tunnels, err := s.registry.List(r.Context())
	if err != nil {
		http.Error(w, "Failed to list tunnels", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tunnels)
}

func (s *Server) handleTunnel(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET" && r.URL.Path == "/tunnel/ws":
		s.handleWebSocket(w, r)
	case r.Method == "DELETE":
		s.handleTunnelDelete(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTunnelDelete(w http.ResponseWriter, r *http.Request) {
	if !s.auth(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tunnelID := strings.TrimPrefix(r.URL.Path, "/tunnel/")
	if tunnelID == "" {
		http.Error(w, "Tunnel ID required", http.StatusBadRequest)
		return
	}
	if err := s.registry.Remove(tunnelID); err != nil {
		http.Error(w, "Failed to remove tunnel", http.StatusInternalServerError)
		return
	}

	// Close the live yamux session so the client is notified immediately.
	s.mu.Lock()
	if session, ok := s.connections[tunnelID]; ok {
		session.Close()
		delete(s.connections, tunnelID)
	}
	s.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !s.auth(r) {
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
		http.Error(w, "port required", http.StatusBadRequest)
		return
	}

	// Cancel any pending grace-period removal — the client is reconnecting.
	s.mu.Lock()
	if cancel, ok := s.pendingRemoves[tunnelID]; ok {
		close(cancel)
		delete(s.pendingRemoves, tunnelID)
		s.logger.Infof("Tunnel %s reconnected, grace period cancelled", tunnelID)
	}
	s.mu.Unlock()

	scheme := "http"
	if s.secure {
		scheme = "https"
	}
	tunnelURL := fmt.Sprintf("%s://%s.%s", scheme, tunnelID, s.domain)
	conn, err := s.upgrader.Upgrade(w, r, http.Header{"X-Tunnel-URL": []string{tunnelURL}})
	if err != nil {
		s.logger.Errorf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	cfg := yamux.DefaultConfig()
	cfg.KeepAliveInterval = 30 * time.Second
	session, err := yamux.Server(&wsConn{Conn: conn}, cfg)
	if err != nil {
		s.logger.Errorf("Yamux server failed: %v", err)
		return
	}
	defer session.Close()

	if _, err := s.registry.Register(r.Context(), tunnelID, uint16(port)); err != nil {
		s.logger.Errorf("Failed to register tunnel %s: %v", tunnelID, err)
		return
	}

	// On disconnect, keep the registry entry for tunnelGracePeriod before removing.
	// This allows the client to reconnect and restore the tunnel without a URL change.
	// Defers run LIFO; keepalive (registered later) stops first, so no Register call
	// can race with the grace-period Remove.
	defer func() {
		cancel := make(chan struct{})
		s.mu.Lock()
		s.pendingRemoves[tunnelID] = cancel
		s.mu.Unlock()
		go func() {
			select {
			case <-time.After(tunnelGracePeriod):
				s.mu.Lock()
				delete(s.pendingRemoves, tunnelID)
				s.mu.Unlock()
				s.registry.Remove(tunnelID)
				s.logger.Infof("Tunnel %s grace period expired", tunnelID)
			case <-cancel:
				// Cancelled by reconnect or shutdown.
			}
		}()
	}()

	s.mu.Lock()
	s.connections[tunnelID] = session
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.connections, tunnelID)
		s.mu.Unlock()
	}()

	s.logger.Infof("Tunnel %s connected (port %d)", tunnelID, port)

	// Keep the Redis TTL alive for as long as the session is open.
	done := make(chan struct{})
	var keepaliveWg sync.WaitGroup
	keepaliveWg.Add(1)
	go func() {
		defer keepaliveWg.Done()
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.registry.Register(context.Background(), tunnelID, uint16(port))
			case <-done:
				return
			}
		}
	}()
	defer func() {
		close(done)
		keepaliveWg.Wait()
	}()

	// Block until session closes. Client-initiated streams are unexpected.
	for {
		stream, err := session.AcceptStream()
		if err != nil {
			s.logger.Infof("Tunnel %s disconnected: %v", tunnelID, err)
			return
		}
		stream.Close()
	}
}

// handleTCP proxies an inbound request through the yamux tunnel to the client's
// local service. Both WebSocket and plain HTTP are handled by writing the full
// HTTP request into the tunnel stream and then relaying raw bytes bidirectionally.
func (s *Server) handleTCP(w http.ResponseWriter, r *http.Request) {
	tunnelID := ParseTunnelID(r.Host, s.domain)
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

	stream, err := session.Open()
	if err != nil {
		http.Error(w, "Failed to open tunnel stream", http.StatusBadGateway)
		return
	}
	defer stream.Close()

	start := time.Now()

	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		size := s.proxyWebSocket(w, r, stream)
		s.addLog(tunnelID, &protocol.TunnelLog{
			Timestamp: start,
			Method:    "WS",
			Path:      r.URL.Path,
			Size:      size,
			Duration:  time.Since(start).Round(time.Millisecond).String(),
		})
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijack not supported", http.StatusInternalServerError)
		return
	}
	client, brw, err := hj.Hijack()
	if err != nil {
		s.logger.Errorf("Hijack failed: %v", err)
		return
	}
	defer client.Close()

	if err := r.Write(stream); err != nil {
		s.logger.Errorf("Failed to forward request: %v", err)
		return
	}

	if brw.Reader.Buffered() > 0 {
		buf := make([]byte, brw.Reader.Buffered())
		brw.Reader.Read(buf)
		stream.Write(buf)
	}

	var respSize int64
	done := make(chan struct{}, 2)
	go func() { io.Copy(stream, client); done <- struct{}{} }()
	go func() { n, _ := io.Copy(client, stream); respSize = n; client.Close(); done <- struct{}{} }()
	<-done
	<-done

	s.addLog(tunnelID, &protocol.TunnelLog{
		Timestamp: start,
		Method:    r.Method,
		Path:      r.URL.Path,
		Size:      respSize,
		Duration:  time.Since(start).Round(time.Millisecond).String(),
	})
}

// proxyWebSocket hijacks the inbound connection and relays the WebSocket
// upgrade and all subsequent frames through the tunnel stream.
func (s *Server) proxyWebSocket(w http.ResponseWriter, r *http.Request, stream net.Conn) int64 {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "WebSocket proxy not supported", http.StatusInternalServerError)
		return 0
	}
	client, brw, err := hj.Hijack()
	if err != nil {
		s.logger.Errorf("WebSocket hijack failed: %v", err)
		return 0
	}
	defer client.Close()

	if err := r.Write(stream); err != nil {
		s.logger.Errorf("Failed to forward WebSocket handshake: %v", err)
		return 0
	}

	if brw.Reader.Buffered() > 0 {
		buf := make([]byte, brw.Reader.Buffered())
		brw.Reader.Read(buf)
		stream.Write(buf)
	}

	var size int64
	done := make(chan struct{}, 2)
	go func() { io.Copy(stream, client); stream.Close(); done <- struct{}{} }()
	go func() { n, _ := io.Copy(client, stream); size = n; client.Close(); done <- struct{}{} }()
	<-done
	<-done
	return size
}

func (s *Server) Start(addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
	}
	s.mu.Lock()
	s.httpServer = srv
	s.mu.Unlock()
	s.logger.Infof("Starting server on %s", addr)
	return srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	var err error
	s.closeOnce.Do(func() {
		s.mu.Lock()
		for id, session := range s.connections {
			session.Close()
			delete(s.connections, id)
		}
		for _, cancel := range s.pendingRemoves {
			close(cancel)
		}
		s.pendingRemoves = make(map[string]chan struct{})
		srv := s.httpServer
		s.mu.Unlock()
		if srv != nil {
			err = srv.Shutdown(ctx)
		}
	})
	return err
}
