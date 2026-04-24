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
	fmt.Println("TUNNEL ID              PORT    LATENCY    RECONNECTS")
	fmt.Println("arcane-dragon-xorn    3000    12ms       0")
	fmt.Println("shadow-lich-umbra      8080    8ms        1")
	return nil
}
