package internal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/hashicorp/yamux"
)


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
	mu             sync.RWMutex
	connections    map[string]*yamux.Session
	pendingRemoves map[string]chan struct{}
	httpServer     *http.Server
	closeOnce      sync.Once
	logMu          sync.RWMutex
	logs           map[string][]*protocol.TunnelLog
	sizeMu         sync.RWMutex
	totalSize      map[string]int64
}

func NewServer(registry *TunnelRegistry, domain, secret string, secure bool) *Server {
	s := &Server{
		registry:    registry,
		domain:      domain,
		secret:      secret,
		secure:      secure,
		startTime:   time.Now(),
		connections:    make(map[string]*yamux.Session),
		pendingRemoves: make(map[string]chan struct{}),
		logs:           make(map[string][]*protocol.TunnelLog),
		totalSize:      make(map[string]int64),
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

func (s *Server) tokenFromRequest(r *http.Request) string {
	if t := r.Header.Get("X-Tunnel-Token"); t != "" {
		return t
	}
	// WARNING: Accepting token from query params is insecure as tokens may
	// appear in server access logs. Prefer X-Tunnel-Token header.
	return r.URL.Query().Get("token")
}

// authTunnel: tunnel creation is always open — no authentication required.
func (s *Server) authTunnel(_ *http.Request) bool {
	return true
}

// authAdmin always requires a valid token; returns false when no secret is set.
func (s *Server) authAdmin(r *http.Request) bool {
	if s.secret == "" {
		return false
	}
	return protocol.ValidateToken(s.tokenFromRequest(r), s.secret)
}

// authForTunnel allows admins or the tunnel's owner. Anonymous tunnels (no owner)
// are accessible without auth.
func (s *Server) authForTunnel(r *http.Request, tunnelID string) bool {
	if s.authAdmin(r) {
		return true
	}
	td, err := s.registry.Get(r.Context(), tunnelID)
	if err != nil || td == nil {
		return false
	}
	if td.OwnerHash == "" {
		return true
	}
	token := s.tokenFromRequest(r)
	return token != "" && hashToken(token) == td.OwnerHash
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func clientIP(r *http.Request) string {
	// X-Real-IP is set by nginx directly to $remote_addr — it's the actual
	// public IP and cannot be spoofed by the downstream client.
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// Fallback: use the last entry nginx appended via $proxy_add_x_forwarded_for.
	// The last entry is always the IP of the most recent trusted proxy, not a
	// client-controlled value.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *Server) addLog(tunnelID string, entry *protocol.TunnelLog) {
	s.logMu.Lock()
	entries := s.logs[tunnelID]
	if len(entries) >= maxLogsPerTunnel {
		entries = entries[1:]
	}
	s.logs[tunnelID] = append(entries, entry)
	s.logMu.Unlock()

	s.sizeMu.Lock()
	s.totalSize[tunnelID] += entry.Size
	s.sizeMu.Unlock()
}

func (s *Server) cleanupTunnel(tunnelID string) {
	s.logMu.Lock()
	delete(s.logs, tunnelID)
	s.logMu.Unlock()

	s.sizeMu.Lock()
	delete(s.totalSize, tunnelID)
	s.sizeMu.Unlock()
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tunnelID := strings.TrimPrefix(r.URL.Path, "/logs/")
	if tunnelID == "" {
		jsonError(w, "Tunnel ID required", http.StatusBadRequest)
		return
	}
	if !s.authForTunnel(r, tunnelID) {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
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
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.authAdmin(r) {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tunnels, err := s.registry.List(r.Context())
	if err != nil {
		jsonError(w, "Failed to list tunnels", http.StatusInternalServerError)
		return
	}
	s.sizeMu.RLock()
	for _, t := range tunnels {
		t.TotalSize = s.totalSize[t.TunnelID]
	}
	s.sizeMu.RUnlock()
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
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTunnelDelete(w http.ResponseWriter, r *http.Request) {
	tunnelID := strings.TrimPrefix(r.URL.Path, "/tunnel/")
	if tunnelID == "" {
		jsonError(w, "Tunnel ID required", http.StatusBadRequest)
		return
	}
	if !s.authForTunnel(r, tunnelID) {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if err := s.registry.Remove(tunnelID); err != nil {
		jsonError(w, "Failed to remove tunnel", http.StatusInternalServerError)
		return
	}

	// Close the live yamux session so the client is notified immediately.
	s.mu.Lock()
	if session, ok := s.connections[tunnelID]; ok {
		session.Close()
		delete(s.connections, tunnelID)
	}
	s.mu.Unlock()

	s.cleanupTunnel(tunnelID)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !s.authTunnel(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tunnelID := r.URL.Query().Get("tunnel_id")
	if tunnelID == "" {
		jsonError(w, "Tunnel ID required", http.StatusBadRequest)
		return
	}

	portStr := r.URL.Query().Get("port")
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil || port == 0 {
		jsonError(w, "port required", http.StatusBadRequest)
		return
	}

	ownerHash := ""
	if token := s.tokenFromRequest(r); token != "" {
		ownerHash = hashToken(token)
	}

	// Cancel any pending grace-period removal — the client is reconnecting.
	s.mu.Lock()
	if cancel, ok := s.pendingRemoves[tunnelID]; ok {
		close(cancel)
		delete(s.pendingRemoves, tunnelID)
	}
	s.mu.Unlock()

	scheme := "http"
	if s.secure {
		scheme = "https"
	}
	tunnelURL := fmt.Sprintf("%s://%s.%s", scheme, tunnelID, s.domain)
	conn, err := s.upgrader.Upgrade(w, r, http.Header{"X-Tunnel-URL": []string{tunnelURL}})
	if err != nil {
		return
	}
	defer conn.Close()

	cfg := yamux.DefaultConfig()
	cfg.KeepAliveInterval = 30 * time.Second
	session, err := yamux.Server(&protocol.WsConn{Conn: conn}, cfg)
	if err != nil {
		return
	}
	defer session.Close()

	if _, err := s.registry.Register(r.Context(), tunnelID, uint16(port), ownerHash); err != nil {
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
				s.cleanupTunnel(tunnelID)
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

	// Keep the Redis TTL alive for as long as the session is open.
	done := make(chan struct{})
	var keepaliveWg sync.WaitGroup
	keepaliveWg.Go(func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.registry.Register(context.Background(), tunnelID, uint16(port), ownerHash)
			case <-done:
				return
			}
		}
	})
	defer func() {
		close(done)
		keepaliveWg.Wait()
	}()

	// Block until session closes. Client-initiated streams are unexpected.
	for {
		stream, err := session.AcceptStream()
		if err != nil {
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
		jsonError(w, "Tunnel not connected", http.StatusServiceUnavailable)
		return
	}

	stream, err := session.Open()
	if err != nil {
		jsonError(w, "Failed to open tunnel stream", http.StatusBadGateway)
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
			IP:        clientIP(r),
		})
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		return
	}
	client, brw, err := hj.Hijack()
	if err != nil {
		return
	}

	if err := r.Write(stream); err != nil {
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
		IP:        clientIP(r),
	})
}

// proxyWebSocket hijacks the inbound connection and relays the WebSocket
// upgrade and all subsequent frames through the tunnel stream.
func (s *Server) proxyWebSocket(w http.ResponseWriter, r *http.Request, stream net.Conn) int64 {
	hj, ok := w.(http.Hijacker)
	if !ok {
		return 0
	}
	client, brw, err := hj.Hijack()
	if err != nil {
		return 0
	}
	defer client.Close()

	if err := r.Write(stream); err != nil {
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
