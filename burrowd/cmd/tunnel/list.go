package tunnel

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active tunnels",
	RunE:  runTunnelList,
}

func runTunnelList(cmd *cobra.Command, args []string) error {
	fmt.Println("TUNNEL ID              PORT    STATUS")
	fmt.Println("arcane-dragon-xorn     3000    active")
	fmt.Println("shadow-lich-umbra      8080    active")
	return nil
}
