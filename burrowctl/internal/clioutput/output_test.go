package clioutput

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type blockedWriter struct {
	started chan struct{}
	release chan struct{}
	writes  atomic.Int32
}

func (w *blockedWriter) Write(p []byte) (int, error) {
	w.writes.Add(1)
	close(w.started)
	<-w.release
	return len(p), nil
}

func TestWriterBlockedOutputFailsClosed(t *testing.T) {
	blocked := &blockedWriter{started: make(chan struct{}), release: make(chan struct{})}
	defer close(blocked.release)
	writer := &Writer{out: blocked, format: "jsonl"}
	result := make(chan error, 1)
	go func() {
		result <- writer.Emit("connected", "example", map[string]bool{"registration_confirmed": false})
	}()
	<-blocked.started
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("blocked output reported success")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("blocked output did not fail within its bounded deadline")
	}
	if err := writer.Emit("stopped", "", nil); err == nil || blocked.writes.Load() != 1 {
		t.Fatal("failed output launched another blocked write")
	}
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

func TestWriterDetectsShortWrites(t *testing.T) {
	writer := &Writer{out: shortWriter{}, format: "json"}
	if err := writer.Emit("result", "", nil); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short write: %v", err)
	}
}

func TestWriterTextEventsAreNotJSONOrANSI(t *testing.T) {
	var out bytes.Buffer
	writer := &Writer{out: &out, format: "text"}
	if err := writer.Emit("connected", "example", map[string]string{"url": "https://example.test"}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "connected example ") || strings.Contains(out.String(), "\x1b") {
		t.Fatalf("not plain headless text: %q", &out)
	}
}
