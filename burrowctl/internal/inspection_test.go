package internal

import (
	"testing"
	"time"

	"burrow/burrowctl/internal/routing"
)

func TestInspectionIsBoundedNonblockingAndLegacySessionFenced(t *testing.T) {
	c := NewClient("localhost", "synthetic-owner", "localhost", false)
	defer c.Close()
	c.SetInspect(true)
	tc := &TunnelConn{ctx: t.Context(), epoch: "private-window-2", history: &liveHistory{}}
	c.tunnels["a"] = tc
	e := routing.Exchange{Method: "POST", Path: "/hook?secret=private", Status: 200, StartedAt: time.Now(), CompletedAt: time.Now(), Outcome: "completed"}
	c.recordExchange("a", "private-window-1", e)
	if len(c.Inspection("a").Rows) != 0 {
		t.Fatal("old legacy session wrote new window")
	}
	tc.history.mu.Lock()
	c.recordExchange("a", "private-window-2", e)
	tc.history.mu.Unlock()
	if c.Inspection("a").Dropped != 1 {
		t.Fatal("stalled reader blocked instead of dropping")
	}
	for range 250 {
		c.recordExchange("a", "private-window-2", e)
	}
	snapshot := c.Inspection("a")
	if len(snapshot.Rows) != 200 || snapshot.Evicted != 50 || snapshot.Rows[0].Path != "/hook" || snapshot.Rows[0].Generation != "" {
		t.Fatal("inspection bounds/privacy or unknown generation changed")
	}
}
