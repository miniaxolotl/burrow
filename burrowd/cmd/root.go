package cmd

import (
	"strings"
	"time"

	"burrow/protocol/telemetry"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "burrowd",
	Short: "burrowd - Tunnel server",
	Long:  `burrowd is the server component that handles tunnel requests from clients.`,
}

func SetVersion(v string) {
	rootCmd.Version = v
}

func Execute(config telemetry.Config) (err error) {
	var reporter *telemetry.Reporter
	started := time.Now()
	rootCmd.PersistentPreRun = func(command *cobra.Command, _ []string) {
		if command.Name() != "serve" {
			return
		}
		reporter = telemetry.New(config)
		command.SetContext(telemetry.WithObserver(command.Context(), reporter))
	}
	defer func() {
		defer reporter.Close()
		if failure := recover(); failure != nil {
			reporter.Operation(telemetry.Serve, started, telemetry.Failure)
			panic(failure)
		}
		outcome := telemetry.Success
		if err != nil {
			outcome = telemetry.Failure
		}
		reporter.Operation(telemetry.Serve, started, outcome)
	}()
	return rootCmd.Execute()
}

func init() {
	// BURROW_* env vars map to config keys (hyphens become underscores).
	viper.SetEnvPrefix("BURROW")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(stopCmd)
}
