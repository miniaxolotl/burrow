package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"burrow/burrowctl/cmd/auth"
	"burrow/burrowctl/cmd/tunnel"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "burrowctl",
	Short: "burrowctl - Tunnel client",
	Long:  `burrowctl is the client CLI for creating tunnels to the burrow server.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(loadClientConfig)

	viper.SetEnvPrefix("BURROW")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	rootCmd.AddCommand(tunnel.TunnelCmd)
	rootCmd.AddCommand(auth.AuthCmd)
}

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".config/burrow"
	}
	return filepath.Join(home, ".config", "burrow")
}

func loadClientConfig() {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return
	}

	cfgFile := filepath.Join(dir, "client.json")
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		defaults := map[string]any{
			"server":    "localhost:25701",
			"token":     "",
			"secret":    "",
			"domain":    "burrow.mawa.dev",
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
