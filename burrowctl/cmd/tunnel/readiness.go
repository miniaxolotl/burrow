package tunnel

import (
	"burrow/burrowctl/internal"
	"sort"
)

type acknowledgedTunnel struct {
	TunnelID   string `json:"tunnel_id"`
	Generation string `json:"generation"`
	URL        string `json:"url"`
}
type readiness struct {
	count int
	ready bool
	ack   map[string]acknowledgedTunnel
}

func newReadiness(count int) *readiness {
	return &readiness{count: count, ack: make(map[string]acknowledgedTunnel)}
}
func (r *readiness) observe(event internal.LifecycleEvent) (string, []acknowledgedTunnel) {
	switch event.Type {
	case "registered":
		if event.RegistrationConfirmed && event.Generation != "" {
			r.ack[event.TunnelID] = acknowledgedTunnel{event.TunnelID, event.Generation, event.URL}
		}
	case "disconnected", "failed", "draining":
		if current, ok := r.ack[event.TunnelID]; ok && (event.Generation == "" || current.Generation == event.Generation) {
			delete(r.ack, event.TunnelID)
		}
	}
	now := len(r.ack) == r.count
	if now == r.ready {
		return "", nil
	}
	r.ready = now
	if !now {
		return "not_ready", nil
	}
	items := make([]acknowledgedTunnel, 0, len(r.ack))
	for _, item := range r.ack {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].TunnelID < items[j].TunnelID })
	return "ready", items
}
