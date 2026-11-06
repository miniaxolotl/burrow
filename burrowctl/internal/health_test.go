package internal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealthFreshnessAndGenerationFence(t *testing.T) {
	c := NewClient("localhost", "owner", "localhost", false)
	defer c.Close()
	c.tunnels["a"] = &TunnelConn{ctx: t.Context(), generation: "new", localHealth: map[uint16]HealthObservation{}}
	c.recordTCP("a", "old", 3000, time.Now(), nil)
	_, local := c.Health("a")
	if len(local) != 0 {
		t.Fatal("old generation changed health")
	}
	c.recordTCP("a", "new", 3000, time.Now(), nil)
	c.recordTCP("a", "new", 3000, time.Now(), errors.New("private dial error"))
	_, local = c.Health("a")
	if len(local) != 1 || local[0].Outcome != "failed" || local[0].LastSuccess.IsZero() {
		t.Fatalf("failure did not replace success: %+v", local)
	}
	o := local[0]
	o.ObservedAt = time.Now().Add(-31 * time.Second)
	if o.fresh(time.Now()).Freshness != "stale" {
		t.Fatal("old observation shown fresh")
	}
	if (HealthObservation{}).fresh(time.Now()).Freshness != "unknown" {
		t.Fatal("missing observation fabricated")
	}
}

func TestServerHealthRejectsRedirectAndStatusFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/private", 302) }))
	defer server.Close()
	o := CheckServer(t.Context(), strings.TrimPrefix(server.URL, "http://"))
	if o.Status != 302 || o.Outcome != "failed" {
		t.Fatalf("redirect followed: %+v", o)
	}
}

func TestDiagnosticBudgetReturnsEveryUncheckedTarget(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	report, err := Diagnose(ctx, "localhost", []uint16{1, 2, 3, 4, 5})
	if !errors.Is(err, context.Canceled) || len(report.Targets) != 5 {
		t.Fatalf("cancelled summary: %+v %v", report, err)
	}
	for _, target := range report.Targets {
		if target.Outcome != "not_checked" {
			t.Fatalf("fabricated check: %+v", target)
		}
	}
}
