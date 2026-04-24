package tunnel

import (
	"fmt"

	"github.com/spf13/cobra"
)

var closeCmd = &cobra.Command{
	Use:   "close [tunnel_id]",
	Short: "Close a specific tunnel",
	Args:  cobra.ExactArgs(1),
	RunE:  runTunnelClose,
}

func runTunnelClose(cmd *cobra.Command, args []string) error {
	tunnelID := args[0]
	fmt.Printf("Closing tunnel: %s\n", tunnelID)
	return nil
}
