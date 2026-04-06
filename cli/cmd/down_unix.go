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

		fmt.Println("Stopping Lighthouse...")
		c := exec.Command("docker", "compose", "down")
		c.Dir = dir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
