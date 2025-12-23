package face

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"log"
	"os"
)

var (
	yunetClient *YuNetClient
)

// InitYuNet initializes the YuNet client
func InitYuNet(socketPath string) error {
	yunetClient = NewYuNetClient(socketPath)
	log.Printf("YuNet client initialized (socket: %s)", socketPath)
	return nil
}

// Cleanup releases resources
func Cleanup() {
	yunetClient = nil
	log.Println("YuNet client cleaned up")
}

// DetectFaces runs face detection on an image file using YuNet service
func DetectFaces(imagePath string) ([]FaceDetection, error) {
	if yunetClient == nil {
		return nil, fmt.Errorf("YuNet face detection not initialized")
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

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Convert image to RGB bytes
	rgbData := make([]byte, width*height*3)
	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			rgbData[idx] = uint8(r >> 8)
			rgbData[idx+1] = uint8(g >> 8)
			rgbData[idx+2] = uint8(b >> 8)
			idx += 3
		}
	}

	// Call YuNet service
	detections := yunetClient.DetectWithFallback(rgbData, width, height)

	log.Printf("Detected %d face(s) in %s", len(detections), imagePath)
	return detections, nil
}

