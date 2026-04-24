package tunnel

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List active tunnels",
	RunE:  runTunnelList,
}

func runTunnelList(cmd *cobra.Command, args []string) error {
	fmt.Println("TUNNEL ID              PORT    URL                                           STATUS")
	fmt.Println("arcane-dragon-xorn    3000    https://arcane-dragon-xorn.inkspire.app       active")
	fmt.Println("shadow-lich-umbra      8080    https://shadow-lich-umbra.inkspire.app       active")
	return nil
}
