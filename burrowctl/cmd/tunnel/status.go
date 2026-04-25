package tunnel

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show tunnel connection status",
	RunE:  runTunnelStatus,
}

func runTunnelStatus(cmd *cobra.Command, args []string) error {
	tunnels, err := fetchTunnels()
	if err != nil {
		return err
	}
	if len(tunnels) == 0 {
		fmt.Println("No active tunnels")
		return nil
	}
	printTunnels(tunnels)
	fmt.Println("-----------------------------------------------------------------------------------------------------------------------------")
	return nil
}
