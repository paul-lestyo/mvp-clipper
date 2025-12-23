package ffmpeg

import (
	"fmt"
	"log"
	"math"
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

	// Get video metadata to handle scaling and cropping correctly
	width, height, _, err := face.GetVideoMetadata(input)
	if err != nil {
		return fmt.Errorf("failed to get video metadata: %w", err)
	}

	// Get video duration to handle the last segment
	duration, err := face.GetVideoDuration(input)
	if err != nil {
		return fmt.Errorf("failed to get video duration: %w", err)
	}

	// Build FFmpeg filter complex
	filterComplex := buildDynamicCropFilter(timeline, duration, width, height)
	
	log.Printf("[DEBUG] Filter complex (%d chars)", len(filterComplex))
	
	// Write filter to temp file
	filterFile := "tmp/filter_complex.txt"
	err = os.WriteFile(filterFile, []byte(filterComplex), 0644)
	if err != nil {
		return fmt.Errorf("failed to write filter file: %w", err)
	}
	
	absFilterFile, err := filepath.Abs(filterFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	
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
		return fmt.Errorf("dynamic crop failed: %w", err)
	}

	return nil
}

// buildDynamicCropFilter creates FFmpeg filter for dynamic cropping
func buildDynamicCropFilter(timeline []face.TimelineEntry, videoDuration float64, origW, origH int) string {
	var filters []string
	var outputs []string

	// Calculate scaled dimensions (we scale height to 1920)
	scaledH := 1920
	ratio := float64(scaledH) / float64(origH)
	scaledW := int(float64(origW) * ratio)

	// 1. Group timeline into segments of same Mode
	type Segment struct {
		Mode    string
		Start   float64
		End     float64
		Entries []face.TimelineEntry
	}

	var segments []Segment
	if len(timeline) > 0 {
		currentSeg := Segment{
			Mode:    timeline[0].Mode,
			Start:   timeline[0].Timestamp,
			Entries: []face.TimelineEntry{timeline[0]},
		}

		for i := 1; i < len(timeline); i++ {
			entry := timeline[i]
			// Limit entries per segment to prevent extremely long FFmpeg expressions
			if entry.Mode == currentSeg.Mode && len(currentSeg.Entries) < 40 {
				currentSeg.Entries = append(currentSeg.Entries, entry)
			} else {
				currentSeg.End = entry.Timestamp
				segments = append(segments, currentSeg)
				currentSeg = Segment{
					Mode:    entry.Mode,
					Start:   entry.Timestamp,
					Entries: []face.TimelineEntry{entry},
				}
			}
		}
		currentSeg.End = videoDuration
		segments = append(segments, currentSeg)
	}

	// 2. Build Filter Graph
	numSegments := len(segments)
	splitOutputs := make([]string, numSegments)
	for i := 0; i < numSegments; i++ {
		splitOutputs[i] = fmt.Sprintf("[v%d]", i)
	}
	
	if numSegments > 1 {
		filters = append(filters, fmt.Sprintf("[0:v]split=%d%s", numSegments, strings.Join(splitOutputs, "")))
	} else {
		filters = append(filters, fmt.Sprintf("[0:v]copy[v0]"))
	}

	// Helper to generate X expression
	generateXExpr := func(seg Segment, width int, centerIdx int) string {
		defaultX := float64(width-1080) / 2.0
		var parts []string
		
		if len(seg.Entries) == 1 {
			targetX := defaultX
			if len(seg.Entries[0].Centers) > centerIdx {
				cx := float64(seg.Entries[0].Centers[centerIdx].X) * ratio
				tx := cx - 1080.0/2.0
				targetX = math.Max(0, math.Min(float64(width-1080), tx))
			} else if seg.Entries[0].Center != nil && centerIdx == 0 {
				// Fallback to old Center field
				cx := float64(seg.Entries[0].Center.X) * ratio
				tx := cx - 1080.0/2.0
				targetX = math.Max(0, math.Min(float64(width-1080), tx))
			}
			return fmt.Sprintf("%.2f", targetX)
		}

		for j, entry := range seg.Entries {
			tStart := entry.Timestamp
			tEnd := seg.End
			if j < len(seg.Entries)-1 {
				tEnd = seg.Entries[j+1].Timestamp
			}
			effectiveEnd := tEnd
			if j < len(seg.Entries)-1 {
				effectiveEnd -= 0.001
			}

			targetX := defaultX
			if len(entry.Centers) > centerIdx {
				cx := float64(entry.Centers[centerIdx].X) * ratio
				tx := cx - 1080.0/2.0
				targetX = math.Max(0, math.Min(float64(width-1080), tx))
			} else if entry.Center != nil && centerIdx == 0 {
				cx := float64(entry.Center.X) * ratio
				tx := cx - 1080.0/2.0
				targetX = math.Max(0, math.Min(float64(width-1080), tx))
			}
			parts = append(parts, fmt.Sprintf("between(t,%.3f,%.3f)*%.2f", tStart, effectiveEnd, targetX))
		}
		
		if len(parts) == 0 {
			return fmt.Sprintf("%.2f", defaultX)
		}
		return strings.Join(parts, "+")
	}

	for i, seg := range segments {
		inputLabel := fmt.Sprintf("[v%d]", i)
		outputLabel := fmt.Sprintf("[c%d]", i)

		if seg.Mode == "split" {
			// SPLIT MODE: Two Dynamic Crops stacked
			leftLabel := fmt.Sprintf("[left%d]", i)
			rightLabel := fmt.Sprintf("[right%d]", i)

			// Split input for this segment
			filters = append(filters, fmt.Sprintf("%ssplit=2%s%s", inputLabel, leftLabel, rightLabel))

			// Left/Top Panel (Index 0)
			xExpr0 := generateXExpr(seg, scaledW, 0)
			// Crop 1080x960 centered vertically (y=480)
			filterLeft := fmt.Sprintf("%s"+
				"scale=-1:1920,"+
				"crop=w=1080:h=960:x='%s':y=480,"+
				"trim=start=%.3f:end=%.3f,"+
				"setpts=PTS-STARTPTS"+
				"[left_scaled%d]",
				leftLabel, xExpr0, seg.Start, seg.End, i)
			filters = append(filters, filterLeft)

			// Right/Bottom Panel (Index 1)
			xExpr1 := generateXExpr(seg, scaledW, 1)
			filterRight := fmt.Sprintf("%s"+
				"scale=-1:1920,"+
				"crop=w=1080:h=960:x='%s':y=480,"+
				"trim=start=%.3f:end=%.3f,"+
				"setpts=PTS-STARTPTS"+
				"[right_scaled%d]",
				rightLabel, xExpr1, seg.Start, seg.End, i)
			filters = append(filters, filterRight)

			// Stack
			filters = append(filters, fmt.Sprintf("[left_scaled%d][right_scaled%d]vstack=inputs=2,setsar=1%s",
				i, i, outputLabel))

		} else {
			// CENTER MODE (Default)
			xExpr := generateXExpr(seg, scaledW, 0)
			
			filter := fmt.Sprintf("%s"+
				"scale=-1:1920,"+
				"crop=w=1080:h=1920:x='%s':y=0,"+
				"trim=start=%.3f:end=%.3f,"+
				"setpts=PTS-STARTPTS,"+
				"setsar=1"+
				"%s",
				inputLabel, xExpr, seg.Start, seg.End, outputLabel)
			filters = append(filters, filter)
		}

		outputs = append(outputs, outputLabel)
	}

	if len(outputs) > 1 {
		filters = append(filters, fmt.Sprintf("%sconcat=n=%d:v=1:a=0[out]",
			strings.Join(outputs, ""), len(outputs)))
	} else {
		filters = append(filters, fmt.Sprintf("%snull[out]", outputs[0]))
	}

	return strings.Join(filters, ";")
}
