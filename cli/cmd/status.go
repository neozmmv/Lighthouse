package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/neozmmv/Lighthouse/cli/internal/process"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if Lighthouse is running",
	RunE: func(cmd *cobra.Command, args []string) error {
		if isNativeMode() {
			return nativeStatus()
		}
		if isRunning() {
			fmt.Println("Lighthouse is running.")
		} else {
			fmt.Println("Lighthouse is not running.")
		}
		return nil
	},
}

func nativeStatus() error {
	pm, err := process.NewManager()
	if err != nil {
		return fmt.Errorf("✗ Failed to initialize process manager: %w", err)
	}

	statuses := pm.Status()

	fmt.Println("Lighthouse status (native mode):")
	fmt.Println()

	anyRunning := false
	for _, name := range []string{"minio", "backend", "caddy", "tor"} {
		s := statuses[name]
		if s.Running {
			anyRunning = true
			fmt.Printf("  %-10s ✓ running  (PID %d, up %s)\n", name, s.PID, formatDuration(s.Uptime))
		} else {
			fmt.Printf("  %-10s ✗ stopped\n", name)
		}
	}

	if anyRunning {
		home, _ := os.UserHomeDir()
		hostnameFile := filepath.Join(home, ".lighthouse", "hidden_service", "hostname")
		if data, err := os.ReadFile(hostnameFile); err == nil {
			if addr := strings.TrimSpace(string(data)); addr != "" {
				fmt.Println()
				fmt.Printf("  Address: %s\n", addr)
			}
		}
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
