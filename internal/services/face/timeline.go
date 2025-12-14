package face

// CompressTimeline removes redundant timeline entries, keeping only mode changes
func CompressTimeline(entries []TimelineEntry) []TimelineEntry {
	if len(entries) == 0 {
		return entries
	}

	compressed := []TimelineEntry{entries[0]}
	
	for i := 1; i < len(entries); i++ {
		// Only keep if mode changed from previous entry
		if entries[i].Mode != entries[i-1].Mode {
			compressed = append(compressed, entries[i])
		}
	}

	return compressed
}
