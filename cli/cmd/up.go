package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"github.com/neozmmv/Lighthouse/cli/internal/config"
	"github.com/neozmmv/Lighthouse/cli/internal/deps"
	"github.com/neozmmv/Lighthouse/cli/internal/process"
	torpkg "github.com/neozmmv/Lighthouse/cli/internal/tor"
	"github.com/spf13/cobra"
)

var nativeFlag bool

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start Lighthouse",
	RunE: func(cmd *cobra.Command, args []string) error {
		useNative := nativeFlag
		if !useNative && (!hasDocker() || !hasCompose()) {
			fmt.Println("⚠ Docker not available, falling back to native mode")
			useNative = true
		}

		if useNative {
			if err := setMode("native"); err != nil {
				return fmt.Errorf("failed to save mode: %w", err)
			}
			return runNativeUp()
		}

		if err := setMode("docker"); err != nil {
			return fmt.Errorf("failed to save mode: %w", err)
		}
		return runDockerUp()
	},
}

func runNativeUp() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	lighthouseDir, err := getLighthouseDir()
	if err != nil {
		return fmt.Errorf("✗ Failed to get Lighthouse directory: %w", err)
	}

	// 1. Ensure all dependencies are downloaded.
	dm, err := deps.NewManager()
	if err != nil {
		return fmt.Errorf("✗ Failed to initialize dependency manager: %w", err)
	}
	if err := dm.EnsureAll(ctx); err != nil {
		return err
	}

	// 2. Generate config files on first run.
	gen, err := config.NewGenerator()
	if err != nil {
		return fmt.Errorf("✗ Failed to initialize config generator: %w", err)
	}
	if err := gen.EnsureAll(); err != nil {
		return fmt.Errorf("✗ Failed to generate configs: %w", err)
	}

	// 3. Start all processes.
	pm, err := process.NewManager()
	if err != nil {
		return fmt.Errorf("✗ Failed to initialize process manager: %w", err)
	}

	fmt.Println("\nStarting Lighthouse (native mode)...")

	// MinIO
	if !pm.IsRunning("minio") {
		minioEnv, err := gen.MinioEnv()
		if err != nil {
			return fmt.Errorf("✗ Failed to get MinIO config: %w", err)
		}
		dataDir := filepath.Join(lighthouseDir, "data")
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return fmt.Errorf("✗ Failed to create data directory: %w", err)
		}
		if err := pm.Start("minio", dm.BinPath("minio"),
			[]string{"server", dataDir, "--address", ":9000", "--console-address", ":9001"},
			minioEnv,
		); err != nil {
			return fmt.Errorf("✗ Failed to start MinIO: %w\n  → Check %s/logs/minio.log for details", err, lighthouseDir)
		}
		// Give MinIO a moment to initialize before the backend connects.
		select {
		case <-ctx.Done():
			pm.StopAll()
			return fmt.Errorf("startup cancelled")
		case <-time.After(2 * time.Second):
		}
	}
	fmt.Println("  ✓ MinIO started")

	// Backend
	if !pm.IsRunning("backend") {
		backendEnv, err := gen.BackendEnv()
		if err != nil {
			pm.StopAll()
			return fmt.Errorf("✗ Failed to get backend config: %w", err)
		}
		if err := pm.Start("backend", dm.BinPath("lighthouse-backend"), nil, backendEnv); err != nil {
			pm.StopAll()
			return fmt.Errorf("✗ Failed to start backend: %w\n  → Check %s/logs/backend.log for details", err, lighthouseDir)
		}
	}
	fmt.Println("  ✓ Backend started")

	// Caddy — on Linux grant the binary permission to bind port 80 without root.
	if runtime.GOOS == "linux" {
		_ = exec.Command("setcap", "cap_net_bind_service=+ep", dm.BinPath("caddy")).Run()
	}
	if !pm.IsRunning("caddy") {
		caddyfilePath := filepath.Join(lighthouseDir, "config", "Caddyfile")
		if err := pm.Start("caddy", dm.BinPath("caddy"),
			[]string{"run", "--config", caddyfilePath},
			nil,
		); err != nil {
			pm.StopAll()
			return fmt.Errorf("✗ Failed to start Caddy: %w\n  → Check %s/logs/caddy.log for details", err, lighthouseDir)
		}
	}
	fmt.Println("  ✓ Caddy started")

	// Tor — include the bundle directory in the library search path so that
	// the bundled libssl / libcrypto are found at runtime.
	if !pm.IsRunning("tor") {
		torrcPath := filepath.Join(lighthouseDir, "config", "torrc")
		torBinDir := filepath.Dir(dm.BinPath("tor"))
		var torEnv []string
		switch runtime.GOOS {
		case "linux":
			torEnv = []string{"LD_LIBRARY_PATH=" + torBinDir}
		case "darwin":
			torEnv = []string{"DYLD_LIBRARY_PATH=" + torBinDir}
		}
		if err := pm.Start("tor", dm.BinPath("tor"), []string{"-f", torrcPath}, torEnv); err != nil {
			pm.StopAll()
			return fmt.Errorf("✗ Failed to start Tor: %w\n  → Check %s/logs/tor.log for details", err, lighthouseDir)
		}
	}
	fmt.Println("  ✓ Tor started")

	// 4. Wait for the hidden service hostname to appear.
	hiddenSvcDir := filepath.Join(lighthouseDir, "hidden_service")

	type onionResult struct {
		addr string
		err  error
	}
	onionCh := make(chan onionResult, 1)
	go func() {
		addr, err := torpkg.WaitForOnionAddress(ctx, hiddenSvcDir, 60*time.Second)
		onionCh <- onionResult{addr, err}
	}()

	select {
	case <-ctx.Done():
		pm.StopAll()
		return fmt.Errorf("startup cancelled")
	case result := <-onionCh:
		if result.err != nil {
			pm.StopAll()
			return fmt.Errorf("✗ Failed to get onion address: %w\n  → Check %s/logs/tor.log for details", result.err, lighthouseDir)
		}
		fmt.Println("  ✓ Hidden service ready")
		fmt.Println()
		fmt.Println("Local management UI:  http://localhost")
		fmt.Println()
		fmt.Println("Recipient address:")
		fmt.Println(result.addr)
		fmt.Println()
		fmt.Println("Share the address above with your recipient.")
		fmt.Println("Run 'lighthouse down' to stop.")
		return nil
	}
}

func runDockerUp() error {
	if err := downloadFiles(); err != nil {
		return err
	}
	dir, err := getLighthouseDir()
	if err != nil {
		return fmt.Errorf("failed to get lighthouse directory: %w", err)
	}
	if isRunning() {
		fmt.Println("Lighthouse is already running.")
		return nil
	}
	fmt.Println("Starting Lighthouse...")
	c := exec.Command("docker", "compose", "up", "-d")
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func downloadFiles() error {
	dir, err := getLighthouseDir()
	if err != nil {
		return fmt.Errorf("failed to get lighthouse directory: %w", err)
	}

	_, composeExists := os.Stat(filepath.Join(dir, "docker-compose.yml"))
	_, caddyExists := os.Stat(filepath.Join(dir, "Caddyfile"))
	if composeExists == nil && caddyExists == nil {
		return nil
	}

	composeUrl := "https://github.com/neozmmv/Lighthouse/releases/latest/download/docker-compose.yml"
	caddyfileUrl := "https://github.com/neozmmv/Lighthouse/releases/latest/download/Caddyfile"

	composeResp, err := http.Get(composeUrl)
	if err != nil {
		return fmt.Errorf("failed to download docker-compose.yml: %w", err)
	}
	defer composeResp.Body.Close()

	caddyfileResp, err := http.Get(caddyfileUrl)
	if err != nil {
		return fmt.Errorf("failed to download Caddyfile: %w", err)
	}
	defer caddyfileResp.Body.Close()

	composeFile, err := os.Create(filepath.Join(dir, "docker-compose.yml"))
	if err != nil {
		return fmt.Errorf("failed to create docker-compose.yml: %w", err)
	}
	defer composeFile.Close()
	if _, err = io.Copy(composeFile, composeResp.Body); err != nil {
		return fmt.Errorf("failed to write docker-compose.yml: %w", err)
	}

	caddyfile, err := os.Create(filepath.Join(dir, "Caddyfile"))
	if err != nil {
		return fmt.Errorf("failed to create Caddyfile: %w", err)
	}
	defer caddyfile.Close()
	if _, err = io.Copy(caddyfile, caddyfileResp.Body); err != nil {
		return fmt.Errorf("failed to write Caddyfile: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().BoolVar(&nativeFlag, "native", false, "Run without Docker (downloads required binaries on first run)")
}
