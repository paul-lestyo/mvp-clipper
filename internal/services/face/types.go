package face

// Point represents a 2D coordinate
type Point struct {
	X float32
	Y float32
}

// TimelineEntry represents a point in time with a cropping mode
type TimelineEntry struct {
	Timestamp  float64        // seconds from start
	Mode       string         // "center" or "split"
	BBox       *BoundingBox   // face bounding box (nil if no detection)
	Center     *Point         // Primary center point (avg if multiple or single)
	Centers    []Point        // List of all center points (important for split mode)
	Confidence float32        // detection confidence (0.0 if no detection)
	Status     DetectionStatus // detection status
}

// BoundingBox represents a rectangular region
type BoundingBox struct {
	X      float32 // top-left x
	Y      float32 // top-left y
	Width  float32 // width
	Height float32 // height
}

// FaceDetection represents a detected face with bounding box, landmarks, and confidence
type FaceDetection struct {
	X          float32   // bounding box x
	Y          float32   // bounding box y
	Width      float32   // bounding box width
	Height     float32   // bounding box height
	Confidence float32   // detection confidence
	Landmarks  [5]Point  // facial landmarks: [left_eye, right_eye, nose, left_mouth, right_mouth]
	Score      float32   // selection score (computed by selector)
}

// StabilizedFace represents a temporally smoothed face detection
type StabilizedFace struct {
	X          float32 // smoothed bbox x
	Y          float32 // smoothed bbox y
	Width      float32 // smoothed bbox width
	Height     float32 // smoothed bbox height
	Confidence float32 // current confidence
	LastSeen   int     // frame index when last detected
}

// DetectionStatus represents the status of a detection
type DetectionStatus string

const (
	StatusDetected      DetectionStatus = "detected"
	StatusInterpolated  DetectionStatus = "interpolated"
	StatusNoDetection   DetectionStatus = "no_detection"
	StatusLowConfidence DetectionStatus = "low_confidence"
)

// Center returns the center point of a face detection
func (f *FaceDetection) Center() Point {
	return Point{
		X: f.X + f.Width/2,
		Y: f.Y + f.Height/2,
	}
}

// Center returns the center point of a stabilized face
func (s *StabilizedFace) Center() Point {
	return Point{
		X: s.X + s.Width/2,
		Y: s.Y + s.Height/2,
	}
}

// ToBoundingBox converts FaceDetection to BoundingBox
func (f *FaceDetection) ToBoundingBox() BoundingBox {
	return BoundingBox{
		X:      f.X,
		Y:      f.Y,
		Width:  f.Width,
		Height: f.Height,
	}
}

// ToBoundingBox converts StabilizedFace to BoundingBox
func (s *StabilizedFace) ToBoundingBox() BoundingBox {
	return BoundingBox{
		X:      s.X,
		Y:      s.Y,
		Width:  s.Width,
		Height: s.Height,
	}
}
