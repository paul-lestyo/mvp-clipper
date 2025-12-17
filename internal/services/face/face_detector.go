package face

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"log"
	"mvp-clipper/internal/utils"
	"os"
	"path/filepath"
	"strconv"
)

// ExtractFrames extracts frames from a video at the specified fps
func ExtractFrames(videoPath string, fps int) ([]string, error) {
	// Create temp directory for frames
	framesDir := filepath.Join("tmp", "frames", filepath.Base(videoPath))
	err := os.MkdirAll(framesDir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create frames directory: %w", err)
	}

	// FFmpeg command to extract frames
	outputPattern := filepath.Join(framesDir, "frame_%04d.jpg")
	cmd := []string{
		"ffmpeg",
		"-i", videoPath,
		"-vf", fmt.Sprintf("fps=%d", fps),
		"-q:v", "2", // JPEG quality
		outputPattern,
	}

	_, err = utils.Exec(cmd...)
	if err != nil {
		return nil, fmt.Errorf("failed to extract frames: %w", err)
	}

	// Get list of extracted frames
	files, err := filepath.Glob(filepath.Join(framesDir, "frame_*.jpg"))
	if err != nil {
		return nil, fmt.Errorf("failed to list frames: %w", err)
	}

	return files, nil
}

// DetectMode analyzes a frame and determines the cropping mode
func DetectMode(framePath string) (string, error) {
	faces, err := DetectFaces(framePath)
	if err != nil {
		log.Printf("[DEBUG] Face detection failed for %s: %v", framePath, err)
		return "", fmt.Errorf("failed to detect faces: %w", err)
	}

	log.Printf("[DEBUG] Detected %d faces in %s", len(faces), framePath)

	// Safety check: if we detect an unreasonable number of faces, it's likely false positives
	const maxFacesPerFrame = 20
	if len(faces) > maxFacesPerFrame {
		log.Printf("[WARNING] Detected %d faces (> %d), likely false positives. Using center mode.", len(faces), maxFacesPerFrame)
		return "center", nil
	}

	// Determine mode based on number and position of faces
	if len(faces) <= 1 {
		log.Printf("[DEBUG] Using center mode (faces <= 1)")
		return "center", nil
	}

	// Get actual frame dimensions to calculate midpoint
	file, err := os.Open(framePath)
	if err != nil {
		log.Printf("[DEBUG] Failed to open frame for dimension check: %v", err)
		return "center", nil // Default to center on error
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Printf("[DEBUG] Failed to decode frame: %v", err)
		return "center", nil // Default to center on error
	}

	// Calculate midpoint based on actual frame width
	frameWidth := float32(img.Bounds().Dx())
	midpoint := frameWidth / 2

	// Pigo returns coordinates in actual image space (no scaling needed)
	log.Printf("[DEBUG] Frame width: %.0f, midpoint: %.0f", frameWidth, midpoint)

	leftFaces := 0
	rightFaces := 0

	for i, face := range faces {
		// Calculate face center X position
		centerX := face.X + face.Width/2
		log.Printf("[DEBUG] Face %d: X=%.1f, centerX=%.1f, side=%s",
			i, face.X, centerX,
			map[bool]string{true: "left", false: "right"}[centerX < midpoint])
		if centerX < midpoint {
			leftFaces++
		} else {
			rightFaces++
		}
	}

	log.Printf("[DEBUG] Left faces: %d, Right faces: %d", leftFaces, rightFaces)

	// If we have faces on both sides, use split mode
	if leftFaces > 0 && rightFaces > 0 {
		log.Printf("[DEBUG] Using SPLIT mode (faces on both sides)")
		return "split", nil
	}

	// Otherwise, use center mode
	log.Printf("[DEBUG] Using center mode (faces on same side)")
	return "center", nil
}

// AnalyzeVideo extracts frames and analyzes face positions to create a timeline
func AnalyzeVideo(videoPath string) ([]TimelineEntry, error) {
	const fps = 1 // Extract 1 frame per second (1 fps)

	// Extract frames
	frames, err := ExtractFrames(videoPath, fps)
	if err != nil {
		return nil, fmt.Errorf("failed to extract frames: %w", err)
	}
	defer cleanupFrames(frames)

	var timeline []TimelineEntry

	// Analyze each frame
	log.Printf("[DEBUG] Analyzing %d frames at %d fps", len(frames), fps)
	for i, framePath := range frames {
		timestamp := float64(i) / float64(fps)
		log.Printf("[DEBUG] Frame %d at %.1fs: %s", i, timestamp, framePath)
		
		mode, err := DetectMode(framePath)
		if err != nil {
			// If detection fails, default to center mode
			log.Printf("[DEBUG] Detection failed, using center mode")
			mode = "center"
		}

		timeline = append(timeline, TimelineEntry{
			Timestamp: timestamp,
			Mode:      mode,
		})
		log.Printf("[DEBUG] Timeline entry: {%.1fs, %s}", timestamp, mode)
	}

	return timeline, nil
}

// cleanupFrames removes extracted frame files
func cleanupFrames(frames []string) {
	if len(frames) == 0 {
		return
	}

	// Remove the frames directory
	framesDir := filepath.Dir(frames[0])
	os.RemoveAll(framesDir)
}

// GetVideoDuration returns the duration of a video in seconds
func GetVideoDuration(videoPath string) (float64, error) {
	cmd := []string{
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	}

	output, err := utils.Exec(cmd...)
	if err != nil {
		return 0, fmt.Errorf("failed to get video duration: %w", err)
	}

	duration, err := strconv.ParseFloat(output[:len(output)-1], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return duration, nil
}
