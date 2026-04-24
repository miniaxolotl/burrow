package tunnel

import (
	"github.com/spf13/cobra"
)

var TunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Tunnel management commands",
	Long:  `Manage tunnels on the burrow server.`,
}

func init() {
	TunnelCmd.AddCommand(listCmd)
	TunnelCmd.AddCommand(revokeCmd)
}
