package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"burrow/burrowctl/internal"
)

// OwnerToken creates an account-free identity for this server. Publishing with
// link is atomic and exclusive: concurrent CLI processes cannot observe a
// partial credential or choose different identities on first use.
func OwnerToken(server string) (string, error) {
	scope, err := internal.ServerURL(server)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(scope.String()))
	base, err := configDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "owners")
	if err := ensurePrivateDir(dir); err != nil {
		return "", err
	}
	path := filepath.Join(dir, hex.EncodeToString(digest[:])+".token")
	read := func() (string, error) {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		token := strings.TrimSpace(string(data))
		decoded, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil || len(decoded) != 32 {
			return "", fmt.Errorf("invalid owner credential in %s", path)
		}
		if err := os.Chmod(path, 0600); err != nil {
			return "", err
		}
		return token, nil
	}
	if token, err := read(); !errors.Is(err, os.ErrNotExist) {
		return token, err
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(secret[:])
	if err := writePrivateFile(path, []byte(token+"\n")); err != nil {
		return "", err
	}
	return read()
}

// StoredToken never reuses a saved token for a --server override.
func StoredToken(server string) (string, error) {
	cfg, _, err := readConfig()
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	want, err := internal.ServerURL(server)
	if err != nil {
		return "", err
	}
	stored, err := internal.ServerURL(cfg.Server)
	if err != nil || stored.String() != want.String() {
		return "", nil
	}
	return cfg.Token, nil
}
