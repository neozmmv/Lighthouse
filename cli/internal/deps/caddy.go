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
	"runtime"
	"strings"
)

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func ensureCaddy(ctx context.Context, m *Manager) (string, error) {
	release, err := fetchLatestGithubRelease(ctx, "caddyserver/caddy")
	if err != nil {
		return "", fmt.Errorf("failed to fetch Caddy release info: %w", err)
	}

	version := strings.TrimPrefix(release.TagName, "v")

	// Caddy uses "mac" instead of "darwin" and "amd64"/"arm64" unchanged.
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "mac"
	}

	var assetName string
	if runtime.GOOS == "windows" {
		assetName = fmt.Sprintf("caddy_%s_%s_%s.zip", version, osName, runtime.GOARCH)
	} else {
		assetName = fmt.Sprintf("caddy_%s_%s_%s.tar.gz", version, osName, runtime.GOARCH)
	}

	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return "", fmt.Errorf("no Caddy asset found for %s", assetName)
	}

	tmpFile, err := os.CreateTemp("", "caddy-download-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := downloadFile(ctx, downloadURL, tmpPath); err != nil {
		return "", fmt.Errorf("failed to download Caddy: %w", err)
	}

	dest := m.BinPath("caddy")

	if runtime.GOOS == "windows" {
		if err := extractFromZip(tmpPath, "caddy.exe", dest); err != nil {
			return "", fmt.Errorf("failed to extract Caddy: %w", err)
		}
	} else {
		if err := extractFromTarGz(tmpPath, "caddy", dest); err != nil {
			return "", fmt.Errorf("failed to extract Caddy: %w", err)
		}
	}

	if err := os.Chmod(dest, 0755); err != nil {
		return "", fmt.Errorf("failed to make caddy executable: %w", err)
	}

	return release.TagName, nil
}

func fetchLatestGithubRelease(ctx context.Context, repo string) (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d for %s", resp.StatusCode, repo)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

// extractFromTarGz extracts a single file by basename from a .tar.gz archive.
func extractFromTarGz(archivePath, targetName, dest string) error {
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
		if header.Typeflag != tar.TypeReg {
			continue
		}
		// Match by basename
		name := header.Name
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		if name != targetName {
			continue
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, tr)
		out.Close()
		return copyErr
	}
	return fmt.Errorf("binary %q not found in archive", targetName)
}

// extractFromZip extracts a single file by basename from a .zip archive.
func extractFromZip(archivePath, targetName, dest string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		name := f.Name
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		if name != targetName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		out.Close()
		rc.Close()
		return copyErr
	}
	return fmt.Errorf("file %q not found in zip", targetName)
}
