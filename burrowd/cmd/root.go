package cmd

import (
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

func SetVersion(v string) {
	rootCmd.Version = v
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// BURROW_* env vars map to config keys (hyphens become underscores).
	viper.SetEnvPrefix("BURROW")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(auth.AuthCmd)
	rootCmd.AddCommand(tunnel.TunnelCmd)
	rootCmd.AddCommand(stopCmd)
}
