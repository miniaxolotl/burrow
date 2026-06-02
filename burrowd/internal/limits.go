package internal

import (
	"fmt"
	"net/http"
)

func CheckLimit(mode string, limits *TunnelLimits, pg *PostgresClient, r *http.Request) (bool, string) {
	if mode == "oss" {
		return true, ""
	}

	ip := clientIP(r)

	// Check JWT for authenticated user
	userID := userIDFromContext(r.Context())
	if userID != nil {
		// Authenticated user — check user limit
		count, err := pg.GetActiveTunnelCountByUser(r.Context(), *userID)
		if err != nil {
			return false, "Error checking tunnel limit"
		}
		plan := r.Context().Value(contextKey("plan"))
		planStr, _ := plan.(string)
		limit := limits.UserLimit
		if planStr == "pro" {
			limit = limits.PaidLimit
		}
		if limit > 0 && count >= limit {
			return false, fmt.Sprintf("You have reached your limit of %d active tunnels. Upgrade to Pro for unlimited tunnels.", limit)
		}
		return true, ""
	}

	// Anonymous — check IP limit
	count, err := pg.GetActiveTunnelCountByIP(r.Context(), ip)
	if err != nil {
		return false, "Error checking tunnel limit"
	}
	if count >= limits.AnonLimit {
		return false, fmt.Sprintf("This IP address already has %d active tunnel(s). Create a free account for up to %d tunnels, or upgrade to Pro for unlimited tunnels.", limits.AnonLimit, limits.UserLimit)
	}
	return true, ""
}
