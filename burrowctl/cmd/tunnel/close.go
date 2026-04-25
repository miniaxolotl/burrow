package tunnel

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var closeCmd = &cobra.Command{
	Use:   "close [tunnel_id]",
	Short: "Close a specific tunnel",
	Args:  cobra.ExactArgs(1),
	RunE:  runTunnelClose,
}

func runTunnelClose(cmd *cobra.Command, args []string) error {
	tunnelID := args[0]
	server := viper.GetString("server")

	resp, err := apiDo("DELETE", fmt.Sprintf("%s://%s/tunnel/%s", httpScheme(), server, tunnelID))
	if err != nil {
		return fmt.Errorf("failed to contact server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("server returned %s", resp.Status)
	}

	fmt.Printf("Tunnel %s closed\n", tunnelID)
	return nil
}
