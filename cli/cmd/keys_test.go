package cmd

import (
	"testing"
)

func TestRSAKeys(t *testing.T) {
	err := generateRSAKeyPair()
	if err != nil {
		t.Fatalf("Error generating RSA key pair: %v", err)
	}
}
