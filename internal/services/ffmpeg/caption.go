package ffmpeg

import (
	"fmt"
	"log"
	"mvp-clipper/internal/utils"
	"strings"
)

// CaptionStyle untuk konfigurasi subtitle
type CaptionStyle struct {
	FontSize   int    // Ukuran font (default: 14)
	FontName   string // Nama font (default: Arial)
	Bold       int    // 1 = bold, 0 = normal
	MarginV    int    // Margin dari bawah dalam pixel (default: 50)
	Outline    int    // Ketebalan outline (default: 2)
	Shadow     int    // Shadow (default: 1)
}

// DefaultCaptionStyle returns TikTok-like caption style
func DefaultCaptionStyle() CaptionStyle {
	return CaptionStyle{
		FontSize: 12,
		FontName: "Roboto-Regular",
		Bold:     1,
		MarginV:  40, // Naik 50 pixel dari bawah
		Outline:  2,
		Shadow:   1,
	}
}

func BurnCaption(input, srt, output string) error {
	return BurnCaptionWithStyle(input, srt, output, DefaultCaptionStyle())
}

func BurnCaptionWithStyle(input, srt, output string, style CaptionStyle) error {
	// Escape path for ffmpeg subtitle filter
	escapedSrt := escapeSubtitlePath(srt)

	// Build force_style string
	// Format: 'Key=Value,Key=Value'
	forceStyle := fmt.Sprintf("Fontsize=%d,Fontname=%s,Bold=%d,MarginV=%d,Outline=%d,Shadow=%d,Alignment=2",
		style.FontSize,
		style.FontName,
		style.Bold,
		style.MarginV,
		style.Outline,
		style.Shadow,
	)

	// Build subtitle filter
	subtitleFilter := fmt.Sprintf("subtitles=%s:force_style='%s'", escapedSrt, forceStyle)

	cmd := []string{
		"ffmpeg",
		"-y", // overwrite output
		"-i", input,
		"-vf", subtitleFilter,
		"-c:v", "libx264",
		"-c:a", "aac",
		"-preset", "fast",
		output,
	}

	log.Println("Running:", cmd)
	_, err := utils.Exec(cmd...)
	if err != nil {
		return fmt.Errorf("burn caption failed: %v", err)
	}

	return nil
}

// escapeSubtitlePath escapes special characters for ffmpeg subtitle filter
func escapeSubtitlePath(path string) string {
	// Replace backslashes with forward slashes (works on all OS)
	path = strings.ReplaceAll(path, "\\", "/")
	// Escape colons (needed for Windows paths like C:)
	path = strings.ReplaceAll(path, ":", "\\:")
	// Escape single quotes
	path = strings.ReplaceAll(path, "'", "\\'")
	return path
}
