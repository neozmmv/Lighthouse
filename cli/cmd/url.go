package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func getOnionUrl() (string, error) {
	return "", nil
}


var urlCmd = &cobra.Command{
	Use: "url",
	Short: "Get the .onion URL for sending files. (Lighthouse must be running!)",
	RunE: func(cmd *cobra.Command, args []string) error {
		running := isRunning()
		if !running {
			return fmt.Errorf("Lighthouse is not running. Please start it with 'lighthouse up' first.")
		}
		return nil
	},
}
func init() {
	rootCmd.AddCommand(urlCmd)
}