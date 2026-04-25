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
	Short: "List active tunnels",
	RunE:  runTunnelList,
}

func fetchTunnels() ([]*protocol.TunnelInfo, error) {
	server := viper.GetString("server")

	resp, err := apiDo("GET", fmt.Sprintf("%s://%s/tunnels", httpScheme(), server))
	if err != nil {
		return nil, fmt.Errorf("failed to contact server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}

	var tunnels []*protocol.TunnelInfo
	if err := json.NewDecoder(resp.Body).Decode(&tunnels); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return tunnels, nil
}

func printTunnels(tunnels []*protocol.TunnelInfo) {
	fmt.Printf("%-30s %-8s %-45s %s\n", "TUNNEL ID", "PORT", "URL", "STATUS")
	for _, t := range tunnels {
		fmt.Printf("%-30s %-8d %-45s %s\n", t.TunnelID, t.Port, t.URL, t.Status)
	}
}

func runTunnelList(cmd *cobra.Command, args []string) error {
	tunnels, err := fetchTunnels()
	if err != nil {
		return err
	}
	if len(tunnels) == 0 {
		fmt.Println("No active tunnels")
		return nil
	}
	printTunnels(tunnels)
	return nil
}
