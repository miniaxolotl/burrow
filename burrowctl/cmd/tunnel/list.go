package tunnel

import (
	"fmt"

	"burrow/protocol"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List your active tunnels and count",
	Args:  cobra.NoArgs,
	RunE:  runTunnelList,
}

func printTunnels(tunnels []*protocol.TunnelInfo) {
	fmt.Printf("%-30s %-8s %-45s %s\n", "TUNNEL ID", "PORT", "URL", "STATUS")
	for _, t := range tunnels {
		fmt.Printf("%-30s %-8d %-45s %s\n", t.TunnelID, t.Port, t.URL, t.Status)
	}
}

func runTunnelList(cmd *cobra.Command, args []string) error {
	client, err := configuredClient(cmd)
	if err != nil {
		return err
	}
	result, err := client.RemoteTunnels()
	if err != nil {
		return err
	}
	fmt.Printf("%d active tunnel(s)\n", result.Count)
	if len(result.Tunnels) == 0 {
		return nil
	}
	printTunnels(result.Tunnels)
	return nil
}
