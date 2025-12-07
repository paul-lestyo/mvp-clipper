package utils

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type SRTEntry struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Text  string `json:"text"`
}

func ParseSRT(path string) ([]SRTEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []SRTEntry
	var entry SRTEntry
	var textLines []string

	reTime := regexp.MustCompile(`(\d\d:\d\d:\d\d[,\.]\d\d\d) --> (\d\d:\d\d:\d\d[,\.]\d\d\d)`)

	sc := bufio.NewScanner(file)
	for sc.Scan() {
		line := sc.Text()
		line = strings.TrimSpace(line)

		// Empty line = end of current entry
		if line == "" {
			if entry.Start != "" && len(textLines) > 0 {
				entry.Text = strings.Join(textLines, " ")
				entries = append(entries, entry)
			}
			entry = SRTEntry{}
			textLines = nil
			continue
		}

		// Match timestamp line
		if matches := reTime.FindStringSubmatch(line); len(matches) == 3 {
			// Save previous entry if exists
			if entry.Start != "" && len(textLines) > 0 {
				entry.Text = strings.Join(textLines, " ")
				entries = append(entries, entry)
				textLines = nil
			}
			entry = SRTEntry{
				Start: strings.Replace(matches[1], ",", ".", 1),
				End:   strings.Replace(matches[2], ",", ".", 1),
			}
			continue
		}

		// Skip sequence number
		if isNumber(line) {
			continue
		}

		// Text line - accumulate
		if entry.Start != "" {
			textLines = append(textLines, line)
		}
	}

	// Don't forget last entry
	if entry.Start != "" && len(textLines) > 0 {
		entry.Text = strings.Join(textLines, " ")
		entries = append(entries, entry)
	}

	return entries, nil
}

func isNumber(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// TimeToSeconds converts timestamp string (HH:MM:SS or HH:MM:SS.mmm) to seconds
func TimeToSeconds(t string) float64 {
	// Replace comma with dot for SRT format
	t = strings.Replace(t, ",", ".", 1)
	
	parts := strings.Split(t, ":")
	if len(parts) != 3 {
		return 0
	}
	
	var h, m float64
	var s float64
	
	fmt.Sscanf(parts[0], "%f", &h)
	fmt.Sscanf(parts[1], "%f", &m)
	fmt.Sscanf(parts[2], "%f", &s)
	
	return h*3600 + m*60 + s
}

// SecondsToSRTTime converts seconds to SRT timestamp format (HH:MM:SS,mmm)
func SecondsToSRTTime(seconds float64) string {
	h := int(seconds / 3600)
	m := int((seconds - float64(h)*3600) / 60)
	s := seconds - float64(h)*3600 - float64(m)*60
	
	return fmt.Sprintf("%02d:%02d:%06.3f", h, m, s)
}

// CutSRT extracts subtitles between start and end time, adjusting timestamps to start from 0
// Merges overlapping entries by combining their text
func CutSRT(inputPath, outputPath, start, end string) error {
	entries, err := ParseSRT(inputPath)
	if err != nil {
		return err
	}

	startSec := TimeToSeconds(start)
	endSec := TimeToSeconds(end)

	// First, filter entries within range
	var filtered []SRTEntry
	for _, entry := range entries {
		entrySt := TimeToSeconds(entry.Start)
		entryEn := TimeToSeconds(entry.End)

		// Check if entry is within range
		if entryEn > startSec && entrySt < endSec {
			filtered = append(filtered, entry)
		}
	}

	// Merge overlapping entries by extending end time and keeping first text
	// YouTube auto-captions have overlapping timestamps - we keep original timing
	// but ensure only one subtitle shows at a time by adjusting end times
	var merged []SRTEntry
	for i, entry := range filtered {
		if i == 0 {
			merged = append(merged, entry)
			continue
		}

		entrySt := TimeToSeconds(entry.Start)
		lastIdx := len(merged) - 1
		prevEnd := TimeToSeconds(merged[lastIdx].End)

		if entrySt < prevEnd {
			// Overlap detected - adjust previous entry's end time to current start
			merged[lastIdx].End = entry.Start
		}

		merged = append(merged, entry)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	index := 1
	for _, entry := range merged {
		entrySt := TimeToSeconds(entry.Start)
		entryEn := TimeToSeconds(entry.End)

		// Adjust timestamps relative to clip start
		newStart := entrySt - startSec
		newEnd := entryEn - startSec

		// Clamp to valid range
		if newStart < 0 {
			newStart = 0
		}
		if newEnd > endSec-startSec {
			newEnd = endSec - startSec
		}

		// Write SRT entry
		fmt.Fprintf(file, "%d\n", index)
		fmt.Fprintf(file, "%s --> %s\n",
			strings.Replace(SecondsToSRTTime(newStart), ".", ",", 1),
			strings.Replace(SecondsToSRTTime(newEnd), ".", ",", 1))
		fmt.Fprintf(file, "%s\n\n", entry.Text)
		index++
	}

	return nil
}
