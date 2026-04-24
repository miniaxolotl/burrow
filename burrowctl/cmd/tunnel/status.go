package tunnel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"burrow/protocol"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show tunnel connection status",
	RunE:  runTunnelStatus,
}

func runTunnelStatus(cmd *cobra.Command, args []string) error {
	server := viper.GetString("server")
	token := resolveToken()

	req, err := http.NewRequest("GET", fmt.Sprintf("%s://%s/tunnels", httpScheme(), server), nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("X-Tunnel-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to contact server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s", resp.Status)
	}

	var tunnels []*protocol.TunnelInfo
	if err := json.NewDecoder(resp.Body).Decode(&tunnels); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(tunnels) == 0 {
		fmt.Println("No active tunnels")
		return nil
	}

	fmt.Printf("%-30s %-8s %-45s %s\n", "TUNNEL ID", "PORT", "URL", "STATUS")
	fmt.Println(strings.Repeat("-", 95))
	for _, t := range tunnels {
		fmt.Printf("%-30s %-8d %-45s %s\n", t.TunnelID, t.Port, t.URL, t.Status)
	}
	return nil
}
