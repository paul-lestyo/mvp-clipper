package face

import (
	"math"
)

// distance calculates Euclidean distance between two points
func distance(p1, p2 Point) float32 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}
