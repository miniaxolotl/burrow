package tunnel

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show tunnel connection status and latency",
	RunE:  runTunnelStatus,
}

func runTunnelStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("not implemented")
	return nil
}
