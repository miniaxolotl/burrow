package tunnel

import (
	"fmt"
	"io"

	"burrow/burrowctl/internal/clioutput"
	"burrow/protocol"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List your active tunnels and count",
	Args:  cobra.NoArgs,
	RunE:  runTunnelList,
}

func printTunnels(out io.Writer, tunnels []*protocol.TunnelInfo) error {
	if _, err := fmt.Fprintf(out, "%-30s %-8s %-45s %s\n", "TUNNEL ID", "PORT", "URL", "STATUS"); err != nil {
		return err
	}
	for _, t := range tunnels {
		if _, err := fmt.Fprintf(out, "%-30s %-8d %-45s %s\n", t.TunnelID, t.Port, t.URL, t.Status); err != nil {
			return err
		}
	}
	return nil
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
	if result.Tunnels == nil {
		result.Tunnels = []*protocol.TunnelInfo{}
	}
	if output := clioutput.FromCommand(cmd); output.Structured() {
		return output.Emit("result", "", listData(result))
	}
	output := clioutput.FromCommand(cmd)
	if _, err := fmt.Fprintf(output, "%d active tunnel(s)\n", result.Count); err != nil {
		return err
	}
	if len(result.Tunnels) == 0 {
		return nil
	}
	return printTunnels(output, result.Tunnels)
}
