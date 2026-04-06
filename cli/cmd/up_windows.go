package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

func startDaemon() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	c := exec.Command(exe, "_daemon")
	c.Stdout = nil
	c.Stderr = nil
	c.Stdin = nil
	c.SysProcAttr = daemonSysProcAttr()

	if err := c.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	if err := savePid(c.Process.Pid); err != nil {
		c.Process.Kill()
		return fmt.Errorf("failed to save daemon PID: %w", err)
	}

	return nil
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start Lighthouse",
	RunE: func(cmd *cobra.Command, args []string) error {
		if isRunning() {
			fmt.Println("Lighthouse is already running.")
			return nil
		}

		if !isInitialized() {
			if err := runSetup(); err != nil {
				return fmt.Errorf("setup failed: %w", err)
			}
		}

		fmt.Println("Starting Lighthouse...")
		if err := startDaemon(); err != nil {
			return err
		}

		// wait for Tor to bootstrap by polling the hostname file
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
		fmt.Printf("Access on host: http://localhost:8080\n")
		return nil
	},
}

// hidden subcommand — manages child processes in the background
var daemonCmd = &cobra.Command{
	Use:    "_daemon",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := getLighthouseDir()
		if err != nil {
			return err
		}
		binDir, err := getBinDir()
		if err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// create job object — kills all children when daemon exits
		job, err := createJobObject()
		if err != nil {
			return fmt.Errorf("failed to create job object: %w", err)
		}

		// start MinIO
		minio := exec.Command(
			filepath.Join(binDir, "minio.exe"),
			"server",
			filepath.Join(dir, "data", "minio"),
			"--address", "127.0.0.1:9000",
			"--console-address", "127.0.0.1:9001",
		)
		minio.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
		minio.Env = append(os.Environ(),
			"MINIO_ROOT_USER="+cfg.MinioUser,
			"MINIO_ROOT_PASSWORD="+cfg.MinioPass,
		)
		if err := minio.Start(); err != nil {
			return fmt.Errorf("failed to start MinIO: %w", err)
		}
		if err := assignToJob(job, minio.Process.Pid); err != nil {
			minio.Process.Kill()
			return fmt.Errorf("failed to assign MinIO to job: %w", err)
		}
		time.Sleep(2 * time.Second)

		// start Tor before the backend so we can read the .onion address
		tor := exec.Command(
			filepath.Join(binDir, "tor.exe"),
			"-f", filepath.Join(dir, "tor", "torrc"),
		)
		tor.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
		torStderr, err := tor.StderrPipe()
		if err != nil {
			minio.Process.Kill()
			return err
		}
		if err := tor.Start(); err != nil {
			minio.Process.Kill()
			return fmt.Errorf("failed to start Tor: %w", err)
		}
		if err := assignToJob(job, tor.Process.Pid); err != nil {
			minio.Process.Kill()
			tor.Process.Kill()
			return fmt.Errorf("failed to assign Tor to job: %w", err)
		}

		// wait for Tor to bootstrap by watching stderr
		torReady := make(chan struct{})
		go func() {
			scanner := bufio.NewScanner(torStderr)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, "Bootstrapped 100%") {
					close(torReady)
					// drain remaining output
					for scanner.Scan() {
					}
					return
				}
			}
		}()

		select {
		case <-torReady:
		case <-time.After(60 * time.Second):
			minio.Process.Kill()
			tor.Process.Kill()
			return fmt.Errorf("tor bootstrap timed out")
		}

		// read the .onion address — Tor has bootstrapped at this point
		onion, err := getOnionAddress()
		if err != nil {
			minio.Process.Kill()
			tor.Process.Kill()
			return fmt.Errorf("failed to read onion address: %w", err)
		}

		// start backend with the .onion address as the public MinIO endpoint
		backend := exec.Command(filepath.Join(binDir, "backend.exe"))
		backend.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
		backend.Env = append(os.Environ(),
			"MINIO_ROOT_USER="+cfg.MinioUser,
			"MINIO_ROOT_PASSWORD="+cfg.MinioPass,
			"MINIO_ENDPOINT=127.0.0.1:9000",
			"MINIO_PUBLIC_ENDPOINT="+onion, // presigned upload URLs point to .onion
			"MINIO_LOCAL_ENDPOINT=127.0.0.1:9000",
			"PORT=8000",
		)
		if err := backend.Start(); err != nil {
			minio.Process.Kill()
			tor.Process.Kill()
			return fmt.Errorf("failed to start backend: %w", err)
		}
		if err := assignToJob(job, backend.Process.Pid); err != nil {
			minio.Process.Kill()
			tor.Process.Kill()
			backend.Process.Kill()
			return fmt.Errorf("failed to assign backend to job: %w", err)
		}

		// start Caddy
		caddy := exec.Command(
			filepath.Join(binDir, "caddy.exe"),
			"run",
			"--config", filepath.Join(dir, "Caddyfile"),
			"--adapter", "caddyfile",
		)
		caddy.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
		caddy.Env = append(os.Environ(),
			"LIGHTHOUSE_STATIC_DIR="+filepath.Join(dir, "frontend"),
		)
		if err := caddy.Start(); err != nil {
			minio.Process.Kill()
			tor.Process.Kill()
			backend.Process.Kill()
			return fmt.Errorf("failed to start Caddy: %w", err)
		}
		if err := assignToJob(job, caddy.Process.Pid); err != nil {
			minio.Process.Kill()
			tor.Process.Kill()
			backend.Process.Kill()
			caddy.Process.Kill()
			return fmt.Errorf("failed to assign Caddy to job: %w", err)
		}

		// wait for any process to exit
		done := make(chan error, 4)
		go func() { done <- minio.Wait() }()
		go func() { done <- tor.Wait() }()
		go func() { done <- backend.Wait() }()
		go func() { done <- caddy.Wait() }()

		// if any process dies, kill the others and clean up
		<-done
		minio.Process.Kill()
		tor.Process.Kill()
		backend.Process.Kill()
		caddy.Process.Kill()
		clearPid()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(daemonCmd)
}
