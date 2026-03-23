//go:build windows

package process

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func setSysProcAttr(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func isProcessAlive(pid int) bool {
	out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), strconv.Itoa(pid))
}

func terminateProcess(pid int) error {
	return exec.Command("taskkill", "/PID", strconv.Itoa(pid)).Run()
}

func killProcess(proc *os.Process) {
	exec.Command("taskkill", "/F", "/PID", strconv.Itoa(proc.Pid)).Run()
}

func killPID(pid int) {
	exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid)).Run()
}
