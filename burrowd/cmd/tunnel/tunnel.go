package tunnel

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var TunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Tunnel management commands",
	Long:  `Manage tunnels on the burrow server.`,
}

func init() {
	TunnelCmd.PersistentFlags().String("server", "localhost:25701", "Burrow server address")
	TunnelCmd.PersistentFlags().String("token", "", "Authentication token")
	TunnelCmd.PersistentFlags().Bool("tls", false, "Use TLS (https://) when connecting to the server")
	viper.BindPFlag("server", TunnelCmd.PersistentFlags().Lookup("server"))
	viper.BindPFlag("token", TunnelCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("tls", TunnelCmd.PersistentFlags().Lookup("tls"))

	TunnelCmd.AddCommand(listCmd)
	TunnelCmd.AddCommand(revokeCmd)
}
