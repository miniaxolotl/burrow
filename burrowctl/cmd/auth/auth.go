package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  `Manage authentication tokens for burrowctl.`,
}

func init() {
	AuthCmd.AddCommand(loginCmd)
	AuthCmd.AddCommand(logoutCmd)
	AuthCmd.AddCommand(statusCmd)
}

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".config/burrow"
	}
	return filepath.Join(home, ".config", "burrow")
}

type clientConfig struct {
	Server   string `json:"server"`
	Token    string `json:"token"`
	Secret   string `json:"secret"`
	Domain   string `json:"domain"`
	TLS      bool   `json:"tls"`
	LogLevel string `json:"log-level"`
}

func readConfig() (*clientConfig, string, error) {
	dir := configDir()
	cfgFile := filepath.Join(dir, "client.json")
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		return nil, "", err
	}
	var cfg clientConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, "", err
	}
	return &cfg, cfgFile, nil
}

func writeConfig(cfg *clientConfig, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(path, append(b, '\n'), 0600)
}
