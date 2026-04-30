//go:build !windows

package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop Lighthouse",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !isRunning() {
			fmt.Println("Lighthouse is already down.")
			return nil
		}

		dir, err := getLighthouseDir()
		if err != nil {
			return err
		}

		mode, _ := getActiveMode()
		composeFile := "docker-compose.yml"
		if mode == "tunnel" {
			composeFile = "docker-compose.cloudflare.yml"
		}

		fmt.Println("Stopping Lighthouse...")
		c := exec.Command("docker", "compose", "-f", composeFile, "down")
		c.Dir = dir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Env = append(os.Environ(), "LIGHTHOUSE_DIR="+dir)
		if err := c.Run(); err != nil {
			return err
		}

		clearTunnelURL()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
