package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use: "down",
	Short: "Stop Lighthouse",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := getLighthouseDir()
		if err != nil {
			return fmt.Errorf("failed to get lighthouse directory: %w", err)
		}
		fmt.Println("Stopping Lighthouse...")
		c := exec.Command("docker", "compose", "down")
		c.Stderr = os.Stderr
		c.Stdout = os.Stdout
		c.Dir = dir
		return c.Run()
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}