package deps

import (
	"context"
	"fmt"
	"os"
	"runtime"
)

func ensureBackend(ctx context.Context, m *Manager) (string, error) {
	release, err := fetchLatestGithubRelease(ctx, "neozmmv/Lighthouse")
	if err != nil {
		return "", fmt.Errorf("failed to fetch backend release info: %w", err)
	}

	assetName := backendAssetName()

	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return "", fmt.Errorf("no backend asset found for %s in release %s", assetName, release.TagName)
	}

	dest := m.BinPath("lighthouse-backend")
	if err := downloadFile(ctx, downloadURL, dest); err != nil {
		return "", fmt.Errorf("failed to download backend: %w", err)
	}
	if err := os.Chmod(dest, 0755); err != nil {
		return "", fmt.Errorf("failed to make backend executable: %w", err)
	}
	return release.TagName, nil
}

func backendAssetName() string {
	name := fmt.Sprintf("lighthouse-backend-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}
