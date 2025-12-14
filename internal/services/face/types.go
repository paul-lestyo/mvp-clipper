package face

// TimelineEntry represents a point in time with a cropping mode
type TimelineEntry struct {
	Timestamp float64 // seconds from start
	Mode      string  // "center" or "split"
}

// FaceDetection represents a detected face with bounding box and confidence
type FaceDetection struct {
	X          float32 // bounding box x
	Y          float32 // bounding box y
	Width      float32 // bounding box width
	Height     float32 // bounding box height
	Confidence float32 // detection confidence
}
