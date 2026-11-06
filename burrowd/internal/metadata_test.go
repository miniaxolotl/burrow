package internal

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServerMetadataRecordsPeerStatusAndProxyFailure(t *testing.T) {
	for _, code := range []int{200, 204, 302, 404, 500} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			s := NewServer(nil, "example.test", false)
			proxyFixture(t, s, "probe", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(code) }))
			w := httptest.NewRecorder()
			s.ServeHTTP(w, httptest.NewRequest("GET", "http://probe.example.test/api?secret=hidden", nil))
			s.mu.RLock()
			defer s.mu.RUnlock()
			if len(s.logs["probe"]) != 1 || s.logs["probe"][0].StatusCode != code {
				t.Fatalf("untruthful metadata: %+v", s.logs["probe"])
			}
			if strings.Contains(s.logs["probe"][0].Path, "secret") {
				t.Fatal("query retained")
			}
		})
	}
	s := NewServer(nil, "example.test", false)
	proxyFixture(t, s, "probe", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic(http.ErrAbortHandler) }))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "http://probe.example.test/failure", nil))
	if w.Code != 502 || len(s.logs["probe"]) != 1 || s.logs["probe"][0].StatusCode != 502 {
		t.Fatal("proxy-generated failure absent from history")
	}
}

func TestServerMetadataRetains200OnMidBodyInterruption(t *testing.T) {
	s := NewServer(nil, "example.test", false)
	proxyFixture(t, s, "probe", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		io.WriteString(w, "partial")
		w.(http.Flusher).Flush()
		panic(http.ErrAbortHandler)
	}))
	proxy := httptest.NewServer(s)
	defer proxy.Close()
	req, _ := http.NewRequest("GET", proxy.URL+"/partial", nil)
	req.Host = "probe.example.test"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err == nil {
		t.Fatal("partial body reported success")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := s.logs["probe"]
	if len(rows) != 1 || rows[0].StatusCode != 200 || rows[0].Outcome != "interrupted" || rows[0].StatusOrigin != "tunnel_peer" {
		t.Fatalf("mid-body status rewritten: %+v", rows)
	}
}

func TestServerMetadataIgnoresInformationalStatus(t *testing.T) {
	s := NewServer(nil, "example.test", false)
	proxyFixture(t, s, "probe", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(103)
		w.WriteHeader(100)
		w.WriteHeader(204)
	}))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "http://probe.example.test/info", nil))
	if len(s.logs["probe"]) != 1 || s.logs["probe"][0].StatusCode != 204 {
		t.Fatal("informational status mistaken for final response")
	}
}
