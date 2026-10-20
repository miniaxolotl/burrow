package main

import (
	"fmt"
	"os"

	"burrow/burrowd/cmd"
	"burrow/protocol/telemetry"
)

var Revision, TelemetryBuild, PostHogKey, SentryDSN string

func main() {
	cmd.SetVersion(Version)
	if err := cmd.Execute(telemetry.Config{Enabled: TelemetryBuild == "production" && os.Getenv("BURROW_TELEMETRY_ENABLED") == "true", Surface: "server", Version: Version, Revision: Revision, Key: PostHogKey, DSN: SentryDSN, StateDir: telemetry.StateDir()}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
