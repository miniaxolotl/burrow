package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"burrow/burrowctl/cmd"
	"burrow/burrowctl/internal/clioutput"
	"burrow/protocol/telemetry"
)

// Public ingestion configuration is linked only into admitted release builds.
var Revision, TelemetryBuild, PostHogKey, SentryDSN string

func main() {
	// Handle EPIPE as an output failure so tunnel shutdown/drain runs before exit.
	// Go otherwise terminates immediately on a broken stdout/stderr pipe.
	signal.Ignore(syscall.SIGPIPE)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cmd.SetVersion(Version)
	if err := cmd.ExecuteContext(ctx, telemetry.Config{Enabled: TelemetryBuild == "production", Surface: "cli", Version: Version, Revision: Revision, Key: PostHogKey, DSN: SentryDSN, StateDir: telemetry.StateDir()}); err != nil {
		os.Exit(clioutput.ExitCode(err))
	}
}
