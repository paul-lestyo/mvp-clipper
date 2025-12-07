package ffmpeg

import (
	"fmt"
	"log"
	"mvp-clipper/internal/utils"
)

// PortraitOptions untuk konfigurasi crop
type PortraitOptions struct {
	Width      int    // Output width (default: 1080)
	Height     int    // Output height (default: 1920)
	CropX      string // Custom X position for crop (default: auto-center)
	CropY      string // Custom Y position for crop (default: "0")
}

// DefaultPortraitOptions returns default 9:16 settings
func DefaultPortraitOptions() PortraitOptions {
	return PortraitOptions{
		Width:  1080,
		Height: 1920,
		CropX:  "", // empty = auto-center
		CropY:  "0",
	}
}

// ToPortrait converts video to 9:16 portrait with auto-center crop
func ToPortrait(input, output string) error {
	return ToPortraitWithOptions(input, output, DefaultPortraitOptions())
}

// ToPortraitWithOptions converts video to portrait with custom options
func ToPortraitWithOptions(input, output string, opts PortraitOptions) error {
	// Build crop position
	cropX := opts.CropX
	if cropX == "" {
		// Auto-center: (in_w - width) / 2
		cropX = fmt.Sprintf("(in_w-%d)/2", opts.Width)
	}
	cropY := opts.CropY
	if cropY == "" {
		cropY = "0"
	}

	// Build filter:
	// 1. scale=-1:height → scale height first, width auto
	// 2. crop=width:height:x:y → crop to exact size
	vf := fmt.Sprintf("scale=-1:%d,crop=%d:%d:%s:%s",
		opts.Height,
		opts.Width,
		opts.Height,
		cropX,
		cropY,
	)

	cmd := []string{
		"ffmpeg",
		"-y", // overwrite output
		"-i", input,
		"-vf", vf,
		"-c:v", "libx264",
		"-c:a", "aac",
		output,
	}

	log.Println("Running:", cmd)
	_, err := utils.Exec(cmd...)
	if err != nil {
		return fmt.Errorf("portrait failed: %v", err)
	}

	return nil
}

// ToPortraitLeft crops focusing on left side of video
func ToPortraitLeft(input, output string) error {
	opts := DefaultPortraitOptions()
	opts.CropX = "0" // Start from left
	return ToPortraitWithOptions(input, output, opts)
}

// ToPortraitRight crops focusing on right side of video
func ToPortraitRight(input, output string) error {
	opts := DefaultPortraitOptions()
	opts.CropX = fmt.Sprintf("in_w-%d", opts.Width) // Start from right
	return ToPortraitWithOptions(input, output, opts)
}

// ToPortraitCustom crops with custom X position (0.0 = left, 0.5 = center, 1.0 = right)
func ToPortraitCustom(input, output string, positionX float64) error {
	opts := DefaultPortraitOptions()
	// Calculate X position: (in_w - width) * positionX
	opts.CropX = fmt.Sprintf("(in_w-%d)*%.2f", opts.Width, positionX)
	return ToPortraitWithOptions(input, output, opts)
}
