package tunnel

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

	fmt.Printf("Creating tunnels to %s...\n", server)

	client := internal.NewClient(server, token, domain)
	defer client.Close()

	ctx := context.Background()
	urls := make([]string, 0, len(ports))

	for _, port := range ports {
		url, err := client.CreateTunnel(ctx, uint16(port))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create tunnel for port %d: %v\n", port, err)
			continue
		}
		urls = append(urls, url)
		fmt.Printf("  %s -> localhost:%d\n", url, port)
	}

	if len(urls) == 0 {
		return fmt.Errorf("no tunnels could be created")
	}

	fmt.Println("\nPress Ctrl+C to close tunnels")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nClosing tunnels...")
	for _, port := range ports {
		client.CloseTunnel(uint16(port))
	}

	return nil
}
