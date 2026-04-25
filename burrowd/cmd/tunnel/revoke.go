package tunnel

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var revokeCmd = &cobra.Command{
	Use:   "revoke [tunnel_id]",
	Short: "Force-close a specific tunnel",
	Args:  cobra.ExactArgs(1),
	RunE:  runTunnelRevoke,
}

func runTunnelRevoke(cmd *cobra.Command, args []string) error {
	tunnelID := args[0]
	server := viper.GetString("server")

	resp, err := apiDo("DELETE", fmt.Sprintf("%s://%s/tunnel/%s", scheme(), server, tunnelID))
	if err != nil {
		return fmt.Errorf("failed to contact server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("server returned %s", resp.Status)
	}

	fmt.Printf("Tunnel %s revoked\n", tunnelID)
	return nil
}
