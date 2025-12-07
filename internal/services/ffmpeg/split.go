package ffmpeg

import (
	"fmt"
	"log"
	"mvp-clipper/internal/utils"
)

// Split 2 speaker → portrait stacked horizontally
func SplitTwoSpeakers(input, output string) error {
	cmd := []string{
		"ffmpeg",
		"-i", input,
		"-filter_complex",
		"[0:v]crop=iw/2:ih:0:0[left];" +
			"[0:v]crop=iw/2:ih:iw/2:0[right];" +
			"[left]scale=540:1920[left2];" +
			"[right]scale=540:1920[right2];" +
			"[left2][right2]hstack=inputs=2",
		"-c:v", "libx264",
		output,
	}

	log.Println(cmd)
	_, err := utils.Exec(cmd...)
	if err != nil {
		return fmt.Errorf("split failed: %v", err)
	}

	return nil
}
