package internal

import "testing"

func TestServerOriginAndCredentialScope(t *testing.T) {
	for server, want := range map[string]string{
		"brmbl.dev":              "https://brmbl.dev:443",
		"BRMBL.DEV.:0443":        "https://brmbl.dev:443",
		"staging.brmbl.dev:8443": "https://staging.brmbl.dev:8443",
		"LOCALHOST.":             "http://localhost:80",
		"127.0.0.1:25701":        "http://127.0.0.1:25701",
		"[::1]:25701":            "http://[::1]:25701",
	} {
		got, err := ServerURL(server)
		if err != nil || got.String() != want {
			t.Fatalf("%s: %v %v", server, got, err)
		}
	}
	for _, server := range []string{"", ".", "https://brmbl.dev", "user@brmbl.dev", "brmbl.dev/path", "brmbl.dev?token=private", "brmbl.dev#fragment", "brmbl.dev:0", "brmbl.dev:65536", "::1"} {
		if _, err := ServerURL(server); err == nil {
			t.Errorf("accepted invalid origin %q", server)
		}
	}
}
