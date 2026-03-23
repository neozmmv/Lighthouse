package deps

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// IsFrontendInstalled returns true when the extracted frontend assets exist.
func (m *Manager) IsFrontendInstalled() bool {
	_, err := os.Stat(filepath.Join(m.FrontendDir(), "index.html"))
	return err == nil
}

// FrontendDir returns the directory where static frontend assets are stored.
func (m *Manager) FrontendDir() string {
	return filepath.Join(m.BaseDir, "frontend")
}

func ensureFrontend(ctx context.Context, m *Manager) (string, error) {
	release, err := fetchLatestGithubRelease(ctx, "neozmmv/Lighthouse")
	if err != nil {
		return "", fmt.Errorf("failed to fetch frontend release info: %w", err)
	}

	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == "frontend.zip" {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return "", fmt.Errorf("no frontend.zip asset found in release %s", release.TagName)
	}

	tmpFile, err := os.CreateTemp("", "frontend-download-*.zip")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := downloadFile(ctx, downloadURL, tmpPath); err != nil {
		return "", fmt.Errorf("failed to download frontend: %w", err)
	}

	destDir := m.FrontendDir()
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create frontend directory: %w", err)
	}

	if err := extractFrontendZip(tmpPath, destDir); err != nil {
		return "", fmt.Errorf("failed to extract frontend: %w", err)
	}

	return release.TagName, nil
}

// extractFrontendZip extracts the contents of the zip's "dist/" directory into destDir.
func extractFrontendZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	const prefix = "dist/"
	found := false
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}
		relPath := strings.TrimPrefix(f.Name, prefix)
		if relPath == "" {
			continue
		}
		found = true
		dest := filepath.Join(destDir, filepath.FromSlash(relPath))

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dest, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
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
		if copyErr != nil {
			return copyErr
		}
	}

	if !found {
		return fmt.Errorf("no dist/ directory found in frontend.zip")
	}
	return nil
}
