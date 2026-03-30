package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// this is for the docker version.
/* func getOnionUrl() (string, error) {
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

	url := strings.TrimSpace(string(output))
	return url, nil
}
*/

var urlCmd = &cobra.Command{
	Use:   "url",
	Short: "Get the .onion URL for sending files. (Lighthouse must be running!)",
	RunE: func(cmd *cobra.Command, args []string) error {
		running := isRunning()
		if !running {
			return fmt.Errorf("Lighthouse is not running. Please start it with 'lighthouse up' first.")
		}
		url, err := getOnionAddress()
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
