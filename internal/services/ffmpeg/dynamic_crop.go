package ffmpeg

import (
	"fmt"
	"log"
	"mvp-clipper/internal/services/face"
	"mvp-clipper/internal/utils"
	"os"
	"path/filepath"
	"strings"
)

// DynamicCrop applies dynamic cropping based on timeline of face positions
func DynamicCrop(input, output string, timeline []face.TimelineEntry) error {
	if len(timeline) == 0 {
		return fmt.Errorf("timeline is empty")
	}

	// Get video duration to handle the last segment
	duration, err := face.GetVideoDuration(input)
	if err != nil {
		return fmt.Errorf("failed to get video duration: %w", err)
	}

	// Build FFmpeg filter complex
	filterComplex := buildDynamicCropFilter(timeline, duration)
	
	log.Printf("[DEBUG] Filter complex (%d chars)", len(filterComplex))
	
	// Write filter to temp file to avoid command line length limits
	filterFile := "tmp/filter_complex.txt"
	err = os.WriteFile(filterFile, []byte(filterComplex), 0644)
	if err != nil {
		return fmt.Errorf("failed to write filter file: %w", err)
	}
	
	// Get absolute path for filter file
	absFilterFile, err := filepath.Abs(filterFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	
	log.Printf("[DEBUG] Filter file: %s", absFilterFile)
	
	cmd := []string{
		"ffmpeg",
		"-y", // overwrite output
		"-i", input,
		"-filter_complex_script", absFilterFile,
		"-map", "[out]",   // Map filtered video
		"-map", "0:a?",    // Map audio from input (? makes it optional)
		"-c:v", "libx264",
		"-c:a", "copy",    // Copy audio codec (faster, no quality loss)
		output,
	}

	log.Println("Running dynamic crop with filter file")
	ffmpegOutput, err := utils.Exec(cmd...)
	if err != nil {
		log.Printf("[ERROR] FFmpeg failed. Output: %s", ffmpegOutput)
		log.Printf("[ERROR] Command: %v", cmd)
		return fmt.Errorf("dynamic crop failed: %w", err)
	}

	return nil
}

// buildDynamicCropFilter creates FFmpeg filter for dynamic cropping
func buildDynamicCropFilter(timeline []face.TimelineEntry, videoDuration float64) string {
	var filters []string
	var outputs []string

	// Split input video into N streams (one for each timeline segment)
	numSegments := len(timeline)
	splitOutputs := make([]string, numSegments)
	for i := 0; i < numSegments; i++ {
		splitOutputs[i] = fmt.Sprintf("[v%d]", i)
	}
	filters = append(filters, fmt.Sprintf("[0:v]split=%d%s", numSegments, strings.Join(splitOutputs, "")))

	// Process each timeline segment
	for i, entry := range timeline {
		startTime := entry.Timestamp
		var endTime float64
		if i < len(timeline)-1 {
			endTime = timeline[i+1].Timestamp
		} else {
			endTime = videoDuration
		}

		inputLabel := fmt.Sprintf("[v%d]", i)
		outputLabel := fmt.Sprintf("[c%d]", i)

		if entry.Mode == "center" {
			// Center crop: 1080x1920 from center
			filter := fmt.Sprintf("%s"+
				"scale=-1:1920,"+
				"crop=1080:1920:(in_w-1080)/2:0,"+
				"trim=start=%.3f:end=%.3f,"+
				"setpts=PTS-STARTPTS,"+
				"setsar=1"+
				"%s",
				inputLabel, startTime, endTime, outputLabel)
			filters = append(filters, filter)
		} else {
			// Split mode: split into left/right, scale, and stack horizontally
			leftLabel := fmt.Sprintf("[left%d]", i)
			rightLabel := fmt.Sprintf("[right%d]", i)

			// Split the segment into two streams
			filter := fmt.Sprintf("%ssplit=2%s%s", inputLabel, leftLabel, rightLabel)
			filters = append(filters, filter)

			// Crop left half, scale to 540x1920
			filter = fmt.Sprintf("%s"+
				"scale=-1:1920,"+
				"crop=iw/2:1920:0:0,"+
				"scale=540:1920,"+
				"trim=start=%.3f:end=%.3f,"+
				"setpts=PTS-STARTPTS"+
				"[left_scaled%d]",
				leftLabel, startTime, endTime, i)
			filters = append(filters, filter)

			// Crop right half, scale to 540x1920
			filter = fmt.Sprintf("%s"+
				"scale=-1:1920,"+
				"crop=iw/2:1920:iw/2:0,"+
				"scale=540:1920,"+
				"trim=start=%.3f:end=%.3f,"+
				"setpts=PTS-STARTPTS"+
				"[right_scaled%d]",
				rightLabel, startTime, endTime, i)
			filters = append(filters, filter)

			// Stack horizontally and normalize SAR
			filter = fmt.Sprintf("[left_scaled%d][right_scaled%d]hstack=inputs=2,setsar=1%s",
				i, i, outputLabel)
			filters = append(filters, filter)
		}

		outputs = append(outputs, outputLabel)
	}

	// Concatenate all segments
	if len(outputs) > 1 {
		concatFilter := fmt.Sprintf("%sconcat=n=%d:v=1:a=0[out]",
			strings.Join(outputs, ""), len(outputs))
		filters = append(filters, concatFilter)
	} else {
		// If only one segment, use null filter to pass through
		filters = append(filters, fmt.Sprintf("%snull[out]", outputs[0]))
	}

	return strings.Join(filters, ";")
}
