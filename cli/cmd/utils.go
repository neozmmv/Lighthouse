package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// gets directory for lighthouse (.lighthouse in home directory)
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


func isRunning() bool {
	dir, err := getLighthouseDir()
	if err != nil {
		return false
	}

	c:= exec.Command("docker", "compose", "ps", "--quiet")
	c.Dir = dir
	out, err := c.Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}