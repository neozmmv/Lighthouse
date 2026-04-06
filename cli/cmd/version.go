package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use: "version",
	Short: "Print the version of Lighthouse",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Lighthouse CLI %s%s\n", version, versionSuffix)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}