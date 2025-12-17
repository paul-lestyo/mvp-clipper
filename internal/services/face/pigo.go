package face

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"io/ioutil"
	"log"
	"os"

	pigo "github.com/esimov/pigo/core"
)

var (
	classifier  *pigo.Pigo
	cascadeFile []byte
)

const (
	// Pigo detection parameters
	minSize      = 20   // Minimum face size (pixels)
	maxSize      = 1000 // Maximum face size (pixels)
	shiftFactor  = 0.1  // Shift factor for detection window
	scaleFactor  = 1.1  // Scale factor for image pyramid
	iouThreshold = 0.2  // IoU threshold for NMS
	qualityThreshold = 5.0 // Minimum quality score
)

// InitPigo initializes the Pigo face detector
func InitPigo(cascadePath string) error {
	var err error
	cascadeFile, err = ioutil.ReadFile(cascadePath)
	if err != nil {
		return fmt.Errorf("failed to read cascade file: %w", err)
	}

	p := pigo.NewPigo()
	classifier, err = p.Unpack(cascadeFile)
	if err != nil {
		return fmt.Errorf("failed to unpack cascade: %w", err)
	}

	log.Printf("Pigo face detector initialized successfully (minSize: %d, qualityThreshold: %.1f)", minSize, qualityThreshold)
	return nil
}

// Cleanup releases Pigo resources
func Cleanup() {
	classifier = nil
	cascadeFile = nil
	log.Println("Pigo detector resources cleaned up")
}

// DetectFaces runs face detection on an image file using Pigo
func DetectFaces(imagePath string) ([]FaceDetection, error) {
	if classifier == nil {
		return nil, fmt.Errorf("Pigo face detection not initialized")
	}

	// Load and decode image
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Convert to grayscale (Pigo requirement)
	gray := toGrayscale(img)

	// Run detection
	cParams := pigo.CascadeParams{
		MinSize:     minSize,
		MaxSize:     maxSize,
		ShiftFactor: shiftFactor,
		ScaleFactor: scaleFactor,
		ImageParams: pigo.ImageParams{
			Pixels: gray,
			Rows:   img.Bounds().Dy(),
			Cols:   img.Bounds().Dx(),
			Dim:    img.Bounds().Dx(),
		},
	}

	// Run cascade detection (0.0 = detect all, filter by quality later)
	dets := classifier.RunCascade(cParams, 0.0)
	
	// Cluster detections to remove duplicates
	dets = classifier.ClusterDetections(dets, iouThreshold)

	// Convert to FaceDetection format
	detections := convertPigoDetections(dets)

	log.Printf("Detected %d face(s) in %s", len(detections), imagePath)
	return detections, nil
}

// toGrayscale converts image to grayscale pixel array
func toGrayscale(img image.Image) []uint8 {
	bounds := img.Bounds()
	gray := make([]uint8, bounds.Dx()*bounds.Dy())

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// Standard grayscale conversion formula
			gray[y*bounds.Dx()+x] = uint8((r*299 + g*587 + b*114) / 1000 >> 8)
		}
	}

	return gray
}

// convertPigoDetections converts Pigo detections to FaceDetection format
func convertPigoDetections(dets []pigo.Detection) []FaceDetection {
	var detections []FaceDetection

	for _, det := range dets {
		// Filter by quality threshold
		if det.Q < qualityThreshold {
			continue
		}

		// Pigo returns center (Row, Col) and Scale (radius)
		// Convert to bounding box (X, Y, Width, Height)
		size := float32(det.Scale * 2) // Diameter
		x := float32(det.Col) - float32(det.Scale)
		y := float32(det.Row) - float32(det.Scale)

		detection := FaceDetection{
			X:          x,
			Y:          y,
			Width:      size,
			Height:     size,
			Confidence: float32(det.Q) / 100.0, // Normalize quality to 0-1 range
		}

		detections = append(detections, detection)
	}

	return detections
}
