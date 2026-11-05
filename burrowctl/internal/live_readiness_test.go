package internal

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Only synthetic, side-effect-free smoke requests use this edge-readiness wait.
// A permanent status or wrong successful response still fails the caller's checks.
func liveRequest(ctx context.Context, client *http.Client, method, url, body string) (*http.Response, error) {
	for {
		req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(body))
		if err != nil {
			return nil, err
		}
		response, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if response.StatusCode != http.StatusBadGateway && response.StatusCode != http.StatusServiceUnavailable && response.StatusCode != http.StatusGatewayTimeout {
			return response, nil
		}
		_, _ = io.CopyN(io.Discard, response.Body, 8192)
		response.Body.Close()
		timer := time.NewTimer(200 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func TestLiveRequestWaitsForTransientEdgeRouting(t *testing.T) {
	for _, status := range []int{502, 503, 504} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					w.WriteHeader(status)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			response, err := liveRequest(ctx, server.Client(), http.MethodPost, server.URL, "synthetic-smoke-body")
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK || calls.Load() != 2 {
				t.Fatalf("edge was not ready: status=%d calls=%d", response.StatusCode, calls.Load())
			}
		})
	}
}

func TestLiveRequestPreservesPermanentFailures(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	response, err := liveRequest(context.Background(), server.Client(), http.MethodPost, server.URL, "synthetic-smoke-body")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusForbidden || calls.Load() != 1 {
		t.Fatal("a permanent failure must not be retried or hidden")
	}
}

func TestLiveRequestBoundsEdgeWaitByCallerDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	response, err := liveRequest(ctx, server.Client(), http.MethodPost, server.URL, "synthetic-smoke-body")
	if response != nil {
		response.Body.Close()
	}
	if err == nil {
		t.Fatal("an unavailable edge must not pass before its deadline")
	}
}
