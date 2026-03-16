package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func downloadFiles() error {
	dir, err := getLighthouseDir()
	if err != nil {
		return fmt.Errorf("failed to get lighthouse directory: %w", err)
	}
	
	// verify if docker-compose.yml and Caddyfile already exists
	_, composeExists := os.Stat(filepath.Join(dir, "docker-compose.yml"))
	_, caddyExists := os.Stat(filepath.Join(dir, "Caddyfile"))
	if composeExists == nil && caddyExists == nil {
		return nil
	}

	composeUrl := "https://github.com/neozmmv/Lighthouse/releases/latest/download/docker-compose.yml"
	caddyfileUrl := "https://github.com/neozmmv/Lighthouse/releases/latest/download/Caddyfile"

	composeResp, err := http.Get(composeUrl)
	if err != nil {
		return fmt.Errorf("failed to download docker-compose.yml: %w", err)
	}
	defer composeResp.Body.Close()

	caddyfileResp, err := http.Get(caddyfileUrl)
	if err != nil {
		return fmt.Errorf("failed to download Caddyfile: %w", err)
	}
	defer caddyfileResp.Body.Close()

	// create docker-compose.yml on .lighthouse directory
	composeFile, err := os.Create(filepath.Join(dir, "docker-compose.yml"))
	if err != nil {
		return fmt.Errorf("failed to create docker-compose.yml: %w", err)
	}
	defer composeFile.Close()

	_, err = io.Copy(composeFile, composeResp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy docker-compose.yml: %w", err)
	}

	// create Caddyfile on .lighthouse directory
	caddyfile, err := os.Create(filepath.Join(dir, "Caddyfile"))
	if err != nil {
		return fmt.Errorf("failed to create Caddyfile: %w", err)
	}
	defer caddyfile.Close()

	_, err = io.Copy(caddyfile, caddyfileResp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy Caddyfile: %w", err)
	}

	return nil
}

var upCmd = &cobra.Command{
	Use: "up",
	Short: "Start Lighthouse",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := downloadFiles(); err != nil {
			return err
		}
		// if has files and not error, run docker compose
		dir, err := getLighthouseDir()
		if err != nil {
			return fmt.Errorf("failed to get lighthouse directory: %w", err)
		}

		running := isRunning()
		if running {
			fmt.Println("Lighthouse is already running.")
			return nil
		}
		
		fmt.Println("Starting Lighthouse...")
		c := exec.Command("docker", "compose", "up", "-d")
		c.Dir = dir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
}