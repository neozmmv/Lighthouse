package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "lighthouse",
	Short: "A temporary file-receiving station on the Tor Network.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip checks for commands that never need Docker or native runtime.
		skipCommands := []string{"help", "version", "update", "up"}
		for _, name := range skipCommands {
			if cmd.Name() == name {
				return nil
			}
		}

		// Native mode is active — no Docker required.
		if isNativeMode() {
			return nil
		}

		if !hasDocker() {
			return fmt.Errorf("Docker is not installed or not running.\nRun 'lighthouse up --native' to use native mode without Docker.")
		}
		if !hasCompose() {
			return fmt.Errorf("Docker Compose is not available. Please install Docker Compose v2 or later.\nRun 'lighthouse up --native' to use native mode without Docker.")
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
