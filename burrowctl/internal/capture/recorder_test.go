package capture

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"burrow/burrowctl/internal/routing"
)

func TestRecorderRedactsBeforeDurableStorageAndFencesIncompleteBodies(t *testing.T) {
	s := testStore(t)
	table := &routing.Table{Version: 1, Routes: []routing.Route{{Path: "/", Port: 3000}}}
	events := make(chan CaptureEvent, 16)
	r, err := NewRecorder(s, "tunnel", table, "", nil, func(e CaptureEvent) { events <- e })
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); r.Run(ctx) }()
	defer func() { cancel(); <-done }()
	req, _ := http.NewRequest("POST", "http://public.test/hook?token=private&safe=one", strings.NewReader("body"))
	req.Header.Set("Authorization", "Bearer private")
	req.Header.Set("X-Signature", "signature")
	observation := r.Begin(req, table.Routes[0], "/hook")
	body := observation.Wrap(req.Body)
	io.Copy(io.Discard, body)
	body.Close()
	observation.Complete()
	select {
	case event := <-events:
		if event.Type != "capture_saved" {
			t.Fatalf("recording failed: %+v", event)
		}
		stored, err := s.Read(event.ID, "tunnel")
		if err != nil {
			t.Fatal(err)
		}
		if stored.Headers.Get("Authorization") != "" || stored.Headers.Get("X-Signature") != "" || strings.Contains(stored.RawQuery, "private") || !stored.Redacted || !bytes.Equal(stored.Body, []byte("body")) {
			t.Fatal("redaction or payload changed incorrectly")
		}
	case <-time.After(time.Second):
		t.Fatal("capture did not commit")
	}
	req, _ = http.NewRequest("POST", "http://public.test/hook", strings.NewReader("incomplete"))
	observation = r.Begin(req, table.Routes[0], "/hook")
	observation.Wrap(req.Body)
	observation.Complete()
	select {
	case event := <-events:
		if event.Type != "capture_unavailable" || event.Code != "incomplete" {
			t.Fatal(event)
		}
	case <-time.After(time.Second):
		t.Fatal("incomplete not reported")
	}
}

func TestRecorderBackpressureNeverWaitsForDisk(t *testing.T) {
	s := testStore(t)
	table := &routing.Table{Version: 1, Routes: []routing.Route{{Path: "/", Port: 3000}}}
	r, err := NewRecorder(s, "tunnel", table, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Deliberately do not start its writer. Forwarding/copying must remain bounded.
	start := time.Now()
	for range 32 {
		req, _ := http.NewRequest("POST", "http://public.test/hook", strings.NewReader("body"))
		o := r.Begin(req, table.Routes[0], "/hook")
		body := o.Wrap(req.Body)
		io.Copy(io.Discard, body)
		body.Close()
		o.Complete()
	}
	if time.Since(start) > time.Second {
		t.Fatal("capture backpressure blocked forwarding")
	}
}
