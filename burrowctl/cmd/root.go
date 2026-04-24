package cmd

import (
	"burrow/burrowctl/cmd/tunnel"

	"github.com/spf13/cobra"
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
	rootCmd.AddCommand(tunnel.TunnelCmd)
}
