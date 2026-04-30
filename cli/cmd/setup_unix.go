//go:build !windows

package cmd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

func generateRSAKeyPair() error {
	dir, err := getLighthouseDir()
	if err != nil {
		return fmt.Errorf("failed to get lighthouse directory: %w", err)
	}
	keysDir := filepath.Join(dir, "keys")
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return fmt.Errorf("failed to create keys directory: %w", err)
	}

	privatePath := filepath.Join(keysDir, "private.pem")
	publicPath := filepath.Join(keysDir, "public.pem")

	_, errPub := os.Stat(publicPath)
	_, errPriv := os.Stat(privatePath)
	if errPub == nil && errPriv == nil {
		return nil
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key pair: %w", err)
	}

	privateKeyFile, err := os.OpenFile(privatePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer privateKeyFile.Close()

	if err := pem.Encode(privateKeyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	publicKeyFile, err := os.OpenFile(publicPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return fmt.Errorf("failed to create public key file: %w", err)
	}
	defer publicKeyFile.Close()

	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	if err := pem.Encode(publicKeyFile, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

func runSetup() error {
	dir, err := getLighthouseDir()
	if err != nil {
		return err
	}

	_, composeErr := os.Stat(filepath.Join(dir, "docker-compose.yml"))
	_, caddyErr := os.Stat(filepath.Join(dir, "Caddyfile"))
	_, cloudflareErr := os.Stat(filepath.Join(dir, "docker-compose.cloudflare.yml"))
	keysDir := filepath.Join(dir, "keys")
	_, pubKeyErr := os.Stat(filepath.Join(keysDir, "public.pem"))
	_, privKeyErr := os.Stat(filepath.Join(keysDir, "private.pem"))
	keysExist := pubKeyErr == nil && privKeyErr == nil

	if composeErr == nil && caddyErr == nil && cloudflareErr == nil && keysExist {
		return nil
	}

	fmt.Println("First run — downloading Lighthouse configuration...")

	if composeErr != nil {
		if err := downloadToFile(
			githubBaseURL+"/docker-compose.yml",
			filepath.Join(dir, "docker-compose.yml"),
		); err != nil {
			return fmt.Errorf("failed to download docker-compose.yml: %w", err)
		}
	}

	if caddyErr != nil {
		if err := downloadToFile(
			githubBaseURL+"/Caddyfile",
			filepath.Join(dir, "Caddyfile"),
		); err != nil {
			return fmt.Errorf("failed to download Caddyfile: %w", err)
		}
	}

	if cloudflareErr != nil {
		if err := downloadToFile(
			githubBaseURL+"/docker-compose.cloudflare.yml",
			filepath.Join(dir, "docker-compose.cloudflare.yml"),
		); err != nil {
			return fmt.Errorf("failed to download docker-compose.cloudflare.yml: %w", err)
		}
	}

	if !keysExist {
		fmt.Println("Generating RSA key pair...")
		if err := generateRSAKeyPair(); err != nil {
			return fmt.Errorf("failed to generate RSA key pair: %w", err)
		}
	}

	fmt.Println("Setup complete.")
	return nil
}
