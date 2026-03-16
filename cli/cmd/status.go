package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use: "status",
	Short: "Check if Lighthouse is running",
	Run: func(cmd *cobra.Command, args []string) {
		running := isRunning()
		if running {
			fmt.Println("Lighthouse is running.")
		} else {
			fmt.Println("Lighthouse is not running.")
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}