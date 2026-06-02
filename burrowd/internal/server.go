package internal

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"burrow/protocol"

	"github.com/IBM/sarama"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
)

const tunnelGracePeriod = 20 * time.Second
const maxLogsPerTunnel = 200

type Server struct {
	registry       *TunnelRegistry
	upgrader       websocket.Upgrader
	mux            *http.ServeMux
	apiHandler     http.Handler
	domain         string
	secret         string
	secure         bool
	deploymentMode string
	pg             *PostgresClient
	jwtSecret      string
	limits         TunnelLimits
	kafka          sarama.AsyncProducer
	stripeSecretKey      string
	stripeWebhookSecret  string
	stripeProPriceID     string
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

func NewServer(cfg ServerConfig) *Server {
	s := &Server{
		registry:       cfg.Registry,
		domain:         cfg.Domain,
		secret:         cfg.Secret,
		secure:         cfg.Secure,
		deploymentMode: cfg.DeploymentMode,
		pg:             cfg.PG,
		jwtSecret:      cfg.JWTSecret,
		limits:         cfg.Limits,
		kafka:          cfg.KafkaProducer,
		stripeSecretKey:     cfg.StripeSecretKey,
		stripeWebhookSecret: cfg.StripeWebhookSecret,
		stripeProPriceID:    cfg.StripeProPriceID,
		startTime:      time.Now(),
		connections:    make(map[string]*yamux.Session),
		pendingRemoves: make(map[string]chan struct{}),
		logs:           make(map[string][]*protocol.TunnelLog),
		totalSize:      make(map[string]int64),
	}

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

	if s.pg != nil {
		apiMux := http.NewServeMux()
		apiMux.HandleFunc("GET /api/config", s.handleConfig)
		apiMux.HandleFunc("POST /api/auth/register", s.handleRegister)
		apiMux.HandleFunc("POST /api/auth/login", s.handleLogin)
		apiMux.HandleFunc("POST /api/auth/refresh", s.handleRefreshToken)
		apiMux.HandleFunc("POST /api/auth/logout", s.handleLogout)
		apiMux.HandleFunc("GET /api/auth/me", s.handleMe)
		apiMux.HandleFunc("GET /api/tunnels", s.handleAPITunnelList)
		apiMux.HandleFunc("GET /api/tunnels/{id}", s.handleAPITunnelDetail)
		apiMux.HandleFunc("GET /api/tunnels/{id}/stats", s.handleAPITunnelStats)
		apiMux.HandleFunc("GET /api/tunnels/{id}/logs", s.handleAPITunnelLogs)
		apiMux.HandleFunc("DELETE /api/tunnels/{id}", s.handleAPITunnelClose)
		apiMux.HandleFunc("POST /api/tunnels/{id}/restart", s.handleAPITunnelRestart)
		apiMux.HandleFunc("POST /api/billing/checkout", s.handleCheckout)
		apiMux.HandleFunc("POST /api/billing/portal", s.handlePortal)
		apiMux.HandleFunc("POST /api/stripe/webhook", s.handleStripeWebhook)
		s.apiHandler = Chain(apiMux, loggingMiddleware(), corsMiddleware(), authMiddleware(s.jwtSecret, s.deploymentMode))
	}

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if ParseTunnelID(r.Host, s.domain) != "" {
		s.handleTCP(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		if s.apiHandler != nil {
			s.apiHandler.ServeHTTP(w, r)
		} else {
			jsonError(w, "Service unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	s.mux.ServeHTTP(w, r)
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func clientIP(r *http.Request) string {
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
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

type contextKey string

const contextKeyUserID contextKey = "user_id"

func userIDFromContext(ctx context.Context) *int64 {
	v := ctx.Value(contextKeyUserID)
	if v == nil {
		return nil
	}
	id, ok := v.(int64)
	if !ok {
		return nil
	}
	return &id
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

	if s.kafka != nil {
		event := &TunnelStatsEvent{
			TunnelID:  tunnelID,
			Method:    entry.Method,
			Path:      entry.Path,
			Size:      entry.Size,
			Duration:  entry.Duration,
			IP:        entry.IP,
			Timestamp: entry.Timestamp,
		}
		if err := PublishEvent(s.kafka, event); err != nil {
			log.Printf("kafka: publish event: %v", err)
		}
	}
}

func (s *Server) cleanupTunnel(tunnelID string) {
	s.logMu.Lock()
	delete(s.logs, tunnelID)
	s.logMu.Unlock()

	s.sizeMu.Lock()
	delete(s.totalSize, tunnelID)
	s.sizeMu.Unlock()
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