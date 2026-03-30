package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

		// start MinIO
		minio := exec.Command(
			filepath.Join(binDir, "minio.exe"),
			"server",
			filepath.Join(dir, "data", "minio"),
			"--address", "127.0.0.1:9000",
			"--console-address", "127.0.0.1:9001",
		)
		minio.Env = append(os.Environ(),
			"MINIO_ROOT_USER="+cfg.MinioUser,
			"MINIO_ROOT_PASSWORD="+cfg.MinioPass,
		)
		if err := minio.Start(); err != nil {
			return fmt.Errorf("failed to start MinIO: %w", err)
		}
		time.Sleep(2 * time.Second)

		// start Caddy
		caddy := exec.Command(
			filepath.Join(binDir, "caddy.exe"),
			"run",
			"--config", filepath.Join(dir, "Caddyfile"),
			"--adapter", "caddyfile",
		)
		caddy.Env = append(os.Environ(),
			"LIGHTHOUSE_STATIC_DIR="+filepath.Join(dir, "frontend"),
		)
		if err := caddy.Start(); err != nil {
			minio.Process.Kill()
			return fmt.Errorf("failed to start Caddy: %w", err)
		}

		// start Tor
		tor := exec.Command(
			filepath.Join(binDir, "tor.exe"),
			"-f", filepath.Join(dir, "tor", "torrc"),
		)
		torStderr, err := tor.StderrPipe()
		if err != nil {
			minio.Process.Kill()
			caddy.Process.Kill()
			return err
		}
		if err := tor.Start(); err != nil {
			minio.Process.Kill()
			caddy.Process.Kill()
			return fmt.Errorf("failed to start Tor: %w", err)
		}

		// drain Tor stderr to prevent the pipe from blocking
		go func() {
			scanner := bufio.NewScanner(torStderr)
			for scanner.Scan() {
				_ = strings.Contains(scanner.Text(), "Bootstrapped")
			}
		}()

		// wait for any process to exit
		done := make(chan error, 3)
		go func() { done <- minio.Wait() }()
		go func() { done <- caddy.Wait() }()
		go func() { done <- tor.Wait() }()

		// if any process dies, kill the others and clean up
		<-done
		minio.Process.Kill()
		caddy.Process.Kill()
		tor.Process.Kill()
		clearPid()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(daemonCmd)
}
