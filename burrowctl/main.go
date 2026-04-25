package main

import (
	"fmt"
	"os"

	"burrow/burrowctl/cmd"
)

func main() {
	cmd.SetVersion(Version)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
