package tunnel

import (
	"context"
	"fmt"
	"net/http"
	"os"

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
	token := viper.GetString("token")
	if token == "" {
		token = os.Getenv("BURROW_TOKEN")
	}

	req, err := http.NewRequestWithContext(context.Background(), "DELETE",
		fmt.Sprintf("http://%s/tunnel/%s", server, tunnelID), nil)
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

	fmt.Printf("Tunnel %s closed\n", tunnelID)
	return nil
}
