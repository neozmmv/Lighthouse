package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MINIO AND CADDY ALREADY RESOLVE THE LATEST VERSION VIA THEIR DOWNLOAD URLS.
// TOR DOESN'T, SO THE VERSION IS HARDCODED HERE AND UPDATED MANUALLY ON RELEASE.
const (
	torVersion       = "15.0.8"
	torDownloadURL   = "https://dist.torproject.org/torbrowser/" + torVersion + "/tor-expert-bundle-windows-x86_64-" + torVersion + ".tar.gz"
	minioDownloadURL = "https://dl.min.io/server/minio/release/windows-amd64/minio.exe"
	caddyDownloadURL = "https://caddyserver.com/api/download?os=windows&arch=amd64&idempotency=1"
)

func downloadBinaries() error {
	binDir, err := getBinDir()
	if err != nil {
		return err
	}

	fmt.Println("Downloading Tor...")
	if err := downloadTor(binDir); err != nil {
		return fmt.Errorf("failed to download Tor: %w", err)
	}

	fmt.Println("Downloading MinIO...")
	if err := downloadBinary(minioDownloadURL, filepath.Join(binDir, "minio.exe")); err != nil {
		return fmt.Errorf("failed to download MinIO: %w", err)
	}

	fmt.Println("Downloading Caddy...")
	if err := downloadBinary(caddyDownloadURL, filepath.Join(binDir, "caddy.exe")); err != nil {
		return fmt.Errorf("failed to download Caddy: %w", err)
	}

	return nil
}

func downloadBinary(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func extractFrontend() error {
	dir, err := getLighthouseDir()
	if err != nil {
		return err
	}

	frontendDir := filepath.Join(dir, "frontend")
	if err := os.MkdirAll(frontendDir, 0755); err != nil {
		return err
	}

	// walk embedded frontend files and extract them
	return fs.WalkDir(frontendFiles, "frontend", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// strip the "frontend/" prefix to get the relative path
		relPath := strings.TrimPrefix(path, "frontend/")
		if relPath == "" {
			return nil
		}

		destPath := filepath.Join(frontendDir, filepath.FromSlash(relPath))

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		data, err := frontendFiles.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, 0644)
	})
}

func extractBackend() error {
	binDir, err := getBinDir()
	if err != nil {
		return err
	}

	dest := filepath.Join(binDir, "backend.exe")
	if err := os.WriteFile(dest, backendBinary, 0755); err != nil {
		return fmt.Errorf("failed to extract backend: %w", err)
	}

	return nil
}

func downloadTor(binDir string) error {
	// download tor expert bundle tar.gz
	tmpFile := filepath.Join(binDir, "tor.tar.gz")
	if err := downloadBinary(torDownloadURL, tmpFile); err != nil {
		return err
	}
	defer os.Remove(tmpFile)

	f, err := os.Open(tmpFile)
	if err != nil {
		return err
	}
	defer f.Close()

	// tor expert bundle uses gzip
	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to open gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if filepath.Base(hdr.Name) == "tor.exe" {
			out, err := os.Create(filepath.Join(binDir, "tor.exe"))
			if err != nil {
				return err
			}
			defer out.Close()
			_, err = io.Copy(out, tr)
			return err
		}
	}

	return fmt.Errorf("tor.exe not found in archive")
}

func writeTorrc() error {
	dir, err := getLighthouseDir()
	if err != nil {
		return err
	}

	torDir := filepath.Join(dir, "tor")
	if err := os.MkdirAll(torDir, 0755); err != nil {
		return err
	}

	torrc := fmt.Sprintf(`DataDirectory %s
HiddenServiceDir %s
HiddenServicePort 80 127.0.0.1:8080
SocksPort 9050
Log notice stderr
`, torDir, filepath.Join(torDir, "hidden_service"))

	return os.WriteFile(filepath.Join(torDir, "torrc"), []byte(torrc), 0600)
}

func writeCaddyfile() error {
	dir, err := getLighthouseDir()
	if err != nil {
		return err
	}

	caddyfile := `:8080 {
    handle /api/files* {
        respond 404
    }
    handle /api/* {
        reverse_proxy 127.0.0.1:8000 {
            transport http {
                read_timeout 0
                write_timeout 0
                response_header_timeout 0
            }
        }
    }
    handle /files* {
        respond 403
    }
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
        root * {$LIGHTHOUSE_STATIC_DIR}
        header *.js Content-Type application/javascript
        header *.css Content-Type text/css
        file_server
        try_files {path} /index.html
    }
}

:4405 {
    handle /api/* {
        reverse_proxy 127.0.0.1:8000 {
            transport http {
                read_timeout 0
                write_timeout 0
                response_header_timeout 0
            }
        }
    }
    handle {
        root * {$LIGHTHOUSE_STATIC_DIR}
        header *.js Content-Type application/javascript
        header *.css Content-Type text/css
        file_server
        try_files {path} /index.html
    }
}
`
	return os.WriteFile(filepath.Join(dir, "Caddyfile"), []byte(caddyfile), 0600)
}

func initMinIOBucket(cfg *Config) error {
	client, err := minio.New("127.0.0.1:9000", &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioUser, cfg.MinioPass, ""),
		Secure: false,
	})
	if err != nil {
		return fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// wait for MinIO to be ready
	for i := 0; i < 10; i++ {
		_, err := client.ListBuckets(context.Background())
		if err == nil {
			break
		}
		if i == 9 {
			return fmt.Errorf("MinIO did not become ready in time: %w", err)
		}
		time.Sleep(1 * time.Second)
	}

	exists, err := client.BucketExists(context.Background(), "lighthouse")
	if err != nil {
		return fmt.Errorf("failed to check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(context.Background(), "lighthouse", minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}

func runSetup() error {
	fmt.Println("First run — setting up Lighthouse...")

	if err := downloadBinaries(); err != nil {
		return err
	}

	if err := extractBackend(); err != nil {
		return fmt.Errorf("failed to extract backend: %w", err)
	}

	if err := extractFrontend(); err != nil {
		return fmt.Errorf("failed to extract frontend: %w", err)
	}

	if err := writeTorrc(); err != nil {
		return fmt.Errorf("failed to write torrc: %w", err)
	}

	if err := writeCaddyfile(); err != nil {
		return fmt.Errorf("failed to write Caddyfile: %w", err)
	}

	// generate MinIO credentials
	minioPass, err := generateSecret(16)
	if err != nil {
		return fmt.Errorf("failed to generate MinIO password: %w", err)
	}

	cfg := &Config{
		MinioUser: "lighthouse",
		MinioPass: minioPass,
	}
	if err := saveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// start MinIO temporarily to create the bucket
	fmt.Println("Initializing MinIO bucket...")
	binDir, err := getBinDir()
	if err != nil {
		return err
	}
	dir, err := getLighthouseDir()
	if err != nil {
		return err
	}

	minioProc := exec.Command(
		filepath.Join(binDir, "minio.exe"),
		"server",
		filepath.Join(dir, "data", "minio"),
		"--address", "127.0.0.1:9000",
		"--console-address", "127.0.0.1:9001",
	)
	minioProc.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	minioProc.Env = append(os.Environ(),
		"MINIO_ROOT_USER="+cfg.MinioUser,
		"MINIO_ROOT_PASSWORD="+cfg.MinioPass,
	)
	if err := minioProc.Start(); err != nil {
		return fmt.Errorf("failed to start MinIO for setup: %w", err)
	}
	defer minioProc.Process.Kill()

	if err := initMinIOBucket(cfg); err != nil {
		return err
	}

	// mark as initialized
	if err := os.WriteFile(filepath.Join(dir, "initialized"), []byte("1"), 0600); err != nil {
		return fmt.Errorf("failed to mark as initialized: %w", err)
	}

	fmt.Println("Setup complete.")
	return nil
}
