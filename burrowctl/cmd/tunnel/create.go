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
	createCmd.Flags().String("server", "", "Burrow server address")
	createCmd.Flags().String("token", "", "Authentication token")
	createCmd.Flags().String("domain", "", "Domain used to build tunnel URLs")

	viper.BindPFlag("port", createCmd.Flags().Lookup("port"))
	viper.BindPFlag("server", createCmd.Flags().Lookup("server"))
	viper.BindPFlag("token", createCmd.Flags().Lookup("token"))
	viper.BindPFlag("domain", createCmd.Flags().Lookup("domain"))

	viper.SetDefault("server", "localhost:25701")
	viper.SetDefault("domain", "localhost")
}

func runTunnelCreate(cmd *cobra.Command, args []string) error {
	ports := viper.GetIntSlice("port")
	server := viper.GetString("server")
	token := viper.GetString("token")
	domain := viper.GetString("domain")
	if domain == "" {
		domain = os.Getenv("BURROW_DOMAIN")
	}

	if server == "" {
		hostname := os.Getenv("BURROW_HOSTNAME")
		if hostname == "" {
			hostname = "localhost"
		}
		port := os.Getenv("BURROW_PORT")
		if port == "" {
			port = "25701"
		}
		server = fmt.Sprintf("%s:%s", hostname, port)
	}

	if token == "" {
		token = os.Getenv("BURROW_TOKEN")
	}
	if token == "" {
		token = viper.GetString("stored_token")
	}

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
