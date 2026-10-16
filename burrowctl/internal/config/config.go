package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"burrow/burrowctl/internal"
)

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate private config directory: %w", err)
	}
	return filepath.Join(home, ".config", "burrow"), nil
}

type clientConfig struct {
	Server string `json:"server"`
	Token  string `json:"token"`
	Domain string `json:"domain"`
}

func readConfig() (*clientConfig, string, error) {
	dir, err := configDir()
	if err != nil {
		return nil, "", err
	}
	cfgFile := filepath.Join(dir, "client.json")
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		return nil, cfgFile, err
	}
	var cfg clientConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, cfgFile, err
	}
	return &cfg, cfgFile, nil
}

func EnsureConfig() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "client.json")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	cfg := &clientConfig{Server: internal.DefaultServer, Domain: internal.DefaultServer}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}
	return path, writePrivateFile(path, append(b, '\n'))
}

func ensurePrivateDir(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.Chmod(dir, 0700)
}

// A private temporary file is fully written before publishing. Exclusive
// publication keeps concurrent first-use processes on the same credential.
func writePrivateFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := ensurePrivateDir(dir); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".burrow-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	err = os.Link(file.Name(), path)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	return err
}
