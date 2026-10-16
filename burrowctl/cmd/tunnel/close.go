package tunnel

import (
	"fmt"

	"github.com/spf13/cobra"
)

var closeCmd = &cobra.Command{
	Use:   "close <id>",
	Short: "Permanently close one of your tunnels",
	Args:  cobra.ExactArgs(1),
	RunE:  runTunnelClose,
}

func runTunnelClose(cmd *cobra.Command, args []string) error {
	tunnelID := args[0]
	client, err := configuredClient(cmd)
	if err != nil {
		return err
	}
	if err := client.DeleteTunnel(tunnelID); err != nil {
		return err
	}

	fmt.Printf("Tunnel %s closed\n", tunnelID)
	return nil
}
