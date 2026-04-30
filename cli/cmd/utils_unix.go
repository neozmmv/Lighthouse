//go:build !windows

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const backendBaseURL = "http://localhost:4406"

func getLighthouseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, ".lighthouse")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create lighthouse directory: %w", err)
	}

	return dir, nil
}

func composeArgs(extraArgs ...string) ([]string, string, error) {
	dir, err := getLighthouseDir()
	if err != nil {
		return nil, "", err
	}
	mode, _ := getActiveMode()
	composeFile := "docker-compose.yml"
	if mode == "tunnel" {
		composeFile = "docker-compose.cloudflare.yml"
	}
	args := append([]string{"compose", "-f", composeFile}, extraArgs...)
	return args, dir, nil
}

func isRunning() bool {
	args, dir, err := composeArgs("ps", "--quiet")
	if err != nil {
		return false
	}

	c := exec.Command("docker", args...)
	c.Dir = dir
	out, err := c.Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

func getOnionAddress() (string, error) {
	c := exec.Command("docker", "exec", "lighthouse-tor", "cat", "/var/lib/tor/hidden_service/hostname")
	out, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get .onion address: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// pollCloudflareTunnelURL scrapes the cloudflared container logs for the trycloudflare.com URL.
func pollCloudflareTunnelURL() (string, error) {
	c := exec.Command("docker", "logs", "lighthouse-cloudflared")
	out, err := c.CombinedOutput()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "trycloudflare.com") {
			for _, field := range strings.Fields(line) {
				field = strings.Trim(field, "|")
				if strings.HasPrefix(field, "https://") {
					return field, nil
				}
			}
		}
	}
	return "", fmt.Errorf("tunnel URL not yet available in logs")
}

func getTunnelURL() (string, error) {
	dir, err := getLighthouseDir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, "tunnel_url"))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func saveTunnelURL(url string) error {
	dir, err := getLighthouseDir()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "tunnel_url"), []byte(url), 0600)
}

func clearTunnelURL() {
	dir, err := getLighthouseDir()
	if err != nil {
		return
	}
	os.Remove(filepath.Join(dir, "tunnel_url"))
}

func getActiveMode() (string, error) {
	dir, err := getLighthouseDir()
	if err != nil {
		return "tor", err
	}
	data, err := os.ReadFile(filepath.Join(dir, "mode"))
	if err != nil {
		return "tor", nil
	}
	return strings.TrimSpace(string(data)), nil
}

func saveActiveMode(mode string) error {
	dir, err := getLighthouseDir()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "mode"), []byte(mode), 0600)
}
