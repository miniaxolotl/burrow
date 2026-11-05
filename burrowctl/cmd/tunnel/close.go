package tunnel

import (
	"burrow/burrowctl/internal/clioutput"
	"burrow/burrowctl/internal/config"
	"fmt"
	"github.com/spf13/viper"

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
	server := viper.GetString("server")
	owner, err := resolveToken(cmd, server)
	if err != nil {
		return err
	}
	if err := config.MarkSaved(server, owner, tunnelID, "uncertain"); err != nil {
		return err
	}
	if err := client.DeleteTunnel(tunnelID); err != nil {
		return err
	}
	if err := config.MarkSaved(server, owner, tunnelID, "closed"); err != nil {
		return err
	}

	if output := clioutput.FromCommand(cmd); output.Structured() {
		return output.Emit("result", tunnelID, struct {
			Closed bool `json:"closed"`
		}{true})
	}
	_, err = fmt.Fprintf(clioutput.FromCommand(cmd), "Tunnel %s closed\n", tunnelID)
	return err
}
