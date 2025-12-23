package face

import (
	"fmt"
	"log"
	"mvp-clipper/internal/utils"
	"path/filepath"
	"strconv"
	"strings"
)

// GetVideoMetadata returns width, height, and fps of the video
func GetVideoMetadata(videoPath string) (int, int, float64, error) {
	cmd := []string{
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,r_frame_rate",
		"-of", "csv=s=x:p=0",
		videoPath,
	}

	output, err := utils.Exec(cmd...)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	// Output format: widthxheightxfps_num/fps_den
	// e.g., 1920x1080x30/1
	var width, height, num, den int
	output = strings.TrimSpace(output)
	lines := strings.Split(output, "\n")
	if len(lines) > 0 {
		output = lines[0]
	}

	_, err = fmt.Sscanf(output, "%dx%dx%d/%d", &width, &height, &num, &den)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse ffprobe output '%s': %w", output, err)
	}

	if den == 0 {
		return 0, 0, 0, fmt.Errorf("invalid fps denominator 0")
	}

	fps := float64(num) / float64(den)
	return width, height, fps, nil
}

// AnalyzeVideo analyzes face positions using the Python service
func AnalyzeVideo(videoPath string) ([]TimelineEntry, error) {
	// 1. Get Video Metadata
	width, height, fps, err := GetVideoMetadata(videoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get video metadata: %w", err)
	}

	log.Printf("[INFO] Video Metadata: %dx%d @ %.2f fps", width, height, fps)

	// 2. Call Python Service
	client := NewPythonClient()
	filename := filepath.Base(videoPath)
	
	log.Printf("[INFO] Requesting analysis for %s", filename)
	results, err := client.ProcessVideo(filename)
	if err != nil {
		return nil, fmt.Errorf("python analysis failed: %w", err)
	}

	if len(results) == 0 {
		log.Printf("[WARNING] No analysis results returned")
		return nil, nil
	}

	log.Printf("[DEBUG] Python service returned %d samples.", len(results))
	if len(results) > 0 {
		log.Printf("[DEBUG] Sample[0]: %+v", results[0])
	}

	// 3. Convert to Timeline
	var timeline []TimelineEntry
	for i, res := range results {
		timestamp := float64(res.Frame) / fps
		
		var centers []Point
		for _, cx := range res.Centers {
			centers = append(centers, Point{
				X: float32(cx),
				Y: float32(height / 2),
			})
		}
		
		var primaryCenter *Point
		if len(centers) > 0 {
			primaryCenter = &centers[0]
		}
		
		mode := res.Mode
		if mode == "" {
			mode = "center"
		}

		// Mock BBox for compatibility
		var bbox *BoundingBox
		if primaryCenter != nil {
			boxSize := float32(200)
			bbox = &BoundingBox{
				X:      primaryCenter.X - boxSize/2,
				Y:      primaryCenter.Y - boxSize/2,
				Width:  boxSize,
				Height: boxSize,
			}
		}

		entry := TimelineEntry{
			Timestamp:  timestamp,
			Mode:       mode,
			BBox:       bbox,
			Center:     primaryCenter,
			Centers:    centers,
			Confidence: 1.0,
			Status:     StatusDetected,
		}
		timeline = append(timeline, entry)

		// Log occasional samples
		if i < 3 || i%50 == 0 || i == len(results)-1 {
			log.Printf("[TRACE] Timeline Entry #%d: T=%.2fs Mode=%s Centers=%v", i, timestamp, mode, centers)
		}
	}

	log.Printf("[INFO] Generated timeline with %d entries", len(timeline))
	return timeline, nil
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

	duration, err := strconv.ParseFloat(strings.TrimSpace(output), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return duration, nil
}
