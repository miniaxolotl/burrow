package tunnel

import (
	"fmt"
	"io"
	"time"

	"burrow/protocol"

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
	var seen map[string]struct{}
	for {
		logs, err := client.GetLogs(args[0])
		if err != nil {
			return err
		}
		seen = printLogBatch(cmd.OutOrStdout(), logs, seen, tail)
		if !follow {
			if len(logs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No log entries")
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

func printLogBatch(out io.Writer, logs []*protocol.TunnelLog, seen map[string]struct{}, tail int) map[string]struct{} {
	start := 0
	if seen == nil && tail > 0 {
		start = max(0, len(logs)-tail)
	}
	current := make(map[string]struct{}, len(logs))
	for i, entry := range logs {
		// Remember the entire snapshot, including entries omitted by --tail.
		// Retaining only this rolling buffer bounds memory during long follows.
		current[entry.ID] = struct{}{}
		if _, printed := seen[entry.ID]; printed || i < start {
			continue
		}
		fmt.Fprintf(out, "%s  %-15s  %-6s  %-40s  %s\n", entry.Timestamp.Format("15:04:05.000"), entry.IP, entry.Method, entry.Path, entry.Duration)
	}
	return current
}
