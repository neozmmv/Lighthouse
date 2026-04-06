//go:build !windows

package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var port string

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
		fmt.Printf("Access on http://localhost:%s\n", port)
		c := exec.Command("docker", "compose", "up", "-d")
		c.Dir = dir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Env = append(os.Environ(), "LIGHTHOUSE_PORT="+port)
		return c.Run()
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVar(&port, "port", "80", "host port for the web interface")
}
