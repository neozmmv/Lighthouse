package tor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WaitForOnionAddress polls for the hidden service hostname file every 500 ms
// until the address is available or the timeout (or context) expires.
func WaitForOnionAddress(ctx context.Context, hiddenServiceDir string, timeout time.Duration) (string, error) {
	hostnameFile := filepath.Join(hiddenServiceDir, "hostname")
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		data, err := os.ReadFile(hostnameFile)
		if err == nil {
			addr := strings.TrimSpace(string(data))
			if addr != "" {
				return addr, nil
			}
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}

	return "", fmt.Errorf("timed out after %s waiting for Tor hidden service address", timeout)
}
