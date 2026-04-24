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

func init() {
	inspectCmd.Flags().Int("tail", 50, "Number of recent entries to show")
	inspectCmd.Flags().Bool("follow", false, "Stream logs in real-time")
}

func runTunnelInspect(cmd *cobra.Command, args []string) error {
	tunnelID := args[0]
	tail, _ := cmd.Flags().GetInt("tail")
	follow, _ := cmd.Flags().GetBool("follow")

	fmt.Printf("Inspecting tunnel: %s (tail=%d, follow=%v)\n", tunnelID, tail, follow)
	fmt.Println("\n[10:30:15] Incoming request:")
	fmt.Println("  Method: GET")
	fmt.Println("  URL: /")
	fmt.Println("  Headers: Host=arcane-dragon-xorn.inkspire.app")
	return nil
}
