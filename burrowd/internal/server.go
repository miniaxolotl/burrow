package internal

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"sort"
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

var validTunnelID = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

type tunnelSession struct {
	conn      *yamux.Session
	ownerHash string
	id        string
	port      uint16
}

type pendingRemoval struct {
	timer *time.Timer
}

type Server struct {
	registry *TunnelRegistry
	upgrader websocket.Upgrader
	mux      *http.ServeMux
	domain   string
	secure   bool
	mu       sync.RWMutex
	// Serialize registry changes with connection replacement and cleanup. Proxy
	// traffic uses mu only and never waits on Redis while holding that lock.
	lifecycleMu    sync.Mutex
	connections    map[string]*tunnelSession
	pendingRemoves map[string]*pendingRemoval
	gracePeriod    time.Duration
	closing        bool
	httpServer     *http.Server
	closeOnce      sync.Once
	logs           map[string][]*protocol.TunnelLog
	totalSize      map[string]int64
}

func NewServer(registry *TunnelRegistry, domain string, secure bool) *Server {
	s := &Server{
		registry:       registry,
		domain:         domain,
		secure:         secure,
		connections:    make(map[string]*tunnelSession),
		pendingRemoves: make(map[string]*pendingRemoval),
		gracePeriod:    tunnelGracePeriod,
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
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /tunnels", s.handleTunnelList)
	s.mux.HandleFunc("GET /tunnels/connect", s.handleWebSocket)
	s.mux.HandleFunc("GET /tunnels/{id}/logs", s.handleLogs)
	s.mux.HandleFunc("DELETE /tunnels/{id}", s.handleTunnelDelete)

	return s
}

// ServeHTTP dispatches tunnel-subdomain traffic to the proxy handler before
// any path-based routing. This prevents management endpoints from intercepting
// requests destined for a client's local service.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if ParseTunnelID(r.Host, s.domain) != "" {
		s.handleTCP(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	s.mux.ServeHTTP(w, r)
}

func requestOwner(w http.ResponseWriter, r *http.Request) string {
	token := r.Header.Get("X-Tunnel-Token")
	if token == "" {
		jsonError(w, "Owner credential required; update burrowctl", http.StatusUnauthorized)
		return ""
	}
	return hashToken(token)
}

// Legacy tokenless tunnels fail closed; a public URL is never a credential.
func (s *Server) authForTunnel(r *http.Request, tunnelID, ownerHash string) bool {
	td, err := s.registry.Get(r.Context(), tunnelID)
	if err != nil || td == nil {
		return false
	}
	return td.OwnerHash != "" && ownerHash == td.OwnerHash
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

func (s *Server) addLog(tunnelID string, session *tunnelSession, entry *protocol.TunnelLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.connections[tunnelID] != session {
		return
	}
	entries := s.logs[tunnelID]
	if len(entries) >= maxLogsPerTunnel {
		entries = entries[1:]
	}
	s.logs[tunnelID] = append(entries, entry)
	s.totalSize[tunnelID] += entry.Size
}

// Caller holds mu.
func (s *Server) cleanupTunnel(tunnelID string) {
	delete(s.logs, tunnelID)
	delete(s.totalSize, tunnelID)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	ownerHash := requestOwner(w, r)
	if ownerHash == "" {
		return
	}
	tunnelID := r.PathValue("id")
	if !s.authForTunnel(r, tunnelID, ownerHash) {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	s.mu.RLock()
	entries := append([]*protocol.TunnelLog{}, s.logs[tunnelID]...)
	s.mu.RUnlock()
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
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok"}`)
}

func (s *Server) tunnelSnapshot(ownerHash string) []*protocol.TunnelInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tunnels := make([]*protocol.TunnelInfo, 0)
	for id, session := range s.connections {
		if session.ownerHash != ownerHash {
			continue
		}
		tunnels = append(tunnels, &protocol.TunnelInfo{
			TunnelID: id, Port: session.port, URL: s.registry.tunnelURL(id),
			Status: "active", TotalSize: s.totalSize[id],
		})
	}
	sort.Slice(tunnels, func(i, j int) bool { return tunnels[i].TunnelID < tunnels[j].TunnelID })
	return tunnels
}

func (s *Server) handleTunnelList(w http.ResponseWriter, r *http.Request) {
	ownerHash := requestOwner(w, r)
	if ownerHash == "" {
		return
	}
	if r.URL.Query().Has("all") {
		jsonError(w, "Global tunnel views are not available", http.StatusBadRequest)
		return
	}
	tunnels := s.tunnelSnapshot(ownerHash)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(protocol.TunnelList{Tunnels: tunnels, Count: len(tunnels)})
}

func (s *Server) handleTunnelDelete(w http.ResponseWriter, r *http.Request) {
	ownerHash := requestOwner(w, r)
	if ownerHash == "" {
		return
	}
	tunnelID := r.PathValue("id")
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if err := s.registry.Revoke(r.Context(), tunnelID, ownerHash); err != nil {
		if errors.Is(err, ErrTunnelOwnership) {
			jsonError(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		jsonError(w, "Failed to remove tunnel", http.StatusInternalServerError)
		return
	}

	// Close the live yamux session so the client is notified immediately.
	s.mu.Lock()
	if session, ok := s.connections[tunnelID]; ok {
		delete(s.connections, tunnelID)
		session.conn.Close()
	}
	if pending := s.pendingRemoves[tunnelID]; pending != nil {
		pending.timer.Stop()
		delete(s.pendingRemoves, tunnelID)
	}
	s.cleanupTunnel(tunnelID)
	s.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ownerHash := requestOwner(w, r)
	if ownerHash == "" {
		return
	}

	tunnelID := r.URL.Query().Get("tunnel_id")
	if !validTunnelID.MatchString(tunnelID) {
		jsonError(w, "Invalid tunnel ID", http.StatusBadRequest)
		return
	}

	portStr := r.URL.Query().Get("port")
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil || port == 0 {
		jsonError(w, "port required", http.StatusBadRequest)
		return
	}

	if !websocket.IsWebSocketUpgrade(r) {
		jsonError(w, "WebSocket upgrade required", http.StatusBadRequest)
		return
	}
	session := s.acceptTunnel(w, r, tunnelID, uint16(port), ownerHash)
	if session == nil {
		return
	}
	defer func() {
		session.conn.Close()
		s.disconnectTunnel(tunnelID, session)
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
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := s.registry.Refresh(ctx, tunnelID, session.id)
				cancel()
				if err != nil {
					session.conn.Close()
					return
				}
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
		stream, err := session.conn.AcceptStream()
		if err != nil {
			return
		}
		stream.Close()
	}
}

func (s *Server) acceptTunnel(w http.ResponseWriter, r *http.Request, tunnelID string, port uint16, ownerHash string) *tunnelSession {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.closing {
		jsonError(w, "Server shutting down", http.StatusServiceUnavailable)
		return nil
	}
	session := &tunnelSession{ownerHash: ownerHash, id: rand.Text(), port: port}
	tunnelURL, err := s.registry.Register(r.Context(), tunnelID, port, ownerHash, session.id)
	if err != nil {
		switch {
		case errors.Is(err, ErrTunnelOwnership):
			jsonError(w, "Tunnel ID is unavailable", http.StatusConflict)
		case errors.Is(err, ErrTunnelRevoked):
			jsonError(w, "Tunnel has been closed", http.StatusGone)
		default:
			jsonError(w, "Failed to register tunnel", http.StatusServiceUnavailable)
		}
		return nil
	}
	// A failed handshake must not remove a concurrently replaced generation.
	installed := false
	defer func() {
		if !installed {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s.registry.Remove(ctx, tunnelID, session.id)
		}
	}()
	conn, err := s.upgrader.Upgrade(w, r, http.Header{"X-Tunnel-URL": []string{tunnelURL}, "Cache-Control": []string{"no-store"}})
	if err != nil {
		return nil
	}
	conn.SetReadLimit(protocol.MaxMessageSize)
	cfg := yamux.DefaultConfig()
	cfg.KeepAliveInterval = 30 * time.Second
	session.conn, err = yamux.Server(&protocol.WsConn{Conn: conn}, cfg)
	if err != nil {
		conn.Close()
		return nil
	}
	s.mu.Lock()
	previous := s.connections[tunnelID]
	s.connections[tunnelID] = session
	if pending := s.pendingRemoves[tunnelID]; pending != nil {
		pending.timer.Stop()
		delete(s.pendingRemoves, tunnelID)
	}
	s.mu.Unlock()
	installed = true
	if previous != nil {
		previous.conn.Close()
	}
	return session
}

func (s *Server) disconnectTunnel(tunnelID string, session *tunnelSession) {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing || s.connections[tunnelID] != session {
		return
	}
	delete(s.connections, tunnelID)
	pending := &pendingRemoval{}
	s.pendingRemoves[tunnelID] = pending
	pending.timer = time.AfterFunc(s.gracePeriod, func() {
		s.expireTunnel(tunnelID, session.id, pending)
	})
}

func (s *Server) expireTunnel(tunnelID, sessionID string, pending *pendingRemoval) {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	s.mu.Lock()
	if s.pendingRemoves[tunnelID] != pending {
		s.mu.Unlock()
		return
	}
	delete(s.pendingRemoves, tunnelID)
	s.cleanupTunnel(tunnelID)
	s.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.registry.Remove(ctx, tunnelID, sessionID)
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

	stream, err := session.conn.Open()
	if err != nil {
		jsonError(w, "Failed to open tunnel stream", http.StatusBadGateway)
		return
	}
	defer stream.Close()

	start := time.Now()
	size, ok := s.proxyStream(w, r, stream)
	if !ok {
		return
	}
	method := r.Method
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		method = "WS"
	}
	s.addLog(tunnelID, session, &protocol.TunnelLog{
		Timestamp: start, Method: method, Path: r.URL.Path, Size: size,
		Duration: time.Since(start).Round(time.Millisecond).String(), IP: clientIP(r),
	})
}

// HTTP and WebSocket upgrades share the same request forwarding and byte relay.
func (s *Server) proxyStream(w http.ResponseWriter, r *http.Request, stream net.Conn) (int64, bool) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		return 0, false
	}
	// Finish reading the original request body before hijacking. Reading it
	// afterward starts net/http's background reader on a connection we own,
	// which can consume the first byte of the next keep-alive request.
	if err := r.Write(stream); err != nil {
		jsonError(w, "Failed to forward request", http.StatusBadGateway)
		return 0, false
	}
	client, brw, err := hj.Hijack()
	if err != nil {
		return 0, false
	}
	defer client.Close()

	var respSize int64
	done := make(chan struct{}, 2)
	// Keep using the returned reader: it can contain pipelined request bytes.
	go func() { io.Copy(stream, brw.Reader); stream.Close(); done <- struct{}{} }()
	go func() { n, _ := io.Copy(client, stream); respSize = n; client.Close(); done <- struct{}{} }()
	<-done
	<-done
	return respSize, true
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
		s.lifecycleMu.Lock()
		s.closing = true
		s.mu.Lock()
		for id, session := range s.connections {
			session.conn.Close()
			delete(s.connections, id)
		}
		for _, pending := range s.pendingRemoves {
			pending.timer.Stop()
		}
		s.pendingRemoves = make(map[string]*pendingRemoval)
		srv := s.httpServer
		s.mu.Unlock()
		s.lifecycleMu.Unlock()
		if srv != nil {
			err = srv.Shutdown(ctx)
		}
	})
	return err
}
