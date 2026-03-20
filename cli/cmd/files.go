package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

type FileInfo struct {
    FileId     string `json:"file_id"`
    Filename   string `json:"filename"`
    Size       int64  `json:"size"`
    UploadedAt string `json:"uploaded_at"`
}

func getFiles() ([]FileInfo, error) {
	resp, err := http.Get("http://localhost:4406/api/files")
	if err != nil {
		fmt.Println("Error fetching files:", err)
		return nil, err
	}
	defer resp.Body.Close()

	var files []FileInfo
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
    	return nil, err
	}
	for i, f := range files {
		var size int64
		// prints file size according to size in MB, KB or B
		if f.Size >= 1024*1024 {
			size = f.Size / (1024 * 1024)
			fmt.Printf("[%d] %s (%dMB)\n", i, f.Filename, size)
		} else if f.Size < 1024 {
			size = f.Size
			fmt.Printf("[%d] %s (%dB)\n", i, f.Filename, size)
		} else {
			size = f.Size / 1024
			fmt.Printf("[%d] %s (%dKB)\n", i, f.Filename, size)
		}
	}
	return files, nil
}

var filesCmd = &cobra.Command{
	Use: "files",
	Short: "Shows the files in your Lighthouse bucket.",
	Run: func(cmd *cobra.Command, args []string) {
		if !isRunning() {
			fmt.Println("Lighthouse is not running. Please start it with 'lighthouse up' command.")
			return
		}
		if _, err := getFiles(); err != nil {
			fmt.Println("Error:", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(filesCmd)
}