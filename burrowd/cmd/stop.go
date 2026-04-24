package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the server gracefully",
	RunE:  runStop,
}

func init() {}

func runStop(cmd *cobra.Command, args []string) error {
	pidFile := getConfigDir() + "/pid"
	data, err := os.ReadFile(pidFile)
	if os.IsNotExist(err) {
		return fmt.Errorf("server not running (no PID file found)")
	}
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}
	fmt.Printf("Stopping server (PID: %s)\n", string(data))
	return nil
}
