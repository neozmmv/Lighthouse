package cmd

import (
	"fmt"
	"testing"
)

func TestGetCloudflareTunnelURL(t *testing.T) {
	// result, cmd, err
	result, _, err := getCloudflareTunnelURL()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	fmt.Printf("URL: %s\n", result)
}

func TestKillCloudflareTunnel(t *testing.T) {
	// result, cmd, err
	_, cmd, err := getCloudflareTunnelURL()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Error killing cloudflared process: %v", err)
	}
}
