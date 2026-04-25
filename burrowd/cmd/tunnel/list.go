package tunnel

import (
	"encoding/json"
	"fmt"
	"net/http"

	"burrow/protocol"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active tunnels",
	RunE:  runTunnelList,
}

func runTunnelList(cmd *cobra.Command, args []string) error {
	server := viper.GetString("server")

	resp, err := apiDo("GET", fmt.Sprintf("%s://%s/tunnels", scheme(), server))
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

	fmt.Printf("%-30s %-8s %s\n", "TUNNEL ID", "PORT", "URL")
	for _, t := range tunnels {
		fmt.Printf("%-30s %-8d %s\n", t.TunnelID, t.Port, t.URL)
	}
	return nil
}
