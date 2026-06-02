package internal

import (
	"encoding/json"
	"net/http"
)

type checkoutRequest struct {
	SuccessURL string `json:"success_url"`
	CancelURL  string `json:"cancel_url"`
}

type checkoutResponse struct {
	URL string `json:"url"`
}

type portalResponse struct {
	URL string `json:"url"`
}

func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := userIDFromContext(r.Context())
	if userID == nil {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.SuccessURL == "" || req.CancelURL == "" {
		jsonError(w, "success_url and cancel_url required", http.StatusBadRequest)
		return
	}
	stripeClient := NewStripeClient(s.stripeSecretKey, s.stripeWebhookSecret, s.stripeProPriceID)
	url, err := CreateCheckout(r.Context(), stripeClient, s.pg, *userID, req.SuccessURL, req.CancelURL)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, checkoutResponse{URL: url})
}

func (s *Server) handlePortal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := userIDFromContext(r.Context())
	if userID == nil {
		jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	stripeClient := NewStripeClient(s.stripeSecretKey, s.stripeWebhookSecret, s.stripeProPriceID)
	returnURL := r.URL.Query().Get("return_url")
	if returnURL == "" {
		returnURL = "/settings"
	}
	url, err := CreatePortal(r.Context(), stripeClient, s.pg, *userID, returnURL)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, portalResponse{URL: url})
}
