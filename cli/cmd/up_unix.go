//go:build !windows

package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start Lighthouse",
	RunE: func(cmd *cobra.Command, args []string) error {
		if isRunning() {
			fmt.Println("Lighthouse is already running.")
			return nil
		}

		if err := runSetup(); err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}

		dir, err := getLighthouseDir()
		if err != nil {
			return err
		}

		fmt.Println("Starting Lighthouse...")
		c := exec.Command("docker", "compose", "up", "-d")
		c.Dir = dir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
}
