package tunnel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"burrow/protocol"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	inspectTail   int
	inspectFollow bool
)

var inspectCmd = &cobra.Command{
	Use:   "inspect [tunnel_id]",
	Short: "View tunnel traffic logs",
	Args:  cobra.ExactArgs(1),
	RunE:  runTunnelInspect,
}

func init() {
	inspectCmd.Flags().IntVar(&inspectTail, "tail", 0, "Show only the last N entries (0 = all)")
	inspectCmd.Flags().BoolVar(&inspectFollow, "follow", false, "Poll for new entries")
}

func runTunnelInspect(cmd *cobra.Command, args []string) error {
	tunnelID := args[0]
	server := viper.GetString("server")
	token := resolveToken()

	if inspectFollow {
		return followLogs(server, token, tunnelID)
	}

	return printLogs(server, token, tunnelID)
}

func fetchLogs(server, token, tunnelID string) ([]*protocol.TunnelLog, error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET",
		fmt.Sprintf("%s://%s/logs/%s", httpScheme(), server, tunnelID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("X-Tunnel-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to contact server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}

	var logs []*protocol.TunnelLog
	if err := json.NewDecoder(resp.Body).Decode(&logs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return logs, nil
}

func printLogs(server, token, tunnelID string) error {
	logs, err := fetchLogs(server, token, tunnelID)
	if err != nil {
		return err
	}

	if inspectTail > 0 && len(logs) > inspectTail {
		logs = logs[len(logs)-inspectTail:]
	}

	if len(logs) == 0 {
		fmt.Println("No log entries")
		return nil
	}

	printLogTable(logs)
	return nil
}

func followLogs(server, token, tunnelID string) error {
	seen := 0
	for {
		logs, err := fetchLogs(server, token, tunnelID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			time.Sleep(time.Second)
			continue
		}

		if inspectTail > 0 && len(logs) > inspectTail {
			logs = logs[len(logs)-inspectTail:]
		}

		if len(logs) > seen {
			printLogTable(logs[seen:])
			seen = len(logs)
		}

		time.Sleep(time.Second)
	}
}

func printLogTable(logs []*protocol.TunnelLog) {
	for _, log := range logs {
		ts := log.Timestamp.Format("15:04:05.000")
		fmt.Printf("%s  %s  %-40s  %d  %s\n",
			ts,
			log.Method,
			log.Path,
			log.StatusCode,
			log.Duration,
		)
	}
}
