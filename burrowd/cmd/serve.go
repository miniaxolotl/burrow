package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"burrow/burrowd/internal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the tunnel server",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().String("port", "25701", "Port to listen on")
	serveCmd.Flags().String("hostname", "localhost", "Hostname for tunnel URLs")
	serveCmd.Flags().String("domain", "inkspire.app", "Domain for tunnel URLs")
	serveCmd.Flags().String("redis-url", "localhost:6379", "Redis connection URL")
	serveCmd.Flags().String("secret", "", "Authentication secret")

	viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
	viper.BindPFlag("hostname", serveCmd.Flags().Lookup("hostname"))
	viper.BindPFlag("domain", serveCmd.Flags().Lookup("domain"))
	viper.BindPFlag("redis-url", serveCmd.Flags().Lookup("redis-url"))
	viper.BindPFlag("secret", serveCmd.Flags().Lookup("secret"))
}

func runServe(cmd *cobra.Command, args []string) error {
	port := viper.GetString("port")
	hostname := viper.GetString("hostname")
	domain := viper.GetString("domain")
	redisURL := viper.GetString("redis-url")
	secret := viper.GetString("secret")

	if secret == "" {
		return fmt.Errorf("secret is required: set --secret flag or BURROW_SECRET env var")
	}

	pidFile := getConfigDir() + "/pid"
	if err := os.MkdirAll(getConfigDir(), 0700); err == nil {
		os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0600)
		defer os.Remove(pidFile)
	}

	redis, err := internal.NewRedisClient(redisURL)
	if err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}
	defer redis.Close()

	registry := internal.NewTunnelRegistry(redis, domain)

	server := internal.NewServer(registry, domain, secret)

	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("Starting burrowd on %s\n", addr)
	fmt.Printf("Hostname: %s\n", hostname)
	fmt.Printf("Domain: %s\n", domain)
	fmt.Printf("Redis: %s\n", redisURL)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		}
	}()

	<-sigCh
	fmt.Println("\nShutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(ctx)
}
