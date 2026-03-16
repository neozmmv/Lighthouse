package cmd

import (
	"os/exec"
	"strings"
)

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