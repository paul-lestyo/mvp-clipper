package handlers

import (
	"fmt"
	"mvp-clipper/internal/services/ai"
	"mvp-clipper/internal/services/face"
	"mvp-clipper/internal/services/ffmpeg"
	"mvp-clipper/internal/services/yt"
	"mvp-clipper/internal/utils"
	"os"

	"github.com/gofiber/fiber/v2"
)

func RegisterClipRoutes(app *fiber.App) {
	app.Post("/clip/analyze", analyzeClip)
	app.Post("/clip/generate", generateClip)
}

func analyzeClip(c *fiber.Ctx) error {
	var payload struct {
		Url string `json:"url"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	// 1. Get video info (title + channel)
	videoInfo, err := yt.GetVideoInfo(payload.Url)
	if err != nil {
		// Non-fatal, continue with empty info
		videoInfo = yt.VideoInfo{}
	}

	// 2. Download transcript
	srtPath, err := yt.DownloadTranscript(payload.Url)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// 3. Read transcript file
	srtContent, err := os.ReadFile(srtPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to read transcript: " + err.Error()})
	}

	// 4. Call AI analyzer with video info for context
	result, err := ai.AnalyzeTranscript(string(srtContent), videoInfo.Title, videoInfo.Channel)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "AI analysis failed: " + err.Error()})
	}

	return c.JSON(fiber.Map{
		"title":      videoInfo.Title,
		"channel":    videoInfo.Channel,
		"transcript": srtPath,
		"analysis":   result,
	})
}

func generateClip(c *fiber.Ctx) error {
	var payload struct {
		URL       string `json:"url"`
		Start     string `json:"start"`
		End       string `json:"end"`
		Portrait  bool   `json:"portrait"`
		Caption   bool   `json:"caption"`
		Split     bool   `json:"split"`
		SmartCrop bool   `json:"smartCrop"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	// Extract video ID for file paths
	videoID := yt.ExtractVideoID(payload.URL)
	if videoID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "invalid YouTube URL"})
	}

	// Ensure directories exist
	os.MkdirAll("tmp/clips", os.ModePerm)
	os.MkdirAll("tmp/downloads", os.ModePerm)

	// 1. Download video (skip if already exists)
	videoPath := fmt.Sprintf("tmp/downloads/%s.mp4", videoID)
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		// File doesn't exist, download it
		videoPath, err = yt.DownloadVideo(payload.URL)
		if err != nil {
			return errJson(c, err)
		}
	}

	// 2. Cut
	cutPath := fmt.Sprintf("tmp/clips/%s_cut.mp4", videoID)
	if err := ffmpeg.Cut(videoPath, cutPath, payload.Start, payload.End); err != nil {
		return errJson(c, err)
	}

	// 3. Smart Crop (if enabled)
	finalPath := cutPath
	if payload.SmartCrop {
		// Analyze video for face positions
		timeline, err := face.AnalyzeVideo(cutPath)
		if err != nil {
			return errJson(c, err)
		}

		// Compress timeline to remove redundant entries
		compressed := face.CompressTimeline(timeline)
		fmt.Println(compressed)
		// Apply dynamic cropping
		smartPath := fmt.Sprintf("tmp/clips/%s_smart.mp4", videoID)
		if err := ffmpeg.DynamicCrop(cutPath, smartPath, compressed); err != nil {
			return errJson(c, err)
		}
		finalPath = smartPath
	} else if payload.Portrait {
		// 4. Portrait (manual mode)
		finalPath = fmt.Sprintf("tmp/clips/%s_portrait.mp4", videoID)
		if err := ffmpeg.ToPortrait(cutPath, finalPath); err != nil {
			return errJson(c, err)
		}
	}

	// 5. Split (2-speaker) - only if not using smart crop
	if payload.Split && !payload.SmartCrop {
		splitPath := fmt.Sprintf("tmp/clips/%s_split.mp4", videoID)
		if err := ffmpeg.SplitTwoSpeakers(finalPath, splitPath); err != nil {
			return errJson(c, err)
		}
		finalPath = splitPath
	}

	// 5. Burn Caption
	if payload.Caption {
		// Find the SRT file for this video
		srtPath, err := findSubtitleFile(videoID)
		if err != nil {
			// Subtitle not found, try to download it
			srtPath, err = yt.DownloadTranscript(payload.URL)
			if err != nil {
				return errJson(c, fmt.Errorf("failed to download subtitle: %w", err))
			}
		}

		// Cut SRT to match clip duration (adjust timestamps)
		cutSrtPath := fmt.Sprintf("tmp/clips/%s_cut.srt", videoID)
		if err := utils.CutSRT(srtPath, cutSrtPath, payload.Start, payload.End); err != nil {
			return errJson(c, err)
		}

		captionPath := fmt.Sprintf("tmp/clips/%s_caption.mp4", videoID)
		if err := ffmpeg.BurnCaption(finalPath, cutSrtPath, captionPath); err != nil {
			return errJson(c, err)
		}
		finalPath = captionPath
	}

	return c.JSON(fiber.Map{
		"clip": finalPath,
	})
}

// findSubtitleFile finds the subtitle file for a given video ID
func findSubtitleFile(videoID string) (string, error) {
	// Try common subtitle extensions
	extensions := []string{".id.srt", ".id.vtt", ".en.srt", ".en.vtt", ".srt", ".vtt"}
	for _, ext := range extensions {
		path := fmt.Sprintf("tmp/downloads/%s%s", videoID, ext)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("subtitle file not found for video ID: %s", videoID)
}

func errJson(c *fiber.Ctx, err error) error {
	return c.Status(500).JSON(fiber.Map{"error": err.Error()})
}
