package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"burrow/burrowd/internal"

	"github.com/IBM/sarama"
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
	serveCmd.Flags().String("deployment-mode", "oss", "Deployment mode: oss or cloud")
	serveCmd.Flags().String("jwt-secret", "", "JWT signing secret (required in cloud mode)")
	serveCmd.Flags().String("postgres-url", "", "Postgres connection string")
	serveCmd.Flags().String("kafka-brokers", "", "Kafka broker addresses (comma-separated)")
	serveCmd.Flags().Int("anon-tunnel-limit", 1, "Max tunnels per IP for anonymous users (cloud mode)")
	serveCmd.Flags().Int("user-tunnel-limit", 5, "Max tunnels per free user (cloud mode)")
	serveCmd.Flags().Int("paid-tunnel-limit", 0, "Max tunnels per paid user, 0=unlimited (cloud mode)")
	serveCmd.Flags().String("stripe-secret-key", "", "Stripe API secret key")
	serveCmd.Flags().String("stripe-webhook-secret", "", "Stripe webhook signing secret")
	serveCmd.Flags().String("stripe-pro-price-id", "", "Stripe Price ID for Pro plan")

	viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
	viper.BindPFlag("domain", serveCmd.Flags().Lookup("domain"))
	viper.BindPFlag("redis-url", serveCmd.Flags().Lookup("redis-url"))
	viper.BindPFlag("secret", serveCmd.Flags().Lookup("secret"))
	viper.BindPFlag("tls", serveCmd.Flags().Lookup("tls"))
	viper.BindPFlag("deployment-mode", serveCmd.Flags().Lookup("deployment-mode"))
	viper.BindPFlag("jwt-secret", serveCmd.Flags().Lookup("jwt-secret"))
	viper.BindPFlag("postgres-url", serveCmd.Flags().Lookup("postgres-url"))
	viper.BindPFlag("kafka-brokers", serveCmd.Flags().Lookup("kafka-brokers"))
	viper.BindPFlag("anon-tunnel-limit", serveCmd.Flags().Lookup("anon-tunnel-limit"))
	viper.BindPFlag("user-tunnel-limit", serveCmd.Flags().Lookup("user-tunnel-limit"))
	viper.BindPFlag("paid-tunnel-limit", serveCmd.Flags().Lookup("paid-tunnel-limit"))
	viper.BindPFlag("stripe-secret-key", serveCmd.Flags().Lookup("stripe-secret-key"))
	viper.BindPFlag("stripe-webhook-secret", serveCmd.Flags().Lookup("stripe-webhook-secret"))
	viper.BindPFlag("stripe-pro-price-id", serveCmd.Flags().Lookup("stripe-pro-price-id"))
}

func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "***"
	}
	if u.User == nil {
		return raw
	}
	if _, hasPass := u.User.Password(); hasPass {
		u.User = url.UserPassword(u.User.Username(), "***")
	} else {
		u.User = url.User(u.User.Username())
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
	deploymentMode := viper.GetString("deployment-mode")
	jwtSecret := viper.GetString("jwt-secret")
	postgresURL := viper.GetString("postgres-url")
	kafkaBrokers := viper.GetString("kafka-brokers")
	anonLimit := viper.GetInt("anon-tunnel-limit")
	userLimit := viper.GetInt("user-tunnel-limit")
	paidLimit := viper.GetInt("paid-tunnel-limit")

	redisURL := viper.GetString("redis-url")
	if !cmd.Flags().Changed("redis-url") {
		if _, ok := os.LookupEnv("BURROW_REDIS_URL"); !ok {
			if u := os.Getenv("REDIS_URL"); u != "" {
				redisURL = u
			}
		}
	}

	secret := viper.GetString("secret")

	pidFile := internal.ConfigDir() + "/pid"
	if err := os.MkdirAll(internal.ConfigDir(), 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create config directory: %v\n", err)
	} else if err := os.WriteFile(pidFile, fmt.Appendf(nil, "%d", os.Getpid()), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write PID file: %v\n", err)
	} else {
		defer os.Remove(pidFile)
	}

	redis, err := internal.NewRedisClient(redisURL)
	if err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}
	defer redis.Close()

	registry := internal.NewTunnelRegistry(redis, domain, secure)

	var pg *internal.PostgresClient
	if postgresURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		pg, err = internal.NewPostgresClient(ctx, postgresURL)
		if err != nil {
			return fmt.Errorf("failed to connect to postgres: %w", err)
		}
		defer pg.Close()

		if err := pg.Migrate(ctx); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
		log.Println("Postgres connected and migrations applied")
	} else {
		log.Println("Warning: no --postgres-url provided, API endpoints will return 503")
	}

	var kafkaProducer sarama.AsyncProducer
	if kafkaBrokers != "" {
		brokers := []string{kafkaBrokers}
		if len(brokers) == 1 && containsComma(kafkaBrokers) {
			brokers = splitComma(kafkaBrokers)
		}
		kafkaProducer, err = internal.NewKafkaProducer(brokers)
		if err != nil {
			return fmt.Errorf("failed to create kafka producer: %w", err)
		}
		defer kafkaProducer.Close()

		if err := internal.EnsureTopic(brokers); err != nil {
			log.Printf("Warning: failed to ensure kafka topic: %v", err)
		}
		log.Println("Kafka producer connected")
	}

	cfg := internal.ServerConfig{
		Registry:       registry,
		Domain:         domain,
		Secret:         secret,
		Secure:         secure,
		DeploymentMode: deploymentMode,
		PG:             pg,
		JWTSecret:      jwtSecret,
		Limits: internal.TunnelLimits{
			AnonLimit: anonLimit,
			UserLimit: userLimit,
			PaidLimit: paidLimit,
		},
		KafkaProducer:       kafkaProducer,
		StripeSecretKey:     viper.GetString("stripe-secret-key"),
		StripeWebhookSecret: viper.GetString("stripe-webhook-secret"),
		StripeProPriceID:    viper.GetString("stripe-pro-price-id"),
	}

	server := internal.NewServer(cfg)

	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("Starting burrowd on %s\n", addr)
	fmt.Printf("Domain: %s (TLS: %v)\n", domain, secure)
	fmt.Printf("Deployment mode: %s\n", deploymentMode)
	fmt.Printf("Redis: %s\n", redactURL(redisURL))
	if pg != nil {
		fmt.Printf("Postgres: %s\n", redactURL(postgresURL))
	}

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

func containsComma(s string) bool {
	for _, c := range s {
		if c == ',' {
			return true
		}
	}
	return false
}

func splitComma(s string) []string {
	var result []string
	for _, part := range splitOnComma(s) {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func splitOnComma(s string) []string {
	var result []string
	start := 0
	for i, c := range s {
		if c == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}