package tunnel

import (
	"context"
	"fmt"
	"os"

	"burrow/burrowctl/cmd/tui"
	"burrow/burrowctl/internal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create tunnels for specified ports",
	RunE:  runTunnelCreate,
}

func init() {
	createCmd.Flags().IntSlice("port", []int{}, "Local port to tunnel (can be specified multiple times)")
	viper.BindPFlag("port", createCmd.Flags().Lookup("port"))
}

func runTunnelCreate(cmd *cobra.Command, args []string) error {
	ports := viper.GetIntSlice("port")
	server := viper.GetString("server")
	domain := viper.GetString("domain")
	token := resolveToken()

	if len(ports) == 0 {
		return fmt.Errorf("at least one port is required")
	}

	client := internal.NewClient(server, token, domain, secureTLS())
	defer client.Close()

	ctx := context.Background()
	for _, port := range ports {
		if _, err := client.CreateTunnel(ctx, uint16(port)); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create tunnel for port %d: %v\n", port, err)
		}
	}

	return tui.RunWithClient(client, server)
}
