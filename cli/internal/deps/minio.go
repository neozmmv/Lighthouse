package deps

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
)

func ensureMinio(ctx context.Context, m *Manager) (string, error) {
	urls := map[string]string{
		"linux/amd64":   "https://dl.min.io/server/minio/release/linux-amd64/minio",
		"linux/arm64":   "https://dl.min.io/server/minio/release/linux-arm64/minio",
		"windows/amd64": "https://dl.min.io/server/minio/release/windows-amd64/minio.exe",
		"darwin/amd64":  "https://dl.min.io/server/minio/release/darwin-amd64/minio",
		"darwin/arm64":  "https://dl.min.io/server/minio/release/darwin-arm64/minio",
	}

	key := runtime.GOOS + "/" + runtime.GOARCH
	url, ok := urls[key]
	if !ok {
		return "", fmt.Errorf("unsupported platform: %s", key)
	}

	dest := m.BinPath("minio")
	if err := downloadFile(ctx, url, dest); err != nil {
		return "", err
	}
	if err := os.Chmod(dest, 0755); err != nil {
		return "", fmt.Errorf("failed to make minio executable: %w", err)
	}
	return "latest", nil
}

// downloadFile downloads url to dest, respecting context cancellation.
func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status %d from %s", resp.StatusCode, url)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}
