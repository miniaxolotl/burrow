package tunnel

import (
	"burrow/burrowctl/internal"
	"testing"
)

func TestReadinessTracksAllAcknowledgedGenerations(t *testing.T) {
	r := newReadiness(2)
	steps := []struct{ kind, id, generation, want string }{
		{"connected", "a", "", ""}, {"registered", "a", "a1", ""},
		{"connected", "b", "", ""}, {"registered", "b", "b1", "ready"},
		{"registered", "b", "b1", ""}, {"disconnected", "a", "", "not_ready"},
		{"reconnecting", "a", "", ""}, {"connected", "a", "", ""},
		{"registered", "a", "a2", "ready"}, {"failed", "b", "", "not_ready"},
	}
	for _, s := range steps {
		kind, ack := r.observe(internal.LifecycleEvent{Type: s.kind, TunnelID: s.id, Generation: s.generation, RegistrationConfirmed: s.kind == "registered"})
		if kind != s.want {
			t.Fatalf("%+v => %q", s, kind)
		}
		if kind == "ready" && len(ack) != 2 {
			t.Fatal("partial ready")
		}
	}
}
