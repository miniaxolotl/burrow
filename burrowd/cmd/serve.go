package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
	serveCmd.Flags().String("domain", "mawa.dev", "Domain for tunnel URLs")
	serveCmd.Flags().String("redis-url", "localhost:6379", "Redis connection URL")
	serveCmd.Flags().String("secret", "", "Authentication secret")
	serveCmd.Flags().Bool("tls", false, "Generate tunnel URLs with https:// (set when server is behind HTTPS)")

	viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
	viper.BindPFlag("domain", serveCmd.Flags().Lookup("domain"))
	viper.BindPFlag("redis-url", serveCmd.Flags().Lookup("redis-url"))
	viper.BindPFlag("secret", serveCmd.Flags().Lookup("secret"))
	viper.BindPFlag("tls", serveCmd.Flags().Lookup("tls"))
}

func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	if _, hasPass := u.User.Password(); hasPass {
		u.User = url.UserPassword(u.User.Username(), "***")
	}
	return u.String()
}

func runServe(cmd *cobra.Command, args []string) error {
	port := viper.GetString("port")
	if !cmd.Flags().Changed("port") {
		if _, ok := os.LookupEnv("BURROW_PORT"); !ok {
			if p := os.Getenv("PORT"); p != "" {
				port = p
			}
		}
	}

	domain := viper.GetString("domain")
	secure := viper.GetBool("tls")

	redisURL := viper.GetString("redis-url")
	if !cmd.Flags().Changed("redis-url") {
		if _, ok := os.LookupEnv("BURROW_REDIS_URL"); !ok {
			if u := os.Getenv("REDIS_URL"); u != "" {
				redisURL = u
			}
		}
	}

	secret := viper.GetString("secret")

	if secret == "" {
		return fmt.Errorf("secret is required: set --secret flag or BURROW_SECRET env var")
	}

	pidFile := internal.ConfigDir() + "/pid"
	if err := os.MkdirAll(internal.ConfigDir(), 0700); err == nil {
		os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0600)
		defer os.Remove(pidFile)
	}

	redis, err := internal.NewRedisClient(redisURL)
	if err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}
	defer redis.Close()

	registry := internal.NewTunnelRegistry(redis, domain, secure)

	server := internal.NewServer(registry, domain, secret, secure)

	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("Starting burrowd on %s\n", addr)
	fmt.Printf("Domain: %s (TLS: %v)\n", domain, secure)
	fmt.Printf("Redis: %s\n", redactURL(redisURL))

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	startErr := make(chan error, 1)
	go func() {
		if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
			startErr <- err
		}
	}()

	select {
	case err := <-startErr:
		return fmt.Errorf("server failed to start: %w", err)
	case <-sigCh:
	}

	fmt.Println("\nShutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(ctx)
}
