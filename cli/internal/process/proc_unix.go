//go:build !windows

package process

import (
	"os"
	"os/exec"
	"syscall"
)

func setSysProcAttr(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // new session; process outlives the parent CLI
	}
}

func isProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks existence without sending a real signal.
	return proc.Signal(syscall.Signal(0)) == nil
}

func terminateProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
}

func killProcess(proc *os.Process) {
	_ = proc.Signal(syscall.SIGKILL)
}

func killPID(pid int) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = proc.Signal(syscall.SIGKILL)
}
