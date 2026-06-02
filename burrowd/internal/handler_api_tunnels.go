package internal

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) handleAPITunnelList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := userIDFromContext(r.Context())
	var tunnels []*Tunnel
	var err error
	if userID != nil {
		tunnels, err = s.pg.ListTunnelsByUser(r.Context(), *userID)
	} else {
		// Admin or anonymous — list all active
		tunnels, err = s.pg.ListActiveTunnels(r.Context())
	}
	if err != nil {
		jsonError(w, "Failed to list tunnels", http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, tunnels)
}

func (s *Server) handleAPITunnelDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tunnelID := strings.TrimPrefix(r.URL.Path, "/api/tunnels/")
	tunnelID = strings.Split(tunnelID, "/")[0]
	if tunnelID == "" {
		jsonError(w, "Tunnel ID required", http.StatusBadRequest)
		return
	}
	tunnel, err := s.pg.GetTunnelByTunnelID(r.Context(), tunnelID)
	if err != nil {
		jsonError(w, "Tunnel not found", http.StatusNotFound)
		return
	}
	respond(w, http.StatusOK, tunnel)
}

func (s *Server) handleAPITunnelStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tunnelID := extractTunnelID(r.URL.Path)
	if tunnelID == "" {
		jsonError(w, "Tunnel ID required", http.StatusBadRequest)
		return
	}
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	if f := r.URL.Query().Get("from"); f != "" {
		if t, err := time.Parse(time.RFC3339, f); err == nil {
			from = t
		}
	}
	if t := r.URL.Query().Get("to"); t != "" {
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			to = parsed
		}
	}
	stats, err := GetTunnelStats(r.Context(), s.pg, tunnelID, from, to)
	if err != nil {
		jsonError(w, "Failed to get tunnel stats", http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, stats)
}

func (s *Server) handleAPITunnelLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tunnelID := extractTunnelID(r.URL.Path)
	if tunnelID == "" {
		jsonError(w, "Tunnel ID required", http.StatusBadRequest)
		return
	}
	limit := 100
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	if f := r.URL.Query().Get("from"); f != "" {
		if t, err := time.Parse(time.RFC3339, f); err == nil {
			from = t
		}
	}
	if t := r.URL.Query().Get("to"); t != "" {
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			to = parsed
		}
	}
	logs, err := GetTunnelLogs(r.Context(), s.pg, tunnelID, from, to, limit, offset)
	if err != nil {
		jsonError(w, "Failed to get tunnel logs", http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, logs)
}

func (s *Server) handleAPITunnelClose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tunnelID := extractTunnelID(r.URL.Path)
	if tunnelID == "" {
		jsonError(w, "Tunnel ID required", http.StatusBadRequest)
		return
	}
	if err := CloseTunnel(r.Context(), s.pg, s.registry, tunnelID); err != nil {
		jsonError(w, "Failed to close tunnel", http.StatusInternalServerError)
		return
	}
	s.closeTunnel(tunnelID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPITunnelRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tunnelID := extractTunnelID(r.URL.Path)
	if tunnelID == "" {
		jsonError(w, "Tunnel ID required", http.StatusBadRequest)
		return
	}
	if err := RestartTunnel(r.Context(), s.pg, s.registry, tunnelID); err != nil {
		jsonError(w, "Failed to restart tunnel", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func extractTunnelID(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "tunnels" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
