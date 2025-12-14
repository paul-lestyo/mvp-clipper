package face

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"os"
	"sort"

	ort "github.com/yalue/onnxruntime_go"
)

var (
	yunetSession   *ort.AdvancedSession
	inputTensor    *ort.Tensor[float32]
	clsTensor      *ort.Tensor[float32]  // cls_8 output
	bboxTensor     *ort.Tensor[float32]  // bbox_8 output
	anchors        []Anchor               // Pre-generated anchors
)

const (
	inputWidth          = 640
	inputHeight         = 640
	confidenceThreshold = 0.7  // Balanced threshold: not too strict, not too loose
	iouThreshold        = 0.7  // Increased from 0.3 for better NMS
	stride              = 8
	gridSize            = 80 // 640 / 8
)

// Anchor represents a detection anchor point
type Anchor struct {
	CX float32 // Center X
	CY float32 // Center Y
}

// InitYuNet initializes the YuNet ONNX model with proper multi-output handling
func InitYuNet(modelPath string) error {
	// Set ONNX Runtime library path
	libraryPath := "libonnxruntime.so"
	if os.PathSeparator == '\\' {
		libraryPath = "onnxruntime.dll"
	}
	ort.SetSharedLibraryPath(libraryPath)

	err := ort.InitializeEnvironment()
	if err != nil {
		return fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}

	// Create input tensor: 1x3x640x640 (NCHW format)
	inputShape := ort.NewShape(1, 3, inputHeight, inputWidth)
	inputData := make([]float32, 1*3*inputHeight*inputWidth)
	inputTensor, err = ort.NewTensor(inputShape, inputData)
	if err != nil {
		return fmt.Errorf("failed to create input tensor: %w", err)
	}

	// Create cls_8 output tensor: [1, 6400, 1]
	clsShape := ort.NewShape(1, 6400, 1)
	clsTensor, err = ort.NewEmptyTensor[float32](clsShape)
	if err != nil {
		return fmt.Errorf("failed to create cls tensor: %w", err)
	}

	// Create bbox_8 output tensor: [1, 6400, 4]
	bboxShape := ort.NewShape(1, 6400, 4)
	bboxTensor, err = ort.NewEmptyTensor[float32](bboxShape)
	if err != nil {
		return fmt.Errorf("failed to create bbox tensor: %w", err)
	}

	// Create ONNX session with 2 outputs
	yunetSession, err = ort.NewAdvancedSession(
		modelPath,
		[]string{"input"},
		[]string{"cls_8", "bbox_8"},
		[]ort.Value{inputTensor},
		[]ort.Value{clsTensor, bboxTensor},
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create ONNX session: %w", err)
	}

	// Generate anchors
	anchors = generateAnchors()
	log.Printf("Generated %d anchors for YuNet detection", len(anchors))

	return nil
}

// generateAnchors creates anchor points for stride-8 feature map
func generateAnchors() []Anchor {
	var result []Anchor
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			// Anchor center in input image coordinates
			cx := (float32(x) + 0.5) * float32(stride)
			cy := (float32(y) + 0.5) * float32(stride)
			result = append(result, Anchor{CX: cx, CY: cy})
		}
	}
	return result
}

// Cleanup releases ONNX Runtime resources
func Cleanup() {
	if yunetSession != nil {
		yunetSession.Destroy()
	}
	if inputTensor != nil {
		inputTensor.Destroy()
	}
	if clsTensor != nil {
		clsTensor.Destroy()
	}
	if bboxTensor != nil {
		bboxTensor.Destroy()
	}
	ort.DestroyEnvironment()
}

// DetectFaces runs face detection on an image file
func DetectFaces(imagePath string) ([]FaceDetection, error) {
	if yunetSession == nil || inputTensor == nil {
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

	// Preprocess image
	err = preprocessImage(img)
	if err != nil {
		return nil, fmt.Errorf("failed to preprocess image: %w", err)
	}

	// Run inference
	err = yunetSession.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run inference: %w", err)
	}

	// Parse and decode detections
	detections := parseAndDecodeDetections()

	// Apply NMS
	detections = applyNMS(detections, iouThreshold)

	return detections, nil
}

// preprocessImage resizes and normalizes the image for YuNet input
func preprocessImage(img image.Image) error {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	inputData := inputTensor.GetData()

	// Resize and convert to BGR, NCHW format
	for y := 0; y < inputHeight; y++ {
		for x := 0; x < inputWidth; x++ {
			// Map to original image coordinates
			origX := x * width / inputWidth
			origY := y * height / inputHeight

			r, g, b, _ := img.At(origX, origY).RGBA()

			// Convert to float32 and normalize (0-255 range)
			// YuNet expects BGR format
			inputData[0*inputHeight*inputWidth+y*inputWidth+x] = float32(b >> 8) // B
			inputData[1*inputHeight*inputWidth+y*inputWidth+x] = float32(g >> 8) // G
			inputData[2*inputHeight*inputWidth+y*inputWidth+x] = float32(r >> 8) // R
		}
	}

	return nil
}

// parseAndDecodeDetections decodes YuNet outputs to face detections
func parseAndDecodeDetections() []FaceDetection {
	clsData := clsTensor.GetData()
	bboxData := bboxTensor.GetData()
	
	var detections []FaceDetection

	// Process each anchor
	for i := 0; i < 6400; i++ {
		// Get confidence from cls_8 [1, 6400, 1]
		confidence := sigmoid(clsData[i])

		if confidence < confidenceThreshold {
			continue
		}

		// Get bbox offsets from bbox_8 [1, 6400, 4]
		// Format: [dx, dy, dw, dh] - offsets relative to anchor
		dx := bboxData[i*4+0]
		dy := bboxData[i*4+1]
		dw := bboxData[i*4+2]
		dh := bboxData[i*4+3]

		// Decode bbox using anchor
		anchor := anchors[i]
		
		// YuNet outputs direct bbox coordinates relative to anchor
		// Not exponential - that was causing huge invalid boxes
		cx := anchor.CX + dx*float32(stride)
		cy := anchor.CY + dy*float32(stride)
		
		// Width and height are also direct values, not log-space
		w := dw * float32(stride)
		h := dh * float32(stride)
		
		// Take absolute values to handle negative predictions
		if w < 0 {
			w = -w
		}
		if h < 0 {
			h = -h
		}
		
		// Convert to top-left corner format
		x := cx - w/2
		y := cy - h/2

		// Validate bbox: filter out unreasonable detections
		// Face should be at least 10x10 pixels and at most the entire image
		const minSize = 10.0
		const maxSize = float32(inputWidth)
		
		if w < minSize || h < minSize || w > maxSize || h > maxSize {
			continue // Skip unreasonable sizes
		}
		
		// Check if bbox is within image bounds
		if x < 0 || y < 0 || x+w > float32(inputWidth) || y+h > float32(inputHeight) {
			continue // Skip out-of-bounds detections
		}

		detection := FaceDetection{
			X:          x,
			Y:          y,
			Width:      w,
			Height:     h,
			Confidence: confidence,
		}

		detections = append(detections, detection)
	}

	return detections
}

// sigmoid applies sigmoid activation
func sigmoid(x float32) float32 {
	return 1.0 / (1.0 + float32(math.Exp(float64(-x))))
}

// applyNMS applies Non-Maximum Suppression to filter overlapping detections
func applyNMS(detections []FaceDetection, iouThreshold float32) []FaceDetection {
	if len(detections) == 0 {
		return detections
	}

	// Sort by confidence (descending)
	sort.Slice(detections, func(i, j int) bool {
		return detections[i].Confidence > detections[j].Confidence
	})

	var keep []FaceDetection
	used := make([]bool, len(detections))

	for i := 0; i < len(detections); i++ {
		if used[i] {
			continue
		}

		keep = append(keep, detections[i])
		used[i] = true

		// Suppress overlapping boxes
		for j := i + 1; j < len(detections); j++ {
			if used[j] {
				continue
			}

			iou := calculateIoU(detections[i], detections[j])
			if iou > iouThreshold {
				used[j] = true
			}
		}
	}

	return keep
}

// calculateIoU calculates Intersection over Union between two detections
func calculateIoU(a, b FaceDetection) float32 {
	// Calculate intersection area
	x1 := max(a.X, b.X)
	y1 := max(a.Y, b.Y)
	x2 := min(a.X+a.Width, b.X+b.Width)
	y2 := min(a.Y+a.Height, b.Y+b.Height)

	if x2 < x1 || y2 < y1 {
		return 0
	}

	intersection := (x2 - x1) * (y2 - y1)

	// Calculate union area
	areaA := a.Width * a.Height
	areaB := b.Width * b.Height
	union := areaA + areaB - intersection

	if union == 0 {
		return 0
	}

	return intersection / union
}

func max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
