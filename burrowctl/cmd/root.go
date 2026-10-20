package cmd

import (
	"strings"
	"time"

	"burrow/burrowctl/cmd/tunnel"
	"burrow/burrowctl/internal/config"
	"burrow/protocol/telemetry"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:               "burrowctl",
	Short:             "burrowctl - Tunnel client",
	Long:              `burrowctl is the client CLI for creating tunnels to the burrow server.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return configErr },
}

var configErr error

func Execute(config telemetry.Config) (err error) {
	var reporter *telemetry.Reporter
	started := time.Now()
	operation := telemetry.Command
	rootCmd.PersistentPreRunE = func(command *cobra.Command, _ []string) error {
		if configErr != nil {
			return configErr
		}
		switch command.Name() {
		case "tunnel":
			operation = telemetry.Command
		case "list":
			operation = telemetry.List
		case "logs":
			operation = telemetry.Logs
		case "close":
			operation = telemetry.Close
		default:
			return nil
		}
		reporter = telemetry.New(config)
		command.SetContext(telemetry.WithObserver(command.Context(), reporter))
		started = time.Now()
		return nil
	}
	defer func() {
		defer reporter.Close()
		if failure := recover(); failure != nil {
			reporter.Operation(operation, started, telemetry.Failure)
			panic(failure)
		}
		outcome := telemetry.Success
		if err != nil {
			outcome = telemetry.Unavailable
		}
		reporter.Operation(operation, started, outcome)
		if err == nil {
			reporter.Track(telemetry.CommandCompleted)
		}
	}()
	return rootCmd.Execute()
}

func SetVersion(v string) {
	rootCmd.Version = v
}

func init() {
	cobra.OnInitialize(loadClientConfig)

	viper.SetEnvPrefix("BURROW")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	tunnel.AddCommands(rootCmd)
	rootCmd.AddCommand(telemetryCommand())
}

func loadClientConfig() {
	var cfgFile string
	cfgFile, configErr = config.EnsureConfig()
	if configErr != nil {
		return
	}

	viper.SetConfigFile(cfgFile)
	viper.SetConfigType("json")
	configErr = viper.ReadInConfig()
}
