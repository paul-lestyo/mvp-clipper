package face

import (
	"log"
	"math"
)

const (
	// EMA smoothing parameter (0.0 = no update, 1.0 = no smoothing)
	emaAlpha = 0.3

	// Maximum consecutive frames to carry forward detection
	maxMissingFrames = 3

	// Confidence decay factor per missed frame
	confidenceDecayFactor = 0.8

	// Maximum spatial jump (as fraction of frame width)
	maxSpatialJumpRatio = 0.3
)

// TemporalTracker manages temporal stabilization of face detections
type TemporalTracker struct {
	stabilized *StabilizedFace
	frameWidth int
}

// NewTemporalTracker creates a new temporal tracker
func NewTemporalTracker(frameWidth int) *TemporalTracker {
	return &TemporalTracker{
		stabilized: nil,
		frameWidth: frameWidth,
	}
}

// Update processes a new detection and returns the stabilized result
func (t *TemporalTracker) Update(detected *FaceDetection, frameIdx int) *StabilizedFace {
	if detected == nil {
		return t.handleMissedDetection(frameIdx)
	}

	if t.stabilized == nil {
		// First detection, initialize
		return t.initializeTracking(detected, frameIdx)
	}

	// Check spatial consistency
	if !t.isSpatiallyConsistent(detected) {
		log.Printf("[TRACKER] Spatial jump detected, resetting tracker")
		return t.initializeTracking(detected, frameIdx)
	}

	// Apply EMA smoothing
	return t.applyEMA(detected, frameIdx)
}

// GetStabilized returns the current stabilized face (may be nil)
func (t *TemporalTracker) GetStabilized() *StabilizedFace {
	return t.stabilized
}

// Reset clears the tracker state
func (t *TemporalTracker) Reset() {
	t.stabilized = nil
	log.Printf("[TRACKER] Reset")
}

// initializeTracking starts tracking a new face
func (t *TemporalTracker) initializeTracking(detected *FaceDetection, frameIdx int) *StabilizedFace {
	t.stabilized = &StabilizedFace{
		X:          detected.X,
		Y:          detected.Y,
		Width:      detected.Width,
		Height:     detected.Height,
		Confidence: detected.Confidence,
		LastSeen:   frameIdx,
	}

	log.Printf("[TRACKER] Initialized tracking at frame %d", frameIdx)
	return t.stabilized
}

// applyEMA applies exponential moving average smoothing
func (t *TemporalTracker) applyEMA(detected *FaceDetection, frameIdx int) *StabilizedFace {
	// EMA formula: new = α * detected + (1-α) * old
	t.stabilized.X = emaAlpha*detected.X + (1-emaAlpha)*t.stabilized.X
	t.stabilized.Y = emaAlpha*detected.Y + (1-emaAlpha)*t.stabilized.Y
	t.stabilized.Width = emaAlpha*detected.Width + (1-emaAlpha)*t.stabilized.Width
	t.stabilized.Height = emaAlpha*detected.Height + (1-emaAlpha)*t.stabilized.Height
	t.stabilized.Confidence = detected.Confidence // Don't smooth confidence
	t.stabilized.LastSeen = frameIdx

	log.Printf("[TRACKER] EMA update at frame %d: bbox=(%.1f,%.1f,%.1fx%.1f) conf=%.2f",
		frameIdx, t.stabilized.X, t.stabilized.Y, t.stabilized.Width, t.stabilized.Height, t.stabilized.Confidence)

	return t.stabilized
}

// handleMissedDetection manages frames where no face is detected
func (t *TemporalTracker) handleMissedDetection(frameIdx int) *StabilizedFace {
	if t.stabilized == nil {
		return nil
	}

	missedFrames := frameIdx - t.stabilized.LastSeen

	if missedFrames > maxMissingFrames {
		// Too many misses, reset tracking
		log.Printf("[TRACKER] Lost tracking after %d missed frames", missedFrames)
		t.stabilized = nil
		return nil
	}

	// Carry forward with decayed confidence
	decayedConf := t.stabilized.Confidence * float32(math.Pow(float64(confidenceDecayFactor), float64(missedFrames)))

	log.Printf("[TRACKER] Carrying forward detection (missed %d frames, conf %.2f → %.2f)",
		missedFrames, t.stabilized.Confidence, decayedConf)

	// Return copy with decayed confidence
	return &StabilizedFace{
		X:          t.stabilized.X,
		Y:          t.stabilized.Y,
		Width:      t.stabilized.Width,
		Height:     t.stabilized.Height,
		Confidence: decayedConf,
		LastSeen:   t.stabilized.LastSeen,
	}
}

// isSpatiallyConsistent checks if new detection is close to previous position
func (t *TemporalTracker) isSpatiallyConsistent(detected *FaceDetection) bool {
	if t.stabilized == nil {
		return true
	}

	detectedCenter := detected.Center()
	stabilizedCenter := t.stabilized.Center()

	dist := distance(detectedCenter, stabilizedCenter)
	maxJump := float32(t.frameWidth) * maxSpatialJumpRatio

	return dist <= maxJump
}
