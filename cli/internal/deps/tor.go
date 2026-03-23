package deps

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type torDownloads struct {
	Version string `json:"version"`
}

func ensureTor(ctx context.Context, m *Manager) (string, error) {
	version, err := fetchLatestTorVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Tor version: %w", err)
	}

	archiveURL, isZip, err := torArchiveURL(version)
	if err != nil {
		return "", err
	}

	tmpFile, err := os.CreateTemp("", "tor-download-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := downloadFile(ctx, archiveURL, tmpPath); err != nil {
		return "", fmt.Errorf("failed to download Tor: %w", err)
	}

	torBinDir := filepath.Join(m.BaseDir, "tor-data", "bin")
	if err := os.MkdirAll(torBinDir, 0755); err != nil {
		return "", err
	}

	if isZip {
		if err := extractTorFromZip(tmpPath, torBinDir); err != nil {
			return "", fmt.Errorf("failed to extract Tor: %w", err)
		}
	} else {
		if err := extractTorFromTarGz(tmpPath, torBinDir); err != nil {
			return "", fmt.Errorf("failed to extract Tor: %w", err)
		}
	}

	torBin := filepath.Join(torBinDir, "tor")
	if runtime.GOOS == "windows" {
		torBin = filepath.Join(torBinDir, "tor.exe")
	}
	if err := os.Chmod(torBin, 0755); err != nil {
		return "", fmt.Errorf("failed to make tor executable: %w", err)
	}

	return version, nil
}

func fetchLatestTorVersion(ctx context.Context) (string, error) {
	url := "https://aus1.torproject.org/torbrowser/update_3/release/downloads.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Tor update API returned status %d", resp.StatusCode)
	}

	var data torDownloads
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if data.Version == "" {
		return "", fmt.Errorf("empty version in Tor update API response")
	}
	return data.Version, nil
}

func torArchiveURL(version string) (url string, isZip bool, err error) {
	// Since Tor Browser 14.x the version is embedded in the filename,
	// e.g. tor-expert-bundle-linux-x86_64-15.0.7.tar.gz
	base := "https://archive.torproject.org/tor-package-archive/torbrowser/" + version + "/"
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		return base + "tor-expert-bundle-linux-x86_64-" + version + ".tar.gz", false, nil
	case "linux/arm64":
		return base + "tor-expert-bundle-linux-aarch64-" + version + ".tar.gz", false, nil
	case "windows/amd64":
		return base + "tor-expert-bundle-windows-x86_64-" + version + ".zip", true, nil
	case "darwin/amd64":
		return base + "tor-expert-bundle-macos-x86_64-" + version + ".tar.gz", false, nil
	case "darwin/arm64":
		return base + "tor-expert-bundle-macos-aarch64-" + version + ".tar.gz", false, nil
	default:
		return "", false, fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

// extractTorFromTarGz extracts the entire tor/ directory from the expert bundle.
func extractTorFromTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if !strings.HasPrefix(header.Name, "tor/") {
			continue
		}

		relPath := strings.TrimPrefix(header.Name, "tor/")
		if relPath == "" {
			continue
		}

		destPath := filepath.Join(destDir, filepath.FromSlash(relPath))

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode)|0644)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			out.Close()
			if copyErr != nil {
				return copyErr
			}
		case tar.TypeSymlink:
			os.Remove(destPath)
			// Non-fatal: ignore symlink errors on platforms that don't support them
			_ = os.Symlink(header.Linkname, destPath)
		}
	}
	return nil
}

func extractTorFromZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "tor/") {
			continue
		}
		relPath := strings.TrimPrefix(f.Name, "tor/")
		if relPath == "" {
			continue
		}
		destPath := filepath.Join(destDir, filepath.FromSlash(relPath))

		if f.FileInfo().IsDir() {
			os.MkdirAll(destPath, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode()|0644)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}
