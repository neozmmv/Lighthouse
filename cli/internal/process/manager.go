package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ProcessStatus holds runtime information about a managed process.
type ProcessStatus struct {
	Running bool
	PID     int
	Uptime  time.Duration
}

// Manager spawns, monitors, and stops OS processes for native mode.
type Manager struct {
	RunDir string // ~/.lighthouse/run/
	LogDir string // ~/.lighthouse/logs/
}

// NewManager creates a Manager, ensuring the run and log directories exist.
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	base := filepath.Join(home, ".lighthouse")
	runDir := filepath.Join(base, "run")
	logDir := filepath.Join(base, "logs")

	for _, d := range []string{runDir, logDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}
	return &Manager{RunDir: runDir, LogDir: logDir}, nil
}

// Start launches a named process in the background, redirecting output to a log file.
// It is idempotent: if the process is already running it returns nil.
func (m *Manager) Start(name, bin string, args, env []string) error {
	if m.IsRunning(name) {
		return nil
	}

	logPath := filepath.Join(m.LogDir, name+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file for %s: %w", name, err)
	}

	c := exec.Command(bin, args...)
	c.Stdout = logFile
	c.Stderr = logFile
	c.Env = append(os.Environ(), env...)

	// Platform-specific: detach from parent so the process outlives the CLI.
	setSysProcAttr(c)

	if err := c.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start %s: %w", name, err)
	}

	pid := c.Process.Pid
	pidPath := filepath.Join(m.RunDir, name+".pid")
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0644); err != nil {
		killProcess(c.Process)
		logFile.Close()
		return fmt.Errorf("failed to write PID file for %s: %w", name, err)
	}

	// Release our reference; the process runs independently.
	c.Process.Release()
	logFile.Close()
	return nil
}

// Stop sends SIGTERM to the named process, waits up to 5 s, then force-kills.
func (m *Manager) Stop(name string) error {
	pid, err := m.readPID(name)
	if err != nil {
		return nil // already stopped / no PID file
	}

	if !isProcessAlive(pid) {
		m.removePID(name)
		return nil
	}

	if err := terminateProcess(pid); err != nil {
		return fmt.Errorf("failed to terminate %s (PID %d): %w", name, pid, err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !isProcessAlive(pid) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if isProcessAlive(pid) {
		killPID(pid)
	}

	m.removePID(name)
	return nil
}

// StopAll stops all known processes in reverse startup order.
func (m *Manager) StopAll() error {
	var lastErr error
	for _, name := range []string{"tor", "caddy", "backend", "minio"} {
		if err := m.Stop(name); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// IsRunning returns true if the named process is currently alive.
func (m *Manager) IsRunning(name string) bool {
	pid, err := m.readPID(name)
	if err != nil {
		return false
	}
	return isProcessAlive(pid)
}

// Status returns a map of process name → status for all managed processes.
func (m *Manager) Status() map[string]ProcessStatus {
	names := []string{"minio", "backend", "caddy", "tor"}
	result := make(map[string]ProcessStatus, len(names))
	for _, name := range names {
		pid, err := m.readPID(name)
		if err != nil || !isProcessAlive(pid) {
			result[name] = ProcessStatus{Running: false}
			continue
		}
		result[name] = ProcessStatus{
			Running: true,
			PID:     pid,
			Uptime:  m.uptimeFor(name),
		}
	}
	return result
}

func (m *Manager) readPID(name string) (int, error) {
	data, err := os.ReadFile(filepath.Join(m.RunDir, name+".pid"))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func (m *Manager) removePID(name string) {
	os.Remove(filepath.Join(m.RunDir, name+".pid"))
}

func (m *Manager) uptimeFor(name string) time.Duration {
	info, err := os.Stat(filepath.Join(m.RunDir, name+".pid"))
	if err != nil {
		return 0
	}
	return time.Since(info.ModTime())
}
