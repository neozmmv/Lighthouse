package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const backendBaseURL = "http://localhost:8000"

// port to point the tunnel to (windows - 8080)
const tunnelBaseURL = "http://localhost:8080"

func getCloudflareTunnelURL() (string, *exec.Cmd, error) {
	// to kill the tunnel use the *exec.Cmd
	// cmd.Process.Kill()
	binDir, err := getBinDir()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get bin dir: %w", err)
	}
	c := exec.Command(filepath.Join(binDir, "cloudflared.exe"), "tunnel", "--url", tunnelBaseURL)
	c.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}

	stderr, err := c.StderrPipe()
	if err != nil {
		return "", nil, err
	}

	if err := c.Start(); err != nil {
		return "", nil, fmt.Errorf("failed to start cloudflared: %w", err)
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "trycloudflare.com") {
			parts := strings.Fields(line)
			for _, p := range parts {
				if strings.HasPrefix(p, "https://") {
					// keep draining stderr so cloudflared never blocks on a full pipe
					go func() {
						for scanner.Scan() {
						}
					}()
					return p, c, nil
				}
			}
		}
	}

	return "", nil, fmt.Errorf("URL not found in cloudflared output")
}

// gets directory for lighthouse (appdata/lighthouse)
func getLighthouseDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", fmt.Errorf("APPDATA environment variable is not set")
	}

	dir := filepath.Join(appData, "lighthouse")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create lighthouse directory: %w", err)
	}
	return dir, nil
}

func getBinDir() (string, error) {
	dir, err := getLighthouseDir()
	if err != nil {
		return "", fmt.Errorf("failed to get lighthouse directory: %w", err)
	}
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin directory: %w", err)
	}
	return binDir, nil
}

func isInitialized() bool {
	dir, err := getLighthouseDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, "initialized"))
	return err == nil
}

func isRunning() bool {
	dir, err := getLighthouseDir()
	if err != nil {
		return false
	}

	data, err := os.ReadFile(filepath.Join(dir, "lighthouse.pid"))
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	// checks if process is running
	c := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH")
	out, err := c.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), strconv.Itoa(pid))
}

func savePid(pid int) error {
	dir, err := getLighthouseDir()
	if err != nil {
		return fmt.Errorf("failed to get lighthouse directory: %w", err)
	}
	//writes pid to lighthouse.pid file
	return os.WriteFile(filepath.Join(dir, "lighthouse.pid"), []byte(strconv.Itoa(pid)), 0600)
}

func clearPid() error {
	dir, err := getLighthouseDir()
	if err != nil {
		return fmt.Errorf("failed to get lighthouse directory: %w", err)
	}
	return os.Remove(filepath.Join(dir, "lighthouse.pid"))
}

func getOnionAddress() (string, error) {
	dir, err := getLighthouseDir()
	if err != nil {
		return "", fmt.Errorf("failed to get lighthouse directory: %w", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "tor", "hidden_service", "hostname"))
	if err != nil {
		return "", fmt.Errorf("failed to read onion address: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

/* func hasDocker() bool {
	c := exec.Command("docker", "--version")
	return c.Run() == nil
} */

/* func hasCompose() bool {
	c := exec.Command("docker", "compose", "version")
	return c.Run() == nil
} */

// checks if docker version is running (not native)
/* func isRunning() bool {
	dir, err := getLighthouseDir()
	if err != nil {
		return false
	}

	c := exec.Command("docker", "compose", "ps", "--quiet")
	c.Dir = dir
	out, err := c.Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
} */
