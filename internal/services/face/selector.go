package face

import (
	"log"
	"math"
)

const (
	// Scoring weights
	sizeWeight     = 0.4 // 40% weight for face size
	positionWeight = 0.3 // 30% weight for center position
	confidenceWeight = 0.2 // 20% weight for detection confidence
	verticalWeight = 0.1 // 10% weight for vertical position bias
)

// SelectPrimaryFace selects the most prominent face from multiple candidates
func SelectPrimaryFace(candidates []FaceDetection, frameWidth, frameHeight int) *FaceDetection {
	if len(candidates) == 0 {
		return nil
	}

	if len(candidates) == 1 {
		candidates[0].Score = ScoreFace(candidates[0], frameWidth, frameHeight)
		return &candidates[0]
	}

	// Score all candidates
	for i := range candidates {
		candidates[i].Score = ScoreFace(candidates[i], frameWidth, frameHeight)
	}

	// Find highest scoring face
	bestIdx := 0
	bestScore := candidates[0].Score

	for i := 1; i < len(candidates); i++ {
		if candidates[i].Score > bestScore {
			bestScore = candidates[i].Score
			bestIdx = i
		}
	}

	log.Printf("[SELECTOR] Selected face %d/%d with score %.3f", bestIdx+1, len(candidates), bestScore)
	return &candidates[bestIdx]
}

// ScoreFace computes a weighted score for face selection
func ScoreFace(face FaceDetection, frameWidth, frameHeight int) float32 {
	// 1. Size score (larger faces are more important)
	faceArea := face.Width * face.Height
	frameArea := float32(frameWidth * frameHeight)
	sizeScore := faceArea / frameArea

	// 2. Position score (center-weighted)
	faceCenter := face.Center()
	frameCenterX := float32(frameWidth) / 2
	frameCenterY := float32(frameHeight) / 2

	distFromCenter := distance(faceCenter, Point{X: frameCenterX, Y: frameCenterY})
	maxDist := float32(math.Sqrt(float64(frameCenterX*frameCenterX + frameCenterY*frameCenterY)))
	positionScore := 1.0 - (distFromCenter / maxDist)

	// 3. Confidence score
	confScore := face.Confidence

	// 4. Vertical position bias (prefer upper 2/3 of frame)
	verticalBias := float32(1.0)
	if faceCenter.Y > float32(frameHeight)*0.66 {
		verticalBias = 0.7 // Penalize lower third
	}

	// Weighted combination
	finalScore := (sizeScore * sizeWeight) +
		(positionScore * positionWeight) +
		(confScore * confidenceWeight) +
		(verticalBias * verticalWeight)

	log.Printf("[SELECTOR] Face score: size=%.3f pos=%.3f conf=%.3f vert=%.3f → total=%.3f",
		sizeScore, positionScore, confScore, verticalBias, finalScore)

	return finalScore
}
