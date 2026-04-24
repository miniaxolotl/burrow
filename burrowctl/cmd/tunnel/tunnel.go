package tunnel

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var TunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Tunnel management commands",
	Long:  `Create and manage tunnels.`,
}

func init() {
	TunnelCmd.PersistentFlags().String("server", "localhost:25701", "Burrow server address")
	TunnelCmd.PersistentFlags().String("token", "", "Authentication token")
	TunnelCmd.PersistentFlags().String("domain", "localhost", "Domain for tunnel URLs")
	viper.BindPFlag("server", TunnelCmd.PersistentFlags().Lookup("server"))
	viper.BindPFlag("token", TunnelCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("domain", TunnelCmd.PersistentFlags().Lookup("domain"))

	TunnelCmd.AddCommand(createCmd)
	TunnelCmd.AddCommand(listCmd)
	TunnelCmd.AddCommand(statusCmd)
	TunnelCmd.AddCommand(inspectCmd)
	TunnelCmd.AddCommand(closeCmd)
}
