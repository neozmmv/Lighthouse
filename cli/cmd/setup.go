package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// MINIO AND CADDY ALREADY ARE THE LATEST VERSIONS.
// TOR DOESN'T, SO WE DO A LITTLE WORKAROUND HERE
// TO KEEP IT WORKING, I'LL HARDCODE A WORKING VERSION OF TOR
// TO AVOID BREAKING THE APP IF SOMETHING CHANGES
// IT WILL BE UPDATED MANUALLY IF NECESSARY, ON RELEASE NOTHING WILL CHANGE
const (
	// CURRENT VERSION OF TOR THAT WORKS WITH LIGHTHOUSE
	torVersion       = "13.5.9"
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
HiddenServicePort 80 127.0.0.1:80
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

	caddyfile := `:80 {
    handle /api/files* {
        respond 404
    }
    handle /api/* {
        reverse_proxy 127.0.0.1:8000
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
        file_server
    }
}

:4405 {
    handle /api/* {
        reverse_proxy 127.0.0.1:8000
    }
    handle {
        root * {$LIGHTHOUSE_STATIC_DIR}
        file_server
    }
}
`
	return os.WriteFile(filepath.Join(dir, "Caddyfile"), []byte(caddyfile), 0600)
}

func runSetup() error {
	fmt.Println("First run — setting up Lighthouse...")

	if err := downloadBinaries(); err != nil {
		return err
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

	// mark as initialized
	dir, err := getLighthouseDir()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "initialized"), []byte("1"), 0600); err != nil {
		return fmt.Errorf("failed to mark as initialized: %w", err)
	}

	fmt.Println("Setup complete.")
	return nil
}
