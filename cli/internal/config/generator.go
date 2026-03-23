package config

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Generator manages Lighthouse config files under ~/.lighthouse/config/.
type Generator struct {
	ConfigDir    string
	HiddenSvcDir string
	DataDir      string
	TorDataDir   string
	FrontendDir  string
}

// NewGenerator creates a Generator, ensuring required directories exist.
func NewGenerator() (*Generator, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	base := filepath.Join(home, ".lighthouse")
	g := &Generator{
		ConfigDir:    filepath.Join(base, "config"),
		HiddenSvcDir: filepath.Join(base, "hidden_service"),
		DataDir:      filepath.Join(base, "data"),
		TorDataDir:   filepath.Join(base, "tor-data"),
		FrontendDir:  filepath.Join(base, "frontend"),
	}
	for _, d := range []string{g.ConfigDir, g.HiddenSvcDir, g.DataDir, g.TorDataDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}
	return g, nil
}

// EnsureAll creates all config files that don't already exist.
func (g *Generator) EnsureAll() error {
	if err := g.ensureSecrets(); err != nil {
		return err
	}
	if err := g.ensureTorrc(); err != nil {
		return err
	}
	if err := g.ensureCaddyfile(); err != nil {
		return err
	}
	return g.ensureBackendEnv()
}

func (g *Generator) ensureSecrets() error {
	secretsPath := filepath.Join(g.ConfigDir, "secrets")
	if _, err := os.Stat(secretsPath); err == nil {
		return nil // already exists; never regenerate
	}
	secret, err := generateSecret(32)
	if err != nil {
		return fmt.Errorf("failed to generate secret: %w", err)
	}
	content := "MINIO_ROOT_PASSWORD=" + secret + "\n"
	return os.WriteFile(secretsPath, []byte(content), 0600)
}

func (g *Generator) readSecret() (string, error) {
	data, err := os.ReadFile(filepath.Join(g.ConfigDir, "secrets"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MINIO_ROOT_PASSWORD=") {
			return strings.TrimPrefix(line, "MINIO_ROOT_PASSWORD="), nil
		}
	}
	return "", fmt.Errorf("MINIO_ROOT_PASSWORD not found in secrets file")
}

func (g *Generator) ensureTorrc() error {
	torrcPath := filepath.Join(g.ConfigDir, "torrc")
	if _, err := os.Stat(torrcPath); err == nil {
		return nil
	}

	torStateDir := filepath.Join(g.TorDataDir, "state")
	if err := os.MkdirAll(torStateDir, 0700); err != nil {
		return err
	}
	// Tor requires strict permissions on the hidden service directory.
	if err := os.Chmod(g.HiddenSvcDir, 0700); err != nil {
		return err
	}

	content := fmt.Sprintf(
		"DataDirectory %s\nHiddenServiceDir %s\nHiddenServicePort 80 127.0.0.1:4405\n",
		torStateDir, g.HiddenSvcDir,
	)
	return os.WriteFile(torrcPath, []byte(content), 0644)
}

func (g *Generator) ensureCaddyfile() error {
	caddyfilePath := filepath.Join(g.ConfigDir, "Caddyfile")
	if _, err := os.Stat(caddyfilePath); err == nil {
		return nil
	}
	// Mirror the production Docker Caddyfile but with localhost addresses.
	// Port 4405: Tor-facing (HiddenServicePort 80 → 127.0.0.1:4405). Has security
	//            rules that block file management routes from external Tor clients.
	// Port 80:   Local management only — no restrictions, full frontend + API access.
	//            Caddy binary must have cap_net_bind_service (set automatically on Linux).
	content := fmt.Sprintf(`:4405 {
	# Block the file management UI and API from external Tor clients
	handle /api/files* {
		respond 404
	}

	handle /files* {
		respond 403
	}

	handle /api/* {
		reverse_proxy 127.0.0.1:8000
	}

	# Proxy presigned upload PUTs to MinIO — the browser sends these directly
	# to the onion address which Tor delivers here, not to the backend.
	handle /lighthouse/* {
		reverse_proxy 127.0.0.1:9000 {
			header_up Host {host}
			transport http {
				read_timeout 0
				write_timeout 0
				response_header_timeout 0
			}
		}
	}

	handle {
		root * %s
		try_files {path} /index.html
		file_server
	}
}

# Local management interface — accessible at http://localhost
:80 {
	handle /api/* {
		reverse_proxy 127.0.0.1:8000
	}

	handle {
		root * %s
		try_files {path} /index.html
		file_server
	}
}
`, g.FrontendDir, g.FrontendDir)
	return os.WriteFile(caddyfilePath, []byte(content), 0644)
}

func (g *Generator) ensureBackendEnv() error {
	envPath := filepath.Join(g.ConfigDir, "backend.env")
	if _, err := os.Stat(envPath); err == nil {
		return nil
	}
	secret, err := g.readSecret()
	if err != nil {
		return fmt.Errorf("failed to read secret for backend env: %w", err)
	}
	content := fmt.Sprintf(
		"MINIO_ENDPOINT=http://127.0.0.1:9000\nMINIO_ACCESS_KEY=lighthouse\nMINIO_SECRET_KEY=%s\nMINIO_BUCKET=lighthouse\nS3_USE_SSL=false\n",
		secret,
	)
	return os.WriteFile(envPath, []byte(content), 0600)
}

// MinioEnv returns the environment variables needed to start MinIO server.
func (g *Generator) MinioEnv() ([]string, error) {
	secret, err := g.readSecret()
	if err != nil {
		return nil, err
	}
	return []string{
		"MINIO_ROOT_USER=lighthouse",
		"MINIO_ROOT_PASSWORD=" + secret,
	}, nil
}

// BackendEnv reads backend.env and returns it as a slice of KEY=VALUE strings.
func (g *Generator) BackendEnv() ([]string, error) {
	data, err := os.ReadFile(filepath.Join(g.ConfigDir, "backend.env"))
	if err != nil {
		return nil, err
	}
	var env []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			env = append(env, line)
		}
	}
	return env, nil
}

func generateSecret(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	out := make([]byte, length)
	for i, b := range raw {
		out[i] = charset[int(b)%len(charset)]
	}
	return string(out), nil
}
