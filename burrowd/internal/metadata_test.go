package internal

import (
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
