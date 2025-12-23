package yt

import (
	"fmt"
	"log"
	"mvp-clipper/internal/utils"
	"os"
	"path/filepath"
	"strings"
)

func DownloadVideo(url, outputDir string) (string, error) {
	// Extract video ID
	videoID := ExtractVideoID(url)
	if videoID == "" {
		return "", fmt.Errorf("invalid YouTube URL: cannot extract video ID")
	}

	if outputDir == "" {
		outputDir = "tmp/downloads"
	}

	// Ensure directory exists
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", outputDir, err)
	}

	// Output path with video ID
	output := filepath.Join(outputDir, fmt.Sprintf("%s.mp4", videoID))

	// Download best video+audio and merge to mp4
	cmd := []string{
		"yt-dlp",
		"-f", "bv*+ba/b",           // best video + audio, fallback to best
		"--merge-output-format", "mp4", // force mp4 output
		"-o", output,
		url,
	}

	log.Println("Running:", cmd)
	_, err := utils.Exec(cmd...)
	if err != nil {
		return "", fmt.Errorf("download failed: %v", err)
	}

	// Check if file exists (yt-dlp might add extension)
	if _, err := os.Stat(output); err == nil {
		return output, nil
	}

	// Try to find the downloaded file
	matches, _ := filepath.Glob(filepath.Join(outputDir, fmt.Sprintf("%s.*", videoID)))
	for _, m := range matches {
		ext := strings.ToLower(filepath.Ext(m))
		if ext == ".mp4" || ext == ".webm" || ext == ".mkv" {
			return m, nil
		}
	}

	return "", fmt.Errorf("downloaded file not found for video ID: %s", videoID)
}
