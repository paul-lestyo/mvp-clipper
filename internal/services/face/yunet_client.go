package face

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

// YuNetClient communicates with Python YuNet service via Unix socket
type YuNetClient struct {
	socketPath string
	timeout    time.Duration
}

// InferenceRequest is sent to Python service
type InferenceRequest struct {
	Height int    `msgpack:"h"`
	Width  int    `msgpack:"w"`
	Data   []byte `msgpack:"d"` // RGB uint8, row-major, shape (H, W, 3)
}

// Detection represents a face detection from YuNet
type YuNetDetection struct {
	X          float32   `msgpack:"x"`
	Y          float32   `msgpack:"y"`
	Width      float32   `msgpack:"w"`
	Height     float32   `msgpack:"h"`
	Confidence float32   `msgpack:"c"`
	Landmarks  []float32 `msgpack:"l"` // 10 values: [x1,y1, x2,y2, x3,y3, x4,y4, x5,y5]
}

// InferenceResponse is received from Python service
type InferenceResponse struct {
	Detections  []YuNetDetection `msgpack:"detections"`
	InferenceMs float32          `msgpack:"inference_ms"`
}

// NewYuNetClient creates a new client for the Python YuNet service
func NewYuNetClient(socketPath string) *YuNetClient {
	return &YuNetClient{
		socketPath: socketPath,
		timeout:    100 * time.Millisecond, // 100ms timeout
	}
}

// Detect sends a frame to Python service and returns detections
func (c *YuNetClient) Detect(frameData []byte, width, height int) ([]FaceDetection, error) {
	// Connect to Unix socket
	conn, err := net.DialTimeout("unix", c.socketPath, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Python service: %w", err)
	}
	defer conn.Close()

	// Set deadline
	conn.SetDeadline(time.Now().Add(c.timeout))

	// Create request
	req := InferenceRequest{
		Height: height,
		Width:  width,
		Data:   frameData,
	}

	// Encode with msgpack
	reqData, err := msgpack.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	// Send request
	_, err = conn.Write(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	respData, err := io.ReadAll(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Decode response
	var resp InferenceResponse
	err = msgpack.Unmarshal(respData, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to FaceDetection format
	detections := make([]FaceDetection, len(resp.Detections))
	for i, det := range resp.Detections {
		// Convert landmarks from flat array to [5]Point
		var landmarks [5]Point
		for j := 0; j < 5 && j*2+1 < len(det.Landmarks); j++ {
			landmarks[j] = Point{
				X: det.Landmarks[j*2],
				Y: det.Landmarks[j*2+1],
			}
		}

		detections[i] = FaceDetection{
			X:          det.X,
			Y:          det.Y,
			Width:      det.Width,
			Height:     det.Height,
			Confidence: det.Confidence,
			Landmarks:  landmarks,
		}
	}

	return detections, nil
}

// DetectWithFallback attempts detection with fallback on error
func (c *YuNetClient) DetectWithFallback(frameData []byte, width, height int) []FaceDetection {
	detections, err := c.Detect(frameData, width, height)
	if err != nil {
		// Log error but return empty detections
		// Temporal tracker will handle missing detections
		return []FaceDetection{}
	}
	return detections
}
