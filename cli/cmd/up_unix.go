//go:build !windows

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

var port string

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start Lighthouse",
	RunE: func(cmd *cobra.Command, args []string) error {
		tunnel, _ := cmd.Flags().GetBool("tunnel")

		if isRunning() {
			fmt.Println("Lighthouse is already running.")
			return nil
		}

		if err := runSetup(); err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}

		dir, err := getLighthouseDir()
		if err != nil {
			return err
		}

		fmt.Println("Starting Lighthouse...")

		baseEnv := append(os.Environ(),
			"LIGHTHOUSE_PORT="+port,
			"LIGHTHOUSE_DIR="+dir,
		)

		if tunnel {
			clearTunnelURL()

			// Phase 1: start cloudflared and minio so we can obtain the tunnel URL
			// before the backend starts (backend needs MINIO_PUBLIC_ENDPOINT set correctly)
			phase1 := exec.Command("docker", "compose", "-f", "docker-compose.cloudflare.yml",
				"up", "-d", "cloudflared", "minio", "minio-init")
			phase1.Dir = dir
			phase1.Stdout = os.Stdout
			phase1.Stderr = os.Stderr
			phase1.Env = baseEnv
			if err := phase1.Run(); err != nil {
				return fmt.Errorf("failed to start cloudflared: %w", err)
			}

			// Phase 2: wait for cloudflared to publish the tunnel URL
			fmt.Print("Waiting for Cloudflare tunnel")
			var tunnelURL string
			for i := 0; i < 60; i++ {
				time.Sleep(2 * time.Second)
				fmt.Print(".")
				if url, err := pollCloudflareTunnelURL(); err == nil {
					tunnelURL = url
					break
				}
			}
			fmt.Println()

			if tunnelURL == "" {
				return fmt.Errorf("cloudflare tunnel did not start in time")
			}

			// Phase 3: start remaining services with the tunnel URL injected
			phase3 := exec.Command("docker", "compose", "-f", "docker-compose.cloudflare.yml", "up", "-d")
			phase3.Dir = dir
			phase3.Stdout = os.Stdout
			phase3.Stderr = os.Stderr
			phase3.Env = append(baseEnv, "MINIO_PUBLIC_ENDPOINT="+tunnelURL)
			if err := phase3.Run(); err != nil {
				return fmt.Errorf("failed to start services: %w", err)
			}

			if err := saveTunnelURL(tunnelURL); err != nil {
				return err
			}
			if err := saveActiveMode("tunnel"); err != nil {
				return err
			}

			fmt.Printf("Lighthouse is running at: %s\n", tunnelURL)
			fmt.Printf("Manage files at: http://localhost:%s\n", port)
		} else {
			c := exec.Command("docker", "compose", "-f", "docker-compose.yml", "up", "-d")
			c.Dir = dir
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Env = baseEnv
			if err := c.Run(); err != nil {
				return err
			}

			if err := saveActiveMode("tor"); err != nil {
				return err
			}

			// wait for Tor hidden service hostname to be written
			fmt.Print("Waiting for Tor")
			for i := 0; i < 60; i++ {
				time.Sleep(1 * time.Second)
				fmt.Print(".")
				if _, err := getOnionAddress(); err == nil {
					break
				}
			}
			fmt.Println()

			onion, err := getOnionAddress()
			if err != nil {
				return fmt.Errorf("tor did not bootstrap in time: %w", err)
			}

			fmt.Printf("Lighthouse is running at: %s\n", onion)
			fmt.Printf("Manage files at: http://localhost:%s\n", port)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVar(&port, "port", "80", "host port for the web interface")
	upCmd.Flags().Bool("tunnel", false, "Use Cloudflare Tunnel instead of Tor to expose Lighthouse")
}
