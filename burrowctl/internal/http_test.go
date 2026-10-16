package internal

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestManagementRedirectsDoNotLeakCredentials(t *testing.T) {
	var externalRequests atomic.Int32
	outside := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { externalRequests.Add(1) }))
	defer outside.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/external":
			http.Redirect(w, r, outside.URL, http.StatusFound)
		case "/internal":
			http.Redirect(w, r, "/destination", http.StatusFound)
		case "/destination":
			if r.Header.Get("X-Tunnel-Token") != "private-owner" {
				t.Error("same-origin credential missing")
			}
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer origin.Close()
	client := NewManagementHTTPClient()
	for _, test := range []struct {
		path   string
		status int
	}{{"/external", http.StatusFound}, {"/internal", http.StatusNoContent}} {
		req, _ := http.NewRequest("GET", origin.URL+test.path, nil)
		req.Header.Set("X-Tunnel-Token", "private-owner")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != test.status {
			t.Fatalf("%s: HTTP %d", test.path, resp.StatusCode)
		}
	}
	if externalRequests.Load() != 0 {
		t.Fatal("credential-bearing redirect reached another origin")
	}
	secure, _ := http.NewRequest("GET", "https://brmbl.dev/", nil)
	plain, _ := http.NewRequest("GET", "http://brmbl.dev/", nil)
	if client.CheckRedirect(plain, []*http.Request{secure}) != http.ErrUseLastResponse {
		t.Fatal("allowed TLS downgrade")
	}
}
