package tunnel

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <id>",
	Short: "View your tunnel's request logs",
	Args:  cobra.ExactArgs(1),
	RunE:  runLogs,
}

func init() {
	logsCmd.Flags().Int("tail", 0, "Show only the last N entries (0 = all)")
	logsCmd.Flags().Bool("follow", false, "Follow new requests")
}

func runLogs(cmd *cobra.Command, args []string) error {
	tail, _ := cmd.Flags().GetInt("tail")
	if tail < 0 {
		return fmt.Errorf("tail must not be negative")
	}
	follow, _ := cmd.Flags().GetBool("follow")
	client, err := configuredClient(cmd)
	if err != nil {
		return err
	}
	var last time.Time
	for {
		logs, err := client.GetLogs(args[0])
		if err != nil {
			return err
		}
		if last.IsZero() && tail > 0 && len(logs) > tail {
			logs = logs[len(logs)-tail:]
		}
		newest := last
		for _, entry := range logs {
			if !entry.Timestamp.After(last) {
				continue
			}
			fmt.Printf("%s  %-15s  %-6s  %-40s  %s\n", entry.Timestamp.Format("15:04:05.000"), entry.IP, entry.Method, entry.Path, entry.Duration)
			if entry.Timestamp.After(newest) {
				newest = entry.Timestamp
			}
		}
		last = newest
		if !follow {
			if len(logs) == 0 {
				fmt.Println("No log entries")
			}
			return nil
		}
		select {
		case <-cmd.Context().Done():
			return nil
		case <-time.After(time.Second):
		}
	}
}
