package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"burrow/protocol"
)

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

	s.closeTunnel(tunnelID)

	w.WriteHeader(http.StatusNoContent)
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