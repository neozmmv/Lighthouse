//go:build !windows

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

func runSetup() error {
	dir, err := getLighthouseDir()
	if err != nil {
		return err
	}

	// already initialized — both config files present
	_, composeErr := os.Stat(filepath.Join(dir, "docker-compose.yml"))
	_, caddyErr := os.Stat(filepath.Join(dir, "Caddyfile"))
	if composeErr == nil && caddyErr == nil {
		return nil
	}

	fmt.Println("First run — downloading Lighthouse configuration...")

	if err := downloadToFile(
		githubBaseURL+"/docker-compose.yml",
		filepath.Join(dir, "docker-compose.yml"),
	); err != nil {
		return fmt.Errorf("failed to download docker-compose.yml: %w", err)
	}

	if err := downloadToFile(
		githubBaseURL+"/Caddyfile",
		filepath.Join(dir, "Caddyfile"),
	); err != nil {
		return fmt.Errorf("failed to download Caddyfile: %w", err)
	}

	fmt.Println("Setup complete.")
	return nil
}
