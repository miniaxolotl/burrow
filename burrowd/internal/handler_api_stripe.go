package internal

import (
	"io"
	"log"
	"net/http"
)

func (s *Server) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	sig := r.Header.Get("Stripe-Signature")
	if sig == "" {
		jsonError(w, "Missing signature", http.StatusBadRequest)
		return
	}
	stripeClient := NewStripeClient(s.stripeSecretKey, s.stripeWebhookSecret, s.stripeProPriceID)
	event, err := stripeClient.ConstructEvent(payload, sig)
	if err != nil {
		jsonError(w, "Invalid signature", http.StatusBadRequest)
		return
	}
	if err := ProcessWebhookEvent(r.Context(), stripeClient, s.pg, event); err != nil {
		log.Printf("stripe webhook: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
