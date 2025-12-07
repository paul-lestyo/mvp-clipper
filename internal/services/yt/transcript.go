package yt

import (
	"fmt"
	"log"
	"mvp-clipper/internal/utils"
	"os"
	"path/filepath"
	"strings"
)

// ExtractVideoID extracts the video ID from a YouTube URL
func ExtractVideoID(url string) string {
	// Handle different YouTube URL formats
	// https://www.youtube.com/watch?v=VIDEO_ID
	// https://youtu.be/VIDEO_ID
	if strings.Contains(url, "watch?v=") {
		parts := strings.Split(url, "watch?v=")
		if len(parts) > 1 {
			// Remove any additional params after &
			id := strings.Split(parts[1], "&")[0]
			return id
		}
	}
	if strings.Contains(url, "youtu.be/") {
		parts := strings.Split(url, "youtu.be/")
		if len(parts) > 1 {
			// Remove any additional params after ?
			id := strings.Split(parts[1], "?")[0]
			return id
		}
	}
	return ""
}

func DownloadTranscript(url string) (string, error) {
	// Extract video ID
	videoID := ExtractVideoID(url)
	if videoID == "" {
		return "", fmt.Errorf("invalid YouTube URL: cannot extract video ID")
	}

	// Ensure directory exists
	utils.EnsureDir("tmp/downloads")

	// Use fixed output name based on video ID
	outputTemplate := fmt.Sprintf("tmp/downloads/%s", videoID)
	outputSrt := fmt.Sprintf("tmp/downloads/%s.id.srt", videoID)
	outputVtt := fmt.Sprintf("tmp/downloads/%s.id.vtt", videoID)

	cmd := []string{
		"yt-dlp",
		"--write-auto-sub",
		"--sub-lang", "id",
		"--sub-format", "srt",
		"--skip-download",
		"-o", outputTemplate,
		url,
	}

	log.Println("Running:", cmd)

	_, err := utils.Exec(cmd...)
	if err != nil {
		return "", fmt.Errorf("yt-dlp failed: %v", err)
	}

	// Check which file was created
	if _, err := os.Stat(outputSrt); err == nil {
		return outputSrt, nil
	}

	// If VTT exists, convert to SRT (or just use VTT)
	if _, err := os.Stat(outputVtt); err == nil {
		return outputVtt, nil
	}

	// Try to find any subtitle file with this video ID
	matches, _ := filepath.Glob(fmt.Sprintf("tmp/downloads/%s*", videoID))
	for _, m := range matches {
		if strings.HasSuffix(m, ".srt") || strings.HasSuffix(m, ".vtt") {
			return m, nil
		}
	}

	return "", fmt.Errorf("subtitle file not found for video ID: %s", videoID)
}

// GetVideoTitle returns the title of a YouTube video
func GetVideoTitle(url string) (string, error) {
	cmd := []string{
		"yt-dlp",
		"--get-title",
		"--no-warnings",
		url,
	}

	output, err := utils.Exec(cmd...)
	if err != nil {
		return "", fmt.Errorf("failed to get video title: %v", err)
	}

	return strings.TrimSpace(output), nil
}

// GetChannelName returns the channel name of a YouTube video
func GetChannelName(url string) (string, error) {
	cmd := []string{
		"yt-dlp",
		"--print", "channel",
		"--no-warnings",
		url,
	}

	output, err := utils.Exec(cmd...)
	if err != nil {
		return "", fmt.Errorf("failed to get channel name: %v", err)
	}

	return strings.TrimSpace(output), nil
}

// VideoInfo contains metadata about a YouTube video
type VideoInfo struct {
	Title   string
	Channel string
}

// GetVideoInfo returns title and channel of a YouTube video in one call
func GetVideoInfo(url string) (VideoInfo, error) {
	cmd := []string{
		"yt-dlp",
		"--print", "%(title)s|||%(channel)s",
		"--no-warnings",
		url,
	}

	output, err := utils.Exec(cmd...)
	if err != nil {
		return VideoInfo{}, fmt.Errorf("failed to get video info: %v", err)
	}

	parts := strings.Split(strings.TrimSpace(output), "|||")
	if len(parts) != 2 {
		return VideoInfo{}, fmt.Errorf("unexpected output format")
	}

	return VideoInfo{
		Title:   parts[0],
		Channel: parts[1],
	}, nil
}
