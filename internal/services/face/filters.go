package face

import (
	"log"
)

const (
	// Size constraints (relative to frame height)
	minFaceHeightRatio = 0.10 // Minimum 10% of frame height
	maxFaceHeightRatio = 0.80 // Maximum 80% of frame height

	// Aspect ratio constraints
	minAspectRatio = 0.6 // Minimum width/height ratio
	maxAspectRatio = 1.4 // Maximum width/height ratio

	// Landmark validation thresholds
	minEyeDistanceRatio = 0.2  // Eye distance should be at least 20% of face width
	minNoseDistanceRatio = 0.15 // Nose to eye midpoint should be at least 15% of face height
)

// FilterDetections applies all filters to remove false positives
func FilterDetections(detections []FaceDetection, frameWidth, frameHeight int) []FaceDetection {
	filtered := make([]FaceDetection, 0)

	for _, det := range detections {
		// Apply all filters
		if !filterBySize(det, frameHeight) {
			log.Printf("[FILTER] Rejected by size: %.1fx%.1f (frame height: %d)", det.Width, det.Height, frameHeight)
			continue
		}

		if !filterByAspectRatio(det) {
			log.Printf("[FILTER] Rejected by aspect ratio: %.2f", det.Width/det.Height)
			continue
		}

		if !filterByLandmarks(det) {
			log.Printf("[FILTER] Rejected by landmark geometry")
			continue
		}

		filtered = append(filtered, det)
	}

	log.Printf("[FILTER] Kept %d/%d detections after filtering", len(filtered), len(detections))
	return filtered
}

// filterBySize checks if face size is within acceptable range
func filterBySize(det FaceDetection, frameHeight int) bool {
	minHeight := float32(frameHeight) * minFaceHeightRatio
	maxHeight := float32(frameHeight) * maxFaceHeightRatio

	return det.Height >= minHeight && det.Height <= maxHeight
}

// filterByAspectRatio checks if face aspect ratio is reasonable
func filterByAspectRatio(det FaceDetection) bool {
	if det.Height == 0 {
		return false
	}

	aspectRatio := det.Width / det.Height
	return aspectRatio >= minAspectRatio && aspectRatio <= maxAspectRatio
}

// filterByLandmarks validates face geometry using landmarks
func filterByLandmarks(det FaceDetection) bool {
	// Extract landmarks
	leftEye := det.Landmarks[0]
	rightEye := det.Landmarks[1]
	nose := det.Landmarks[2]

	// Calculate eye distance
	eyeDistance := distance(leftEye, rightEye)

	// Eye distance should be at least 20% of face width
	if eyeDistance < det.Width*minEyeDistanceRatio {
		return false
	}

	// Calculate eye midpoint
	eyeMidpoint := Point{
		X: (leftEye.X + rightEye.X) / 2,
		Y: (leftEye.Y + rightEye.Y) / 2,
	}

	// Nose should be below eye midpoint
	noseToEyeDistance := distance(nose, eyeMidpoint)

	// Distance should be at least 15% of face height
	if noseToEyeDistance < det.Height*minNoseDistanceRatio {
		return false
	}

	return true
}

