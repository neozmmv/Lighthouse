package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

const githubBaseURL = "https://github.com/neozmmv/Lighthouse/releases/latest/download"

func downloadToFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response status: %s", resp.Status)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Lighthouse to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Downloading latest installer...")

		tmpPath := filepath.Join(os.TempDir(), "LighthouseSetup.exe")
		if err := downloadToFile(githubBaseURL+"/LighthouseSetup.exe", tmpPath); err != nil {
			return fmt.Errorf("failed to download installer: %w", err)
		}
		defer os.Remove(tmpPath)

		fmt.Println("Running installer...")
		c := exec.Command(tmpPath)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin
		if err := c.Run(); err != nil {
			return fmt.Errorf("failed to run installer: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
