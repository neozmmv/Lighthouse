package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/neozmmv/Lighthouse/cli/internal/process"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop Lighthouse",
	RunE: func(cmd *cobra.Command, args []string) error {
		if isNativeMode() {
			return runNativeDown()
		}
		return runDockerDown()
	},
}

func runNativeDown() error {
	pm, err := process.NewManager()
	if err != nil {
		return fmt.Errorf("✗ Failed to initialize process manager: %w", err)
	}

	statuses := pm.Status()
	anyRunning := false
	for _, s := range statuses {
		if s.Running {
			anyRunning = true
			break
		}
	}
	if !anyRunning {
		fmt.Println("Lighthouse is already down.")
		return nil
	}

	fmt.Println("Stopping Lighthouse...")
	if err := pm.StopAll(); err != nil {
		return fmt.Errorf("✗ Failed to stop all processes: %w", err)
	}
	fmt.Println("Lighthouse stopped.")
	return nil
}

func runDockerDown() error {
	dir, err := getLighthouseDir()
	if err != nil {
		return fmt.Errorf("failed to get lighthouse directory: %w", err)
	}
	if !isRunning() {
		fmt.Println("Lighthouse is already down.")
		return nil
	}
	fmt.Println("Stopping Lighthouse...")
	c := exec.Command("docker", "compose", "down")
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	c.Dir = dir
	return c.Run()
}

func init() {
	rootCmd.AddCommand(downCmd)
}
