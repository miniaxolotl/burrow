package tunnel

import (
	"context"
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
	token := viper.GetString("token")
	secure := viper.GetBool("tls")

	scheme := "http"
	if secure {
		scheme = "https"
	}

	req, err := http.NewRequestWithContext(context.Background(), "DELETE",
		fmt.Sprintf("%s://%s/tunnel/%s", scheme, server, tunnelID), nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("X-Tunnel-Token", token)

	resp, err := http.DefaultClient.Do(req)
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
