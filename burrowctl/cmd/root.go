package cmd

import (
	"strings"

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
	viper.SetEnvPrefix("BURROW")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	rootCmd.AddCommand(tunnel.TunnelCmd)
}
