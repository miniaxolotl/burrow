package cmd

import (
	"os"

	"burrow/burrowd/cmd/auth"
	"burrow/burrowd/cmd/tunnel"

	"github.com/spf13/cobra"
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
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(auth.AuthCmd)
	rootCmd.AddCommand(tunnel.TunnelCmd)
	rootCmd.AddCommand(stopCmd)
}

func getConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".burrow"
	}
	return home + "/.burrow"
}
