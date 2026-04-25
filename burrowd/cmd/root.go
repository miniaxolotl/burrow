package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"burrow/burrowd/cmd/auth"
	"burrow/burrowd/cmd/tunnel"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "burrowd",
	Short: "burrowd - Tunnel server",
	Long:  `burrowd is the server component that handles tunnel requests from clients.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(loadServerConfig)

	// BURROW_* env vars map to config keys (hyphens become underscores).
	viper.SetEnvPrefix("BURROW")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(auth.AuthCmd)
	rootCmd.AddCommand(tunnel.TunnelCmd)
	rootCmd.AddCommand(stopCmd)
}

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".config/burrow"
	}
	return filepath.Join(home, ".config", "burrow")
}

func loadServerConfig() {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return
	}

	cfgFile := filepath.Join(dir, "server.json")
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		defaults := map[string]any{
			"port":      "25701",
			"domain":    "mawa.dev",
			"redis-url": "localhost:6379",
			"secret":    "",
			"tls":       false,
			"log-level": "info",
		}
		if b, err := json.MarshalIndent(defaults, "", "  "); err == nil {
			os.WriteFile(cfgFile, append(b, '\n'), 0600)
		}
	}

	viper.SetConfigFile(cfgFile)
	viper.SetConfigType("json")
	viper.ReadInConfig()
}

// getConfigDir returns the legacy ~/.burrow dir used for PID and token files.
func getConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".burrow"
	}
	return home + "/.burrow"
}
