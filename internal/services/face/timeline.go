package face

import (
	"log"
	"math"
)

// CompressTimeline removes redundant timeline entries
// For "center" mode, we only merge if the Center position is stable.
func CompressTimeline(entries []TimelineEntry) []TimelineEntry {
	if len(entries) == 0 {
		return entries
	}
	
	log.Printf("[DEBUG] Compressing timeline with %d initial entries", len(entries))

	compressed := []TimelineEntry{entries[0]}
	last := entries[0]
	
	for i := 1; i < len(entries); i++ {
		current := entries[i]
		
		shouldKeep := false
		
		// If mode changed, definitely keep
		if current.Mode != last.Mode {
			shouldKeep = true
		} else {
			// Mode is same. Check if centers moved significantly.
			// Compare lengths first
			if len(current.Centers) != len(last.Centers) {
				shouldKeep = true
			} else {
				// Compare each center
				for j, c := range current.Centers {
					l := last.Centers[j]
					dist := math.Abs(float64(c.X - l.X))
					if dist > 2.0 {
						shouldKeep = true
						break
					}
				}
			}
		}
		
		if shouldKeep {
			compressed = append(compressed, current)
			last = current
		}
	}
	
	log.Printf("[DEBUG] Timeline compressed to %d entries (%.1f%% reduction)", 
		len(compressed), 100.0 - (float64(len(compressed))/float64(len(entries))*100.0))

	return compressed
}
