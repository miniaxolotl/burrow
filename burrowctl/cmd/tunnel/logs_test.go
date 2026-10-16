package tunnel

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"burrow/protocol"
)

func TestFollowLogsIncludesLateCompletionsAndRollingBuffers(t *testing.T) {
	started := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	old := &protocol.TunnelLog{ID: "old", Timestamp: started, Path: "/omitted-by-tail"}
	fast := &protocol.TunnelLog{ID: "fast", Timestamp: started.Add(time.Second), Path: "/same-request-fields"}
	slow := &protocol.TunnelLog{ID: "slow", Timestamp: started, Path: "/slow-finishes-later"}
	sameTime := &protocol.TunnelLog{ID: "distinct-request", Timestamp: fast.Timestamp, Path: fast.Path}
	afterRestart := &protocol.TunnelLog{ID: "after-restart", Timestamp: started, Path: "/new-buffer"}

	var out bytes.Buffer
	var seen map[string]struct{}
	for _, snapshot := range [][]*protocol.TunnelLog{
		{old, fast},
		{old, fast, slow},      // Earlier start, later completion.
		{fast, slow, sameTime}, // Buffer eviction and identical request fields.
		{fast, slow, sameTime}, // Repeated poll must not repeat entries.
		{afterRestart},         // A fresh buffer may contain older timestamps.
		{afterRestart},
	} {
		seen = printLogBatch(&out, snapshot, seen, 1)
		if len(seen) > len(snapshot) {
			t.Fatal("follow state grew beyond the rolling buffer")
		}
	}
	output := out.String()
	for path, count := range map[string]int{
		old.Path: 0, fast.Path: 2, slow.Path: 1, afterRestart.Path: 1,
	} {
		if got := strings.Count(output, path); got != count {
			t.Errorf("%s printed %d times, want %d\n%s", path, got, count, output)
		}
	}
}

func TestFollowLogsTailAppliesOnlyToInitialSnapshot(t *testing.T) {
	var out bytes.Buffer
	seen := printLogBatch(&out, nil, nil, 1)
	printLogBatch(&out, []*protocol.TunnelLog{
		{ID: "first", Path: "/first"}, {ID: "second", Path: "/second"},
	}, seen, 1)
	if !strings.Contains(out.String(), "/first") || !strings.Contains(out.String(), "/second") {
		t.Fatalf("tail suppressed newly completed requests: %s", out.String())
	}
}
