package tunnel

import (
	"fmt"
	"os"
	"strconv"

	"burrow/burrowctl/cmd/tui"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func runTunnelCreate(cmd *cobra.Command, args []string) error {
	ports := make([]int, 0, len(args))
	for _, arg := range args {
		port, err := strconv.Atoi(arg)
		if err != nil {
			return fmt.Errorf("invalid port %q", arg)
		}
		ports = append(ports, port)
	}
	seen := make(map[int]bool)
	for _, port := range ports {
		if port < 1 || port > 65535 {
			return fmt.Errorf("port must be between 1 and 65535: %d", port)
		}
		if seen[port] {
			return fmt.Errorf("port specified more than once: %d", port)
		}
		seen[port] = true
	}
	server := viper.GetString("server")
	client, err := configuredClient(cmd)
	if err != nil {
		return err
	}

	defer client.Close()

	failures := 0
	for _, port := range ports {
		if _, err := client.CreateTunnel(cmd.Context(), uint16(port)); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create tunnel for port %d: %v\n", port, err)
			failures++
		}
	}

	if failures == len(ports) {
		return fmt.Errorf("all %d tunnel(s) failed to create", len(ports))
	}

	return tui.RunWithClient(client, server)
}
