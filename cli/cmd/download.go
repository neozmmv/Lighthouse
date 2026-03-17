package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

var here bool
var remove bool

func downloadFile(f FileInfo, destDir string) error {
    apiURL := fmt.Sprintf("http://localhost:8000/api/files/%s/download", f.FileId)

    resp, err := http.Get(apiURL)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("Error obtaining download URL: status %d", resp.StatusCode)
    }

    var result struct {
        URL      string `json:"url"`
        Filename string `json:"filename"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return err
    }

	// download the file from the obtained URL
    fileResp, err := http.Get(result.URL)
    if err != nil {
        return err
    }
    defer fileResp.Body.Close()

    if fileResp.StatusCode != http.StatusOK {
        return fmt.Errorf("Error downloading file: status %d", fileResp.StatusCode)
    }

    // saves to disk based on the --here flag, if not set saves to ~/Downloads
    destPath := filepath.Join(destDir, result.Filename)
    out, err := os.Create(destPath)
    if err != nil {
        return err
    }
    defer out.Close()

    _, err = io.Copy(out, fileResp.Body)
    if err != nil {
        return err
    }

    fmt.Printf("Saved to: %s\n", destPath)
    return nil
}

func deleteFile(fileId string) error {
    apiURL := fmt.Sprintf("http://localhost:8000/api/files/%s", url.PathEscape(fileId))

    req, err := http.NewRequest(http.MethodDelete, apiURL, nil)
    if err != nil {
        return err
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("Error deleting file: status %d", resp.StatusCode)
    }

    return nil
}

var downloadCmd = &cobra.Command{
    Use:   "download [index]",
    Short: "Downloads a file from the bucket using the index returned by 'lighthouse files'",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        idx, err := strconv.Atoi(args[0])
        if err != nil {
            return fmt.Errorf("Invalid index: %s", args[0])
        }

        files, err := getFiles()
        if err != nil {
            return err
        }

        if idx < 0 || idx >= len(files) {
            return fmt.Errorf("Index out of bounds (0-%d)", len(files)-1)
        }

        selected := files[idx]

        // saving directory based on --here flag
        var destDir string
        if here {
            destDir = "."
        } else {
            home, err := os.UserHomeDir()
            if err != nil {
                return err
            }
            destDir = filepath.Join(home, "Downloads")
        }

        if err := downloadFile(selected, destDir); err != nil {
			return err
		}

		if remove {
			if err := deleteFile(selected.FileId); err != nil {
                return fmt.Errorf("Error removing file from the bucket: %w", err)
            }
		}
		return nil
    },
}

func init() {
    rootCmd.AddCommand(downloadCmd)
    downloadCmd.Flags().BoolVar(&here, "here", false, "save in current directory (defaults to ~/Downloads)")
	downloadCmd.Flags().BoolVar(&remove, "remove", false, "removes the file from the bucket after downloading")
}