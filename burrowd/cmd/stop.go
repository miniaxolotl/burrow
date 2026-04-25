package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"burrow/burrowd/internal"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the server gracefully",
	RunE:  runStop,
}

func init() {}

func runStop(cmd *cobra.Command, args []string) error {
	pidFile := internal.ConfigDir() + "/pid"
	data, err := os.ReadFile(pidFile)
	if os.IsNotExist(err) {
		return fmt.Errorf("server not running (no PID file found)")
	}
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	pid := string(data)
	fmt.Printf("Stopping server (PID: %s)\n", pid)

	pidInt := 0
	if _, err := fmt.Sscanf(pid, "%d", &pidInt); err != nil {
		return fmt.Errorf("invalid PID in file: %w", err)
	}

	process, err := os.FindProcess(pidInt)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGCHLD)
	go func() {
		<-ch
		os.Remove(pidFile)
	}()

	fmt.Println("Server stopped")
	return nil
}
