package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/neozmmv/Lighthouse/cli/internal/process"
)

// getLighthouseDir returns ~/.lighthouse, creating it if necessary.
func getLighthouseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".lighthouse")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func hasDocker() bool {
	return exec.Command("docker", "--version").Run() == nil
}

func hasCompose() bool {
	return exec.Command("docker", "compose", "version").Run() == nil
}

func isRunning() bool {
	if isNativeMode() {
		pm, err := process.NewManager()
		if err != nil {
			return false
		}
		return pm.IsRunning("minio") || pm.IsRunning("tor")
	}

	dir, err := getLighthouseDir()
	if err != nil {
		return false
	}
	c := exec.Command("docker", "compose", "ps", "--quiet")
	c.Dir = dir
	out, err := c.Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

func setMode(mode string) error {
	dir, err := getLighthouseDir()
	if err != nil {
		return err
	}
	configDir := filepath.Join(dir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configDir, "mode"), []byte(mode), 0644)
}

func getMode() string {
	dir, err := getLighthouseDir()
	if err != nil {
		return "docker"
	}
	data, err := os.ReadFile(filepath.Join(dir, "config", "mode"))
	if err != nil {
		return "docker"
	}
	return strings.TrimSpace(string(data))
}

func isNativeMode() bool {
	return getMode() == "native"
}

// apiBaseURL returns the base URL for internal API calls.
// Docker exposes the backend on port 4406; native mode runs it directly on 8000.
func apiBaseURL() string {
	if isNativeMode() {
		return "http://localhost:8000"
	}
	return "http://localhost:4406"
}
