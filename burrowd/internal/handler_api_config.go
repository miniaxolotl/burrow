package internal

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"mode": s.deploymentMode,
		"limits": map[string]int{
			"anon": s.limits.AnonLimit,
			"free": s.limits.UserLimit,
			"paid": s.limits.PaidLimit,
		},
		"features": map[string]bool{
			"auth":    s.deploymentMode == "cloud",
			"billing": s.deploymentMode == "cloud",
		},
	})
}