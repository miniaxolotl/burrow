package tunnel

import (
	"fmt"

	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect [tunnel_id]",
	Short: "View tunnel traffic logs",
	Args:  cobra.ExactArgs(1),
	RunE:  runTunnelInspect,
}

func runTunnelInspect(cmd *cobra.Command, args []string) error {
	fmt.Println("not implemented")
	return nil
}
