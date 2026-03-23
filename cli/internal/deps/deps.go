package deps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Manager handles downloading and verifying third-party binaries.
type Manager struct {
	BinDir  string // ~/.lighthouse/bin/
	BaseDir string // ~/.lighthouse/
}

// NewManager creates a new dependency manager.
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Join(home, ".lighthouse")
	binDir := filepath.Join(baseDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create bin directory: %w", err)
	}
	return &Manager{BinDir: binDir, BaseDir: baseDir}, nil
}

// BinPath returns the full path to a binary.
// Tor is a special case: it lives in tor-data/bin/ alongside its shared libraries.
func (m *Manager) BinPath(dep string) string {
	if dep == "tor" {
		bin := "tor"
		if runtime.GOOS == "windows" {
			bin = "tor.exe"
		}
		return filepath.Join(m.BaseDir, "tor-data", "bin", bin)
	}

	name := dep
	if runtime.GOOS == "windows" {
		name = dep + ".exe"
	}
	return filepath.Join(m.BinDir, name)
}

// IsInstalled checks whether a binary exists on disk.
func (m *Manager) IsInstalled(dep string) bool {
	_, err := os.Stat(m.BinPath(dep))
	return err == nil
}

// EnsureAll downloads all missing dependencies, printing progress.
func (m *Manager) EnsureAll(ctx context.Context) error {
	fmt.Println("→ Checking dependencies...")

	type entry struct {
		name   string
		ensure func(context.Context, *Manager) (string, error)
	}

	all := []entry{
		{"minio", ensureMinio},
		{"caddy", ensureCaddy},
		{"tor", ensureTor},
		{"lighthouse-backend", ensureBackend},
	}

	for _, d := range all {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if m.IsInstalled(d.name) {
			fmt.Printf("  ✓ %s already installed\n", d.name)
			continue
		}

		fmt.Printf("  ↓ Downloading %s...", d.name)
		version, err := d.ensure(ctx, m)
		if err != nil {
			fmt.Println()
			return fmt.Errorf("✗ Failed to download %s: %v\n  → Check your internet connection and try again", d.name, err)
		}
		fmt.Printf(" done (%s)\n", version)
	}

	// Frontend is a directory, not a binary — handled separately.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if m.IsFrontendInstalled() {
		fmt.Println("  ✓ frontend already installed")
	} else {
		fmt.Printf("  ↓ Downloading frontend...")
		version, err := ensureFrontend(ctx, m)
		if err != nil {
			fmt.Println()
			fmt.Printf("  ⚠ Frontend not available (%v)\n", err)
			fmt.Println("    → Run 'npm run build' in the frontend/ directory, then copy dist/ to ~/.lighthouse/frontend/")
		} else {
			fmt.Printf(" done (%s)\n", version)
		}
	}

	return nil
}

// EnsureOne downloads a specific dependency if not already installed.
func (m *Manager) EnsureOne(ctx context.Context, dep string) error {
	if m.IsInstalled(dep) {
		return nil
	}
	switch dep {
	case "minio":
		_, err := ensureMinio(ctx, m)
		return err
	case "caddy":
		_, err := ensureCaddy(ctx, m)
		return err
	case "tor":
		_, err := ensureTor(ctx, m)
		return err
	case "lighthouse-backend":
		_, err := ensureBackend(ctx, m)
		return err
	default:
		return fmt.Errorf("unknown dependency: %s", dep)
	}
}
