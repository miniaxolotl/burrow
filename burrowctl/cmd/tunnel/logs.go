package tunnel

import (
	"fmt"
	"io"
	"time"

	"burrow/burrowctl/internal/clioutput"
	"burrow/burrowctl/internal/inspection"
	"burrow/protocol"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <id>",
	Short: "View your tunnel's request logs",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			return err
		}
		_, err := readFilters(cmd, "server")
		return err
	},
	RunE: runLogs,
}

func init() {
	logsCmd.Flags().Int("tail", 0, "Show only the last N entries (0 = all)")
	logsCmd.Flags().Bool("follow", false, "Follow new requests")
	addFilters(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	tail, _ := cmd.Flags().GetInt("tail")
	if tail < 0 {
		return clioutput.Invalid(fmt.Errorf("tail must not be negative"))
	}
	query, err := readFilters(cmd, "server")
	if err != nil {
		return clioutput.Invalid(err)
	}
	follow, _ := cmd.Flags().GetBool("follow")
	client, err := configuredClient(cmd)
	if err != nil {
		return err
	}
	var seen map[string]struct{}
	output := clioutput.FromCommand(cmd)
	for {
		logs, err := client.GetLogs(args[0])
		if err != nil {
			return err
		}
		retained := logs
		logs = query.Apply(retained)
		if output.Structured() && !follow {
			if tail > 0 && tail < len(logs) {
				logs = logs[len(logs)-tail:]
			}
			if logs == nil {
				logs = []*protocol.TunnelLog{}
			}
			return output.Emit("result", args[0], requestsData(logs))
		}
		if output.Structured() {
			batch, _ := selectLogBatch(logs, seen, tail)
			_, current := selectLogBatch(retained, seen, 0)
			if err := output.Emit("snapshot", args[0], map[string]any{"source": "server_logs", "matching": len(logs), "retained": len(retained), "observed_at": time.Now().UTC(), "continuity": "bounded_polling_possible_gaps"}); err != nil {
				return err
			}
			for _, entry := range batch {
				if err := output.Emit("log", args[0], requestData(entry)); err != nil {
					return err
				}
			}
			seen = current
		} else {
			_, err = printLogBatch(output, logs, seen, tail)
			_, seen = selectLogBatch(retained, seen, 0)
			if err != nil {
				return err
			}
		}
		if !follow {
			if _, err := fmt.Fprintf(output, "%d matching / %d retained · server logs · bounded window\n", len(logs), len(retained)); err != nil {
				return err
			}
			if len(logs) == 0 {
				_, err := fmt.Fprintln(output, "No log entries")
				return err
			}
			return nil
		}
		select {
		case <-cmd.Context().Done():
			if output.Structured() {
				return output.Emit("stopped", "", map[string]string{"reason": "cancelled"})
			}
			return nil
		case <-time.After(time.Second):
		}
	}
}

func printLogBatch(out io.Writer, logs []*protocol.TunnelLog, seen map[string]struct{}, tail int) (map[string]struct{}, error) {
	batch, current := selectLogBatch(logs, seen, tail)
	for _, entry := range batch {
		if _, err := fmt.Fprintf(out, "%s  %-15s  %-6s  %-40s  %s\n", entry.Timestamp.Format("15:04:05.000"), inspection.Safe(entry.IP), inspection.Safe(entry.Method), inspection.Safe(entry.Path), inspection.Safe(entry.Duration)); err != nil {
			return current, err
		}
	}
	return current, nil
}

func selectLogBatch(logs []*protocol.TunnelLog, seen map[string]struct{}, tail int) ([]*protocol.TunnelLog, map[string]struct{}) {
	start := 0
	if seen == nil && tail > 0 {
		start = max(0, len(logs)-tail)
	}
	current := make(map[string]struct{}, len(logs))
	batch := make([]*protocol.TunnelLog, 0, len(logs))
	for i, entry := range logs {
		// Remember the entire snapshot, including entries omitted by --tail.
		// Retaining only this rolling buffer bounds memory during long follows.
		key := entry.Generation + "\x00" + entry.ID
		current[key] = struct{}{}
		if _, printed := seen[key]; printed || i < start {
			continue
		}
		batch = append(batch, entry)
	}
	return batch, current
}
