package internal

import "testing"

type liveLoadStage struct{ rps, concurrency, seconds int }

func liveLoadProfile(heavy bool) []liveLoadStage {
	if heavy {
		return []liveLoadStage{{100, 32, 20}, {250, 64, 20}, {500, 96, 20}, {750, 128, 20}, {1000, 128, 20}}
	}
	return []liveLoadStage{{5, 2, 10}, {15, 5, 10}, {30, 10, 10}, {60, 20, 10}, {100, 32, 10}}
}

func TestLiveLoadProfileBounds(t *testing.T) {
	for _, tc := range []struct {
		name                                string
		heavy                               bool
		requests, seconds, rps, concurrency int
	}{
		{"normal", false, 2100, 50, 100, 32},
		{"heavy", true, 52000, 100, 1000, 128},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests, seconds, peakRPS, peakConcurrency := 0, 0, 0, 0
			for _, stage := range liveLoadProfile(tc.heavy) {
				if stage.rps <= 0 || stage.seconds <= 0 || stage.concurrency <= 0 || stage.rps < peakRPS {
					t.Fatal("invalid/unbounded/nonascending load stage")
				}
				requests += stage.rps * stage.seconds
				seconds += stage.seconds
				if stage.rps > peakRPS {
					peakRPS = stage.rps
				}
				if stage.concurrency > peakConcurrency {
					peakConcurrency = stage.concurrency
				}
			}
			if requests != tc.requests || seconds != tc.seconds || peakRPS != tc.rps || peakConcurrency != tc.concurrency {
				t.Fatalf("profile requests=%d seconds=%d peakRPS=%d concurrency=%d; want %d/%d/%d/%d", requests, seconds, peakRPS, peakConcurrency, tc.requests, tc.seconds, tc.rps, tc.concurrency)
			}
		})
	}
}
