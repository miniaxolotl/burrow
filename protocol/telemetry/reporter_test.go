package telemetry

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/posthog/posthog-go"
)

func fixtureConfig(t *testing.T) Config {
	t.Helper()
	t.Setenv("CI", "")
	t.Setenv("DO_NOT_TRACK", "")
	t.Setenv("BURROW_TELEMETRY", "")
	return Config{Enabled: true, Surface: "cli", Version: "1.0.0", Revision: strings.Repeat("a", 40), Key: "phc_test", StateDir: t.TempDir()}
}

func TestRealSDKSerializationAndPrivateScope(t *testing.T) {
	config := fixtureConfig(t)
	var mu sync.Mutex
	payloads := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var body io.Reader = req.Body
		if req.Header.Get("Content-Encoding") == "gzip" {
			compressed, err := gzip.NewReader(req.Body)
			if err != nil {
				t.Error(err)
				return
			}
			defer compressed.Close()
			body = compressed
		}
		data, _ := io.ReadAll(body)
		mu.Lock()
		payloads = append(payloads, string(data))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":1}`)
	}))
	defer server.Close()
	config.DSN = strings.Replace(server.URL, "http://", "http://public@", 1) + "/1"
	global := sentry.CurrentHub().Client()
	r := newReporter(config, server.URL, nil, 1)
	if r.posthog == nil || r.hub == nil {
		t.Fatal("SDK initialization failed")
	}
	r.Track(TunnelOpened)
	r.Operation(Connect, time.Now().Add(-12*time.Millisecond), Failure)
	_ = r.posthog.Enqueue(posthog.Capture{Event: "operation_completed", DistinctId: "PRIVATE", Properties: posthog.Properties{"operation": "tunnel_connect", "outcome": "error", "token": "PRIVATE", "$set": map[string]any{"email": "PRIVATE"}, "path": "PRIVATE"}})
	r.hub.CaptureEvent(&sentry.Event{Tags: map[string]string{"operation": "tunnel_connect"}, Message: "PRIVATE", ServerName: "PRIVATE", User: sentry.User{Email: "PRIVATE"}, Request: &sentry.Request{URL: "PRIVATE"}, Exception: []sentry.Exception{{Type: "PRIVATE", Value: "PRIVATE", Stacktrace: &sentry.Stacktrace{Frames: []sentry.Frame{{Filename: "/PRIVATE/burrowctl/main.go", Function: "main", Lineno: 12, ContextLine: "PRIVATE", Vars: map[string]interface{}{"secret": "PRIVATE"}}}}}}})
	r.Close()
	if sentry.CurrentHub().Client() != global {
		t.Fatal("reporter changed global hub")
	}
	mu.Lock()
	defer mu.Unlock()
	all := strings.Join(payloads, "\n")
	if strings.Contains(all, "PRIVATE") {
		t.Fatalf("private sentinel in SDK payload: %s", all)
	}
	for _, marker := range []string{"tunnel_opened", "operation_completed", `"type":"transaction"`, `"type":"event"`, `"type":"log"`, `"type":"trace_metric"`, "burrowctl/main.go"} {
		if !strings.Contains(all, marker) {
			t.Errorf("missing %s in %s", marker, all)
		}
	}
	if host, _ := os.Hostname(); host != "" && strings.Contains(all, host) {
		t.Error("machine hostname leaked")
	}
	for _, body := range payloads {
		var batch struct {
			Batch []struct {
				Properties map[string]any `json:"properties"`
				DistinctID string         `json:"distinct_id"`
			} `json:"batch"`
		}
		if json.Unmarshal([]byte(body), &batch) != nil {
			continue
		}
		for _, event := range batch.Batch {
			if event.DistinctID != r.id || event.Properties["$process_person_profile"] != false {
				t.Error("non-anonymous PostHog event")
			}
		}
	}
}

func TestOptOutIsInertAndAffectsRunningReporter(t *testing.T) {
	config := fixtureConfig(t)
	var requests int
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { mu.Lock(); requests++; mu.Unlock(); w.WriteHeader(200) }))
	defer server.Close()
	config.DSN = strings.Replace(server.URL, "http://", "http://public@", 1) + "/1"
	r := newReporter(config, server.URL, nil, 0)
	r.Track(TunnelOpened) // queued, then opted out before either SDK can flush
	r.Operation(Connect, time.Now(), Success)
	if err := SetEnabled(config.StateDir, false); err != nil {
		t.Fatal(err)
	}
	r.Track(TunnelReconnected)
	r.Operation(Connect, time.Now(), Failure)
	r.Close()
	mu.Lock()
	defer mu.Unlock()
	if requests != 0 {
		t.Fatalf("opt-out sent %d requests", requests)
	}
	if _, err := os.Stat(filepath.Join(config.StateDir, "telemetry-id")); !os.IsNotExist(err) {
		t.Fatal("identity persisted after opt-out")
	}
	if err := SetEnabled(config.StateDir, true); err != nil {
		t.Fatal(err)
	}
	if next := installationID(config.StateDir); next == r.id {
		t.Fatal("old identity reused")
	}
	if !r.disabled() {
		t.Fatal("old reporter resumed after identity reset")
	}
	for _, key := range []string{"DO_NOT_TRACK", "CI", "BURROW_TELEMETRY"} {
		t.Run(key, func(t *testing.T) {
			value := "1"
			if key == "BURROW_TELEMETRY" {
				value = "0"
			}
			t.Setenv(key, value)
			local := config
			local.StateDir = filepath.Join(t.TempDir(), "not-created")
			disabled := New(local)
			if disabled.posthog != nil || disabled.hub != nil {
				t.Fatal("opt-out initialized SDK")
			}
			if _, err := os.Stat(local.StateDir); !os.IsNotExist(err) {
				t.Fatal("opt-out wrote state")
			}
		})
	}
}

func TestConcurrentIdentityAndResetPreserveOwnerCredential(t *testing.T) {
	config := fixtureConfig(t)
	owner := filepath.Join(config.StateDir, "owners.json")
	if err := os.WriteFile(owner, []byte("OWNER_SECRET"), 0600); err != nil {
		t.Fatal(err)
	}
	ids := make(chan string, 12)
	var wg sync.WaitGroup
	for range 12 {
		wg.Go(func() { ids <- installationID(config.StateDir) })
	}
	wg.Wait()
	close(ids)
	first := ""
	for id := range ids {
		if id == "" {
			t.Fatal("missing identity")
		}
		if first == "" {
			first = id
		}
		if first != id {
			t.Fatal("concurrent identities differ")
		}
	}
	info, _ := os.Stat(filepath.Join(config.StateDir, "telemetry-id"))
	if info.Mode().Perm() != 0600 {
		t.Fatal("identity is not private")
	}
	if err := SetEnabled(config.StateDir, false); err != nil {
		t.Fatal(err)
	}
	if err := SetEnabled(config.StateDir, true); err != nil {
		t.Fatal(err)
	}
	if next := installationID(config.StateDir); next == "" || next == first {
		t.Fatal("opt-in did not reset identity")
	}
	data, _ := os.ReadFile(owner)
	if string(data) != "OWNER_SECRET" {
		t.Fatal("owner credentials changed")
	}
}

type unavailableTransport struct{}

func (unavailableTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	<-req.Context().Done()
	return nil, req.Context().Err()
}
func TestUnavailableProviderHasBoundedQueueAndShutdown(t *testing.T) {
	config := fixtureConfig(t)
	config.DSN = "http://public@127.0.0.1:1/1"
	r := newReporter(config, "http://127.0.0.1:1", unavailableTransport{}, 1)
	start := time.Now()
	for range 10_000 {
		r.Track(TunnelOpened)
	}
	if time.Since(start) > time.Second {
		t.Fatal("capture blocked on unavailable provider")
	}
	r.Operation(Connect, start, Failure)
	start = time.Now()
	r.Close()
	if time.Since(start) > 800*time.Millisecond {
		t.Fatal("shutdown exceeded telemetry budget")
	}
}
