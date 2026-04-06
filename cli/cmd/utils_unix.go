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

func isRunning() bool {
	dir, err := getLighthouseDir()
	if err != nil {
		return false
	}

	c := exec.Command("docker", "compose", "ps", "--quiet")
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
