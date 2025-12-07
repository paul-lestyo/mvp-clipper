package ffmpeg

import (
	"fmt"
	"log"
	"mvp-clipper/internal/utils"
)

func Cut(input, output, start, end string) error {
	// Calculate duration from start and end
	startSec := utils.TimeToSeconds(start)
	endSec := utils.TimeToSeconds(end)
	duration := endSec - startSec
	
	if duration <= 0 {
		return fmt.Errorf("invalid time range: start=%s, end=%s", start, end)
	}

	// Transcode to H.264 + AAC for compatibility
	// YouTube videos often use AV1/VP9 + Opus which need re-encoding
	// Use -ss before -i for fast seek, then -t for duration
	cmd := []string{
		"ffmpeg",
		"-y",                              // overwrite output
		"-ss", start,                      // seek to start
		"-i", input,
		"-t", fmt.Sprintf("%.3f", duration), // duration in seconds
		"-c:v", "libx264",                 // transcode video to H.264
		"-c:a", "aac",                     // transcode audio to AAC
		"-preset", "fast",                 // faster encoding
		"-crf", "23",                      // quality (lower = better, 18-28 is good)
		output,
	}

	log.Println("Running:", cmd)
	_, err := utils.Exec(cmd...)
	if err != nil {
		return fmt.Errorf("cut failed: %v", err)
	}

	return nil
}
