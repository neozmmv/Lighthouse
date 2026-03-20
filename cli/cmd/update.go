package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

const githubBaseURL = "https://github.com/neozmmv/Lighthouse/releases/latest/download"

func downloadToFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response status: %s", resp.Status)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func updateConfigFiles() error {
	dir, err := getLighthouseDir()
	if err != nil {
		return fmt.Errorf("failed to get lighthouse directory: %w", err)
	}

	fmt.Println("Updating docker-compose.yml...")
	if err := downloadToFile(githubBaseURL+"/docker-compose.yml", filepath.Join(dir, "docker-compose.yml")); err != nil {
		return fmt.Errorf("failed to update docker-compose.yml: %w", err)
	}

	fmt.Println("Updating Caddyfile...")
	if err := downloadToFile(githubBaseURL+"/Caddyfile", filepath.Join(dir, "Caddyfile")); err != nil {
		return fmt.Errorf("failed to update Caddyfile: %w", err)
	}

	c:= exec.Command("docker", "compose", "pull")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Dir = dir
	if err := c.Run(); err != nil {
		return fmt.Errorf("failed to pull latest Docker images: %w", err)
	}
	return nil
}

func updateCLI() error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	binaryName := fmt.Sprintf("lighthouse-%s-%s", goos, goarch)
	url := githubBaseURL + "/" + binaryName

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine executable path: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "lighthouse-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	fmt.Printf("Downloading %s...\n", binaryName)
	if err := downloadToFile(url, tmpPath); err != nil {
		return fmt.Errorf("failed to download CLI binary: %w", err)
	}

	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to set binary permissions: %w", err)
	}

	fmt.Printf("Installing to %s...\n", execPath)
	if err := os.Rename(tmpPath, execPath); err != nil {
		// Retry with sudo (common on Linux/macOS when installed to /usr/local/bin)
		c := exec.Command("sudo", "mv", tmpPath, execPath)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin
		if sudoErr := c.Run(); sudoErr != nil {
			return fmt.Errorf("failed to install binary (try running with sudo): %w", err)
		}
	}

	return nil
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Lighthouse CLI and config files to the latest release",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Updating Lighthouse...")

		if err := updateCLI(); err != nil {
			return err
		}
		fmt.Println("Lighthouse CLI updated successfully.")

		if err := updateConfigFiles(); err != nil {
			return err
		}
		fmt.Println("Config files updated successfully.")

		if isRunning() {
			fmt.Println("\nLighthouse is currently running. Restart it to apply the new config:")
			fmt.Println("  lighthouse down && lighthouse up")
		}

		fmt.Println("\nLighthouse updated to the latest version!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
