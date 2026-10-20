// Package telemetry reports content-free operational outcomes. It never accepts
// tunnel requests, credentials, command arguments, local ports, or user content.
package telemetry

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func StateDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "burrow")
}

func Disabled(dir string) bool {
	if os.Getenv("DO_NOT_TRACK") == "1" || os.Getenv("CI") != "" || os.Getenv("BURROW_TELEMETRY") == "0" || strings.EqualFold(os.Getenv("BURROW_TELEMETRY"), "false") || dir == "" {
		return true
	}
	_, err := os.Stat(filepath.Join(dir, "telemetry-disabled"))
	return err == nil || !errors.Is(err, os.ErrNotExist)
}

// SetEnabled affects future events in running processes as well as new commands.
// Disabling clears only the telemetry identity, never owner credentials.
func SetEnabled(dir string, enabled bool) error {
	if dir == "" {
		return errors.New("telemetry preference directory unavailable")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	marker := filepath.Join(dir, "telemetry-disabled")
	if enabled {
		err := os.Remove(marker)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := os.WriteFile(marker, []byte("disabled\n"), 0600); err != nil {
		return err
	}
	err := os.Remove(filepath.Join(dir, "telemetry-id"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func installationID(dir string) string {
	if Disabled(dir) {
		return ""
	}
	path := filepath.Join(dir, "telemetry-id")
	read := func() string {
		data, err := os.ReadFile(path)
		if err != nil {
			return ""
		}
		id, err := uuid.Parse(strings.TrimSpace(string(data)))
		if err != nil {
			return ""
		}
		return id.String()
	}
	if id := read(); id != "" {
		return id
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return ""
	}
	file, err := os.CreateTemp(dir, ".telemetry-*")
	if err != nil {
		return ""
	}
	defer os.Remove(file.Name())
	if _, err = file.WriteString(uuid.NewString() + "\n"); err != nil {
		file.Close()
		return ""
	}
	if err = file.Close(); err != nil {
		return ""
	}
	// Exclusive publication prevents concurrent first-use processes from replacing
	// one another's identity. A malformed existing identity fails closed.
	if err = os.Link(file.Name(), path); err != nil && !errors.Is(err, os.ErrExist) {
		return ""
	}
	if Disabled(dir) {
		return ""
	}
	return read()
}
