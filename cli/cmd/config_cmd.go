package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show Lighthouse configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Printf("MinIO User:     %s\n", cfg.MinioUser)
		fmt.Printf("MinIO Password: %s\n", cfg.MinioPass)
		fmt.Printf("MinIO Console:  http://127.0.0.1:9001\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
