package face

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/jpeg"
	"log"
	"os"
	"path/filepath"
)

// AnnotateFrameWithLandmarks draws bounding boxes and landmarks on detected faces
func AnnotateFrameWithLandmarks(framePath string, rawDetections, filteredDetections []FaceDetection, primaryFace *FaceDetection, mode string) error {
	// Load the original image
	file, err := os.Open(framePath)
	if err != nil {
		return fmt.Errorf("failed to open frame: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode frame: %w", err)
	}

	// Create a new RGBA image for drawing
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	// Draw raw detections in red (rejected by filters)
	for _, face := range rawDetections {
		if !containsFace(filteredDetections, face) {
			drawRect(rgba, int(face.X), int(face.Y), int(face.X+face.Width), int(face.Y+face.Height),
				color.RGBA{255, 0, 0, 255}, 2) // Red for rejected
		}
	}

	// Draw filtered detections in yellow (not selected)
	for _, face := range filteredDetections {
		if primaryFace == nil || !facesEqual(face, *primaryFace) {
			drawRect(rgba, int(face.X), int(face.Y), int(face.X+face.Width), int(face.Y+face.Height),
				color.RGBA{255, 255, 0, 255}, 2) // Yellow for filtered but not selected
			drawLandmarks(rgba, face, color.RGBA{255, 255, 0, 255})
		}
	}

	// Draw primary face in green
	if primaryFace != nil {
		drawRect(rgba, int(primaryFace.X), int(primaryFace.Y), int(primaryFace.X+primaryFace.Width), int(primaryFace.Y+primaryFace.Height),
			color.RGBA{0, 255, 0, 255}, 3) // Green for selected
		drawLandmarks(rgba, *primaryFace, color.RGBA{0, 255, 0, 255})
	}

	// Add mode text overlay
	drawModeLabel(rgba, mode, len(rawDetections), len(filteredDetections), primaryFace != nil)

	// Create annotated directory
	framesDir := filepath.Dir(framePath)
	annotatedDir := filepath.Join(framesDir, "annotated")
	err = os.MkdirAll(annotatedDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create annotated directory: %w", err)
	}

	// Save annotated image
	annotatedPath := filepath.Join(annotatedDir, filepath.Base(framePath))
	outFile, err := os.Create(annotatedPath)
	if err != nil {
		return fmt.Errorf("failed to create annotated file: %w", err)
	}
	defer outFile.Close()

	err = jpeg.Encode(outFile, rgba, &jpeg.Options{Quality: 95})
	if err != nil {
		return fmt.Errorf("failed to encode annotated image: %w", err)
	}

	log.Printf("[DEBUG] Saved annotated frame to: %s", annotatedPath)
	return nil
}

// drawRect draws a rectangle with specified thickness
func drawRect(img *image.RGBA, x1, y1, x2, y2 int, col color.RGBA, thickness int) {
	// Draw top and bottom horizontal lines
	for t := 0; t < thickness; t++ {
		for x := x1; x <= x2; x++ {
			if y1+t >= 0 && y1+t < img.Bounds().Dy() && x >= 0 && x < img.Bounds().Dx() {
				img.Set(x, y1+t, col)
			}
			if y2-t >= 0 && y2-t < img.Bounds().Dy() && x >= 0 && x < img.Bounds().Dx() {
				img.Set(x, y2-t, col)
			}
		}
	}

	// Draw left and right vertical lines
	for t := 0; t < thickness; t++ {
		for y := y1; y <= y2; y++ {
			if x1+t >= 0 && x1+t < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
				img.Set(x1+t, y, col)
			}
			if x2-t >= 0 && x2-t < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
				img.Set(x2-t, y, col)
			}
		}
	}
}

// drawLandmarks draws facial landmarks (eyes, nose, mouth)
func drawLandmarks(img *image.RGBA, face FaceDetection, col color.RGBA) {
	radius := 3
	for _, landmark := range face.Landmarks {
		x := int(landmark.X)
		y := int(landmark.Y)

		// Draw small circle for each landmark
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if dx*dx+dy*dy <= radius*radius {
					px := x + dx
					py := y + dy
					if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
						img.Set(px, py, col)
					}
				}
			}
		}
	}
}

// drawModeLabel draws the mode and face count as text overlay
func drawModeLabel(img *image.RGBA, mode string, rawCount, filteredCount int, hasSelected bool) {
	// Simple text rendering - draw a colored bar at the top with mode info
	labelColor := color.RGBA{0, 0, 0, 200} // Semi-transparent black
	if mode == "split" {
		labelColor = color.RGBA{255, 165, 0, 200} // Orange for split mode
	}

	// Draw label bar at top (50 pixels high)
	for y := 0; y < 50; y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.Set(x, y, labelColor)
		}
	}

	// Note: For actual text rendering, you'd need to import a font library
	// For now, the colored bar indicates the mode (black=center, orange=split)
	log.Printf("[DEBUG] Mode label: %s (raw=%d, filtered=%d, selected=%v)", mode, rawCount, filteredCount, hasSelected)
}

// Helper functions
func containsFace(faces []FaceDetection, target FaceDetection) bool {
	for _, face := range faces {
		if facesEqual(face, target) {
			return true
		}
	}
	return false
}

func facesEqual(a, b FaceDetection) bool {
	return a.X == b.X && a.Y == b.Y && a.Width == b.Width && a.Height == b.Height
}
