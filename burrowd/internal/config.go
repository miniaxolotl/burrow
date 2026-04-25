package internal

import "os"

// ConfigDir returns the ~/.burrow directory used for PID and token files.
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".burrow"
	}
	return home + "/.burrow"
}
