package config

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAutomaticOwnerCredentialPersistenceAndScope(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	directory := filepath.Join(os.Getenv("HOME"), ".config", "burrow", "owners")
	const processes = 12
	var wg sync.WaitGroup
	results := make(chan string, processes)
	for range processes {
		wg.Go(func() {
			token, err := OwnerToken("brmbl.dev")
			if err != nil {
				t.Error(err)
				return
			}
			results <- token
		})
	}
	wg.Wait()
	close(results)
	var first string
	for token := range results {
		if first == "" {
			first = token
		}
		if token != first {
			t.Fatal("concurrent first use created different owners")
		}
	}
	raw, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil || len(raw) != 32 {
		t.Fatal("owner credential must contain 256 random bits")
	}
	for _, server := range []string{"brmbl.dev", "BRMBL.DEV:443", "brmbl.dev."} {
		token, err := OwnerToken(server)
		if err != nil || token != first {
			t.Fatalf("identity changed for %s: %v", server, err)
		}
	}
	for _, server := range []string{"staging.brmbl.dev", "brmbl.dev:8443"} {
		token, err := OwnerToken(server)
		if err != nil || token == first {
			t.Fatalf("credential not scoped to %s: %v", server, err)
		}
	}
	files, err := filepath.Glob(filepath.Join(directory, "*.token"))
	if err != nil || len(files) != 3 {
		t.Fatalf("unexpected owner files: %v %v", files, err)
	}
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("credential permissions: %o", info.Mode().Perm())
		}
	}
	dir, err := os.Stat(directory)
	if err != nil || dir.Mode().Perm() != 0700 {
		t.Fatal("owner directory must be private")
	}
	// Corruption fails closed instead of silently changing the user's identity.
	for _, file := range files {
		if err := os.WriteFile(file, []byte("broken"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := OwnerToken("brmbl.dev"); err == nil {
		t.Fatal("corrupt owner credential accepted")
	}
}

func TestSavedCredentialsStayWithTheirServer(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	file := filepath.Join(os.Getenv("HOME"), ".config", "burrow", "client.json")
	cfg := &clientConfig{Server: "brmbl.dev", Domain: "brmbl.dev", Token: "saved-token"}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := writePrivateFile(file, data); err != nil {
		t.Fatal(err)
	}
	token, err := StoredToken("brmbl.dev:443")
	if err != nil || token != cfg.Token {
		t.Fatal("lost existing server credentials")
	}
	token, err = StoredToken("staging.brmbl.dev")
	if err != nil || token != "" {
		t.Fatal("sent saved credentials to a different server")
	}
	original, err := OwnerToken("brmbl.dev")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OwnerToken("staging.brmbl.dev"); err != nil {
		t.Fatal(err)
	}
	restored, err := OwnerToken("brmbl.dev")
	if err != nil || restored != original {
		t.Fatal("switching servers lost per-server owner identity")
	}
}
