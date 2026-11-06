package routing

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMetadataHeadersAndCompletionHaveSeparateLifetimes(t *testing.T) {
	release := make(chan struct{})
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		w.(http.Flusher).Flush()
		select {
		case <-release:
			io.WriteString(w, "done")
		case <-r.Context().Done():
		}
	}))
	defer app.Close()
	headers := make(chan int, 1)
	completed := make(chan Exchange, 1)
	table := &Table{Version: 1, Routes: []Route{{Path: "/", Port: port(t, app)}}, Headers: func(_ Route, status int, _ time.Time) { headers <- status }, Report: func(e Exchange) { completed <- e }}
	proxy := httptest.NewServer(table)
	defer proxy.Close()
	resp, err := http.Get(proxy.URL + "/hook?token=not-metadata")
	if err != nil {
		close(release)
		t.Fatal(err)
	}
	if status := <-headers; status != 503 {
		close(release)
		t.Fatal("app failure was not an observed response")
	}
	select {
	case <-completed:
		close(release)
		t.Fatal("headers marked stream complete")
	default:
	}
	close(release)
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	e := <-completed
	if e.Status != 503 || e.Path != "/hook" || e.Origin != "application" || e.Outcome != "completed" {
		t.Fatalf("untruthful metadata: %+v", e)
	}
}

func TestRejectedRoutesStillProduceMetadata(t *testing.T) {
	var got Exchange
	table := &Table{Version: 1, Routes: []Route{{Path: "/api", Port: 3000}}, Report: func(e Exchange) { got = e }}
	table.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/other?secret=private", nil))
	if got.Status != 404 || got.Route != "" || got.Outcome != "rejected" || got.Path != "/other" {
		t.Fatalf("rejection metadata: %+v", got)
	}
}
