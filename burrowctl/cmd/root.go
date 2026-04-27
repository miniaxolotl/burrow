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

func SetVersion(v string) {
	rootCmd.Version = v
}

func init() {
	cobra.OnInitialize(loadClientConfig)

	viper.SetEnvPrefix("BURROW")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	rootCmd.AddCommand(tunnel.TunnelCmd)
	rootCmd.AddCommand(auth.AuthCmd)
}

func loadClientConfig() {
	dir := auth.ConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return
	}

	cfgFile := filepath.Join(dir, "client.json")
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		defaults := map[string]any{
			"server": "burrow.mawa.dev",
			"token":  "",
			"secret": "",
			"domain": "burrow.mawa.dev",
		}
		if b, err := json.MarshalIndent(defaults, "", "  "); err == nil {
			os.WriteFile(cfgFile, append(b, '\n'), 0600)
		}
	}

	viper.SetConfigFile(cfgFile)
	viper.SetConfigType("json")
	viper.ReadInConfig()
}
