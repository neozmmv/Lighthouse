package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func getOnionUrl() (string, error) {
	if isNativeMode() {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		data, err := os.ReadFile(filepath.Join(home, ".lighthouse", "hidden_service", "hostname"))
		if err != nil {
			return "", fmt.Errorf("failed to read .onion address: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	}

	dir, err := getLighthouseDir()
	if err != nil {
		return "", fmt.Errorf("failed to get lighthouse directory: %w", err)
	}
	c := exec.Command("docker", "exec", "lighthouse-tor", "cat", "/var/lib/tor/hidden_service/hostname")
	c.Dir = dir
	output, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get .onion URL: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

var urlCmd = &cobra.Command{
	Use:   "url",
	Short: "Get the .onion URL for sending files. (Lighthouse must be running!)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !isRunning() {
			return fmt.Errorf("Lighthouse is not running. Please start it with 'lighthouse up' first.")
		}
		url, err := getOnionUrl()
		if err != nil {
			return fmt.Errorf("failed to get .onion URL: %w", err)
		}
		fmt.Println(url)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(urlCmd)
}
