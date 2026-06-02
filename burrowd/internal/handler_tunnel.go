package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"burrow/protocol"

	"github.com/hashicorp/yamux"
)

func (s *Server) tokenFromRequest(r *http.Request) string {
	if t := r.Header.Get("X-Tunnel-Token"); t != "" {
		return t
	}
	return r.URL.Query().Get("token")
}

func (s *Server) authAdmin(r *http.Request) bool {
	if s.secret == "" {
		return false
	}
	return protocol.ValidateToken(s.tokenFromRequest(r), s.secret)
}

func (s *Server) authForTunnel(r *http.Request, tunnelID string) bool {
	if s.authAdmin(r) {
		return true
	}
	td, err := s.registry.Get(r.Context(), tunnelID)
	if err != nil || td == nil {
		return false
	}
	if td.UserID == nil {
		return true
	}
	userID := userIDFromContext(r.Context())
	return userID != nil && *userID == *td.UserID
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Rate limit check
	if allowed, msg := CheckLimit(s.deploymentMode, &s.limits, s.pg, r); !allowed {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"error":       "Tunnel limit reached",
			"message":     msg,
			"upgrade_url": "/register",
		})
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

	clientIPVal := clientIP(r)

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
	conn.SetReadLimit(protocol.MaxMessageSize)
	defer conn.Close()

	cfg := yamux.DefaultConfig()
	cfg.KeepAliveInterval = 30 * time.Second
	session, err := yamux.Server(&protocol.WsConn{Conn: conn}, cfg)
	if err != nil {
		return
	}
	defer session.Close()

	if _, err := s.registry.Register(r.Context(), tunnelID, uint16(port), nil, clientIPVal); err != nil {
		return
	}

	// Insert tunnel into Postgres
	if s.pg != nil {
		var userID *int64
		if uid := userIDFromContext(r.Context()); uid != nil {
			userID = uid
		}
		if _, err := s.pg.CreateTunnel(r.Context(), tunnelID, int(port), clientIPVal, userID); err != nil {
			log.Printf("create tunnel: %v", err)
		}
	}

	defer func() {
		cancel := make(chan struct{})
		s.mu.Lock()
		s.pendingRemoves[tunnelID] = cancel
		s.mu.Unlock()
		go func() {
			select {
			case <-cancel:
				s.mu.Lock()
				if s.pendingRemoves[tunnelID] == cancel {
					delete(s.pendingRemoves, tunnelID)
				}
				s.mu.Unlock()
			case <-time.After(tunnelGracePeriod):
				s.mu.Lock()
				if s.pendingRemoves[tunnelID] == cancel {
					delete(s.pendingRemoves, tunnelID)
				}
				s.mu.Unlock()
				s.registry.Remove(tunnelID)
				s.cleanupTunnel(tunnelID)
				if s.pg != nil {
					if err := s.pg.UpdateTunnelStatus(context.Background(), tunnelID, "stopped"); err != nil {
						log.Printf("update tunnel status: %v", err)
					}
				}
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

	done := make(chan struct{})
	var keepaliveWg sync.WaitGroup
	keepaliveWg.Go(func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.registry.Register(context.Background(), tunnelID, uint16(port), nil, clientIPVal)
			case <-done:
				return
			}
		}
	})
	defer func() {
		close(done)
		keepaliveWg.Wait()
	}()

	for {
		stream, err := session.AcceptStream()
		if err != nil {
			return
		}
		stream.Close()
	}
}

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
		jsonError(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client, brw, err := hj.Hijack()
	if err != nil {
		jsonError(w, "Failed to hijack connection", http.StatusInternalServerError)
		return
	}

	if err := r.Write(stream); err != nil {
		client.Close()
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

func (s *Server) closeTunnel(tunnelID string) {
	s.mu.Lock()
	if session, ok := s.connections[tunnelID]; ok {
		session.Close()
		delete(s.connections, tunnelID)
	}
	s.mu.Unlock()
	s.registry.Remove(tunnelID)
	s.cleanupTunnel(tunnelID)
}