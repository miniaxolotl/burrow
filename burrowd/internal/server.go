package internal

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"burrow/protocol"
	"burrow/protocol/telemetry"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

const tunnelGracePeriod = 20 * time.Second
const maxLogsPerTunnel = 200

var validTunnelID = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

type tunnelSession struct {
	conn           *yamux.Session
	ownerHash      string
	id             string
	port           uint16
	lifecycle      *yamux.Stream
	lifecycleReady chan struct{}
	tunnelID       string
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
	shutdownErr    error
	drainTimeout   time.Duration
	requests       sync.WaitGroup
	handlers       sync.WaitGroup
	requestCancels map[*http.Request]context.CancelFunc
	intake         context.Context
	stopIntake     context.CancelFunc
	readyMu        sync.Mutex
	readyFlight    chan struct{}
	readyAt        time.Time
	readyOK        bool
	logs           map[string][]*protocol.TunnelLog
	totalSize      map[string]int64
	observer       telemetry.Observer
}

func NewServer(registry *TunnelRegistry, domain string, secure bool, observers ...telemetry.Observer) *Server {
	intake, stopIntake := context.WithCancel(context.Background())
	var observer telemetry.Observer = (*telemetry.Reporter)(nil)
	if len(observers) > 0 && observers[0] != nil {
		observer = observers[0]
	}
	s := &Server{
		registry:       registry,
		domain:         domain,
		secure:         secure,
		connections:    make(map[string]*tunnelSession),
		pendingRemoves: make(map[string]*pendingRemoval),
		gracePeriod:    tunnelGracePeriod,
		logs:           make(map[string][]*protocol.TunnelLog),
		totalSize:      make(map[string]int64),
		observer:       observer,
		drainTimeout:   15 * time.Second,
		requestCancels: make(map[*http.Request]context.CancelFunc),
		intake:         intake, stopIntake: stopIntake,
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
	s.mux.HandleFunc("GET /ready", s.handleReady)
	s.mux.HandleFunc("GET "+protocol.CapabilityPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(protocol.Capabilities{Version: 1, Registration: 1, PersistentAttachment: 1, Lifecycle: 1})
	})
	s.mux.HandleFunc("GET /tunnels/{id}/reservation", s.handleReservation)
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
	started := time.Now()
	defer func() {
		if failure := recover(); failure != nil {
			s.observer.Operation(telemetry.Serve, started, telemetry.Failure)
			panic(failure)
		}
	}()
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
	// Request start times can arrive out of order when forwarding completes.
	// Opaque entry IDs allow bounded deduplication without exposing global counts.
	entry.ID = rand.Text()
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
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		drainRejected(w)
		return
	}
	s.handlers.Add(1)
	s.mu.Unlock()
	defer s.handlers.Done()
	if version := r.Header.Get(protocol.LifecycleHeader); version != "" && (version != "1" || r.Header.Get(protocol.RegistrationHeader) != "1") {
		jsonError(w, "Lifecycle requires registration v1 and lifecycle v1", http.StatusBadRequest)
		return
	}
	if version := r.Header.Get(protocol.RegistrationHeader); version != "" && version != protocol.RegistrationVersion {
		jsonError(w, "Unsupported registration version", http.StatusBadRequest)
		return
	}
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
	started := time.Now()
	outcome := telemetry.Unavailable
	defer func() { s.observer.Operation(telemetry.Connect, started, outcome) }()
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	s.mu.RLock()
	closing := s.closing
	s.mu.RUnlock()
	if closing {
		drainRejected(w)
		return nil
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	stop := context.AfterFunc(s.intake, cancel)
	defer stop()
	r = r.WithContext(ctx)
	session := &tunnelSession{ownerHash: ownerHash, id: rand.Text(), port: port, tunnelID: tunnelID, lifecycleReady: make(chan struct{})}
	defer close(session.lifecycleReady)
	persistent := r.Header.Get(protocol.AttachmentHeader) != ""
	if persistent {
		if r.Header.Get(protocol.AttachmentHeader) != "1" || r.Header.Get(protocol.RegistrationHeader) != "1" || !validAttachment(r.Header.Get(protocol.AttachmentIDHeader), false) || !validAttachment(r.Header.Get(protocol.AttemptHeader), false) || !validAttachment(r.Header.Get(protocol.ExpectedGenerationHeader), true) {
			jsonError(w, "Invalid attachment negotiation", http.StatusBadRequest)
			return nil
		}
		s.mu.RLock()
		live := s.connections[tunnelID]
		s.mu.RUnlock()
		if live != nil && !live.conn.IsClosed() {
			if live.ownerHash != ownerHash {
				jsonError(w, "Tunnel ID is unavailable", http.StatusConflict)
			} else {
				jsonError(w, "Tunnel attachment busy", http.StatusLocked)
			}
			return nil
		}
	}
	var tunnelURL string
	var err error
	if persistent {
		err = s.registry.redis.ReservePersistent(r.Context(), tunnelID, ownerHash)
		tunnelURL = s.registry.tunnelURL(tunnelID)
	} else {
		tunnelURL, err = s.registry.Reserve(r.Context(), tunnelID, ownerHash)
	}
	if err != nil {
		switch {
		case errors.Is(err, ErrTunnelOwnership), errors.Is(err, ErrPersistentRequired), errors.Is(err, ErrStaleAttachment):
			jsonError(w, "Tunnel ID is unavailable", http.StatusConflict)
		case errors.Is(err, ErrTunnelRevoked):
			jsonError(w, "Tunnel has been closed", http.StatusGone)
		default:
			outcome = telemetry.Failure
			jsonError(w, "Failed to register tunnel", http.StatusServiceUnavailable)
		}
		return nil
	}
	if persistent {
		if err := s.registry.redis.CheckPersistent(r.Context(), tunnelID, ownerHash, r.Header.Get(protocol.AttachmentIDHeader), r.Header.Get(protocol.ExpectedGenerationHeader), r.Header.Get(protocol.AttemptHeader)); err != nil {
			status := http.StatusServiceUnavailable
			if errors.Is(err, ErrStaleAttachment) || errors.Is(err, ErrTunnelOwnership) || errors.Is(err, ErrPersistentRequired) {
				status = http.StatusConflict
			}
			if errors.Is(err, ErrTunnelRevoked) {
				status = http.StatusGone
			}
			jsonError(w, "Attachment admission rejected", status)
			return nil
		}
	}
	headers := http.Header{"X-Tunnel-URL": {tunnelURL}, "Cache-Control": {"no-store"}}
	registration := r.Header.Get(protocol.RegistrationHeader) == protocol.RegistrationVersion
	if registration {
		headers.Set(protocol.RegistrationHeader, protocol.RegistrationVersion)
	}
	if r.Header.Get(protocol.LifecycleHeader) == "1" {
		headers.Set(protocol.LifecycleHeader, "1")
	}
	conn, err := s.upgrader.Upgrade(w, r, headers)
	if err != nil {
		return nil
	}
	stopHandshake := context.AfterFunc(ctx, func() { conn.Close() })
	defer stopHandshake()
	conn.SetReadLimit(protocol.MaxMessageSize)
	cfg := yamux.DefaultConfig()
	cfg.KeepAliveInterval = 30 * time.Second
	if persistent {
		cfg.KeepAliveInterval = 10 * time.Second
		cfg.ConnectionWriteTimeout = 20 * time.Second
	}
	session.conn, err = yamux.Server(&protocol.WsConn{Conn: conn}, cfg)
	if err != nil {
		conn.Close()
		return nil
	}
	var control *yamux.Stream
	if registration {
		control, err = session.conn.OpenStream()
		if err != nil {
			session.conn.Close()
			return nil
		}
		defer control.Close()
		control.SetWriteDeadline(time.Now().Add(5 * time.Second))
	}
	if headers.Get(protocol.LifecycleHeader) == "1" {
		session.lifecycle, err = session.conn.OpenStream()
		if err != nil {
			session.conn.Close()
			return nil
		}
		session.lifecycle.SetWriteDeadline(time.Now().Add(5 * time.Second))
	}
	// Commit the new generation only after the handshake succeeds. A failed
	// upgrade cannot invalidate the previous connection or its concurrent refresh.
	if persistent {
		session.id, err = s.registry.redis.SetPersistent(r.Context(), tunnelID, port, ownerHash, r.Header.Get(protocol.AttachmentIDHeader), r.Header.Get(protocol.ExpectedGenerationHeader), r.Header.Get(protocol.AttemptHeader), session.id)
	} else {
		_, err = s.registry.Register(r.Context(), tunnelID, port, ownerHash, session.id)
	}
	if err != nil {
		outcome = telemetry.Failure
		session.conn.Close()
		return nil
	}
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		session.conn.Close()
		return nil
	}
	previous := s.connections[tunnelID]
	s.connections[tunnelID] = session
	if pending := s.pendingRemoves[tunnelID]; pending != nil {
		pending.timer.Stop()
		delete(s.pendingRemoves, tunnelID)
	}
	s.mu.Unlock()
	if previous != nil {
		previous.conn.Close()
	}
	if control != nil {
		if err := json.NewEncoder(control).Encode(protocol.Registration{Version: 1, TunnelID: tunnelID, Generation: session.id}); err != nil {
			// Return the installed session so the normal deferred disconnect path
			// runs even if the acknowledgement was lost after commit.
			session.conn.Close()
		}
		control.Close() // EOF before the client consumes the separate lifecycle hello.
	}
	if session.lifecycle != nil {
		if err := json.NewEncoder(session.lifecycle).Encode(protocol.Lifecycle{Version: 1, Type: "hello", TunnelID: tunnelID, Generation: session.id}); err != nil {
			session.conn.Close()
		}
		session.lifecycle.SetWriteDeadline(time.Time{})
	}
	outcome = telemetry.Success
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

// handleTCP opens one yamux stream for each HTTP request. An ingress connection
// can be reused across hosts, so only negotiated upgrades may own that connection.
func (s *Server) handleTCP(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	outcome := telemetry.Unavailable
	defer func() { s.observer.Operation(telemetry.Forward, started, outcome) }()
	tunnelID := ParseTunnelID(r.Host, s.domain)
	if tunnelID == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		drainRejected(w)
		return
	}
	session, ok := s.connections[tunnelID]
	if ok {
		ctx, cancel := context.WithCancel(r.Context())
		r = r.WithContext(ctx)
		s.requestCancels[r] = cancel
		s.requests.Add(1)
	}
	s.mu.Unlock()

	if !ok {
		jsonError(w, "Tunnel not connected", http.StatusServiceUnavailable)
		return
	}
	defer func() {
		s.mu.Lock()
		cancel := s.requestCancels[r]
		delete(s.requestCancels, r)
		s.mu.Unlock()
		cancel()
		s.requests.Done()
	}()

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
	if size > 0 {
		outcome = telemetry.Success
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

// responseConn counts response-stream bytes, including HTTP headers, as before.
// Upgrade forwarding can read concurrently while the proxy finishes shutting down.
type responseConn struct {
	net.Conn
	bytes atomic.Int64
}

func (c *responseConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	c.bytes.Add(int64(n))
	return n, err
}

// Keep normal HTTP under net/http's request dispatcher. Relaying a hijacked
// connection would send subsequent requests (even other hosts or owner APIs) to
// the first local service. ReverseProxy hijacks only a negotiated 101 upgrade.
func (s *Server) proxyStream(w http.ResponseWriter, r *http.Request, stream net.Conn) (int64, bool) {
	response := &responseConn{Conn: stream}
	transport := &http.Transport{
		DialContext: func(context.Context, string, string) (net.Conn, error) {
			return response, nil
		},
		DisableKeepAlives:  true, // Each stream belongs to exactly one request.
		DisableCompression: true,
	}
	defer transport.CloseIdleConnections()
	ok := true
	proxy := &httputil.ReverseProxy{
		Director: func(out *http.Request) {
			out.URL.Scheme = "http"
			out.URL.Host = r.Host
		},
		Transport: transport,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			ok = false
			jsonError(w, "Failed to forward request", http.StatusBadGateway)
		},
	}
	proxy.ServeHTTP(w, r)
	return response.bytes.Load(), ok
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
	s.closeOnce.Do(func() {
		s.shutdownErr = s.shutdown(ctx)
	})
	return s.shutdownErr
}
