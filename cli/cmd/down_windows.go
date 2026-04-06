package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

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
			return fmt.Errorf("failed to get lighthouse directory: %w", err)
		}

		// read daemon PID from file
		data, err := os.ReadFile(dir + "\\lighthouse.pid")
		if err != nil {
			return fmt.Errorf("failed to read PID file: %w", err)
		}

		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			return fmt.Errorf("invalid PID in file: %w", err)
		}

		// find and kill the daemon process
		p, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("failed to find process %d: %w", pid, err)
		}

		if err := p.Kill(); err != nil {
			return fmt.Errorf("failed to kill daemon process: %w", err)
		}

		if err := clearPid(); err != nil {
			return fmt.Errorf("failed to clear PID file: %w", err)
		}

		fmt.Println("Lighthouse stopped.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
