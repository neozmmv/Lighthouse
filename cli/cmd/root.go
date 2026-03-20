package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use: "lighthouse",
	Short: "A temporary file-receiving station on the Tor Network.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// skip docker checks for help and version commands
		skipCommands := []string{"help", "version", "update"}
		for _, name := range skipCommands {
			if cmd.Name() == name {
				return nil
			}
		}

		if !hasDocker() {
			return fmt.Errorf("Docker is not installed or not running.")
		}
		if !hasCompose() {
			return fmt.Errorf("Docker Compose is not available. Please install Docker Compose v2 or later.")
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}