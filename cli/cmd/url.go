package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var urlCmd = &cobra.Command{
	Use:   "url",
	Short: "Get the URL for sending files. (Lighthouse must be running!)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !isRunning() {
			return fmt.Errorf("Lighthouse is not running. Please start it with 'lighthouse up' first.")
		}
		// Tunnel URL is only present when started with --tunnel
		if url, err := getTunnelURL(); err == nil {
			fmt.Println(url)
			return nil
		}
		url, err := getOnionAddress()
		if err != nil {
			return fmt.Errorf("failed to get URL: %w", err)
		}
		fmt.Println(url)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(urlCmd)
}
