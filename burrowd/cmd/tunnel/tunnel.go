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
	TunnelCmd.PersistentFlags().String("redis-url", "localhost:6379", "Redis connection URL")
	TunnelCmd.PersistentFlags().String("domain", "inkspire.app", "Domain for tunnel URLs")
	viper.BindPFlag("redis-url", TunnelCmd.PersistentFlags().Lookup("redis-url"))
	viper.BindPFlag("domain", TunnelCmd.PersistentFlags().Lookup("domain"))

	TunnelCmd.AddCommand(listCmd)
	TunnelCmd.AddCommand(revokeCmd)
}
