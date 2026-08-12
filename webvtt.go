package transcript

import (
	"bufio"
	"errors"
	"fmt"
	"html"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
)

const maxWebVTTLineSize = 1024 * 1024

var (
	webVTTTimingPattern = regexp.MustCompile(`^([^[:space:]]+)[[:space:]]+-->[[:space:]]+([^[:space:]]+)`)
	webVTTTagPattern    = regexp.MustCompile(`<[^>]*>`)
)

// ParseWebVTT parses WebVTT subtitle data from r into ordered, normalized
// segments. The reader must contain a WEBVTT header. Cue settings and cue
// identifiers are accepted but are not retained by the normalized model.
func ParseWebVTT(r io.Reader) ([]Segment, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: WebVTT reader must not be nil", ErrInvalidInput)
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), maxWebVTTLineSize)
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, strings.TrimSuffix(scanner.Text(), "\r"))
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return nil, fmt.Errorf("%w: WebVTT line exceeds %d bytes", ErrInvalidInput, maxWebVTTLineSize)
		}
		return nil, fmt.Errorf("%w: read WebVTT: %v", ErrInvalidInput, err)
	}
	if len(lines) == 0 || !validWebVTTHeader(lines[0]) {
		return nil, fmt.Errorf("%w: missing WEBVTT header", ErrUnsupportedFormat)
	}

	segments, err := parseWebVTTBlocks(lines[1:])
	if err != nil {
		return nil, err
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("%w: WebVTT contains no non-empty cues", ErrTranscriptUnavailable)
	}
	sort.SliceStable(segments, func(i, j int) bool { return segments[i].Start < segments[j].Start })
	return normalizeRollingSegments(segments), nil
}

// validWebVTTHeader reports whether line is a valid WebVTT file signature.
func validWebVTTHeader(line string) bool {
	line = strings.TrimPrefix(line, "\ufeff")
	return line == "WEBVTT" || strings.HasPrefix(line, "WEBVTT ") || strings.HasPrefix(line, "WEBVTT\t")
}

// parseWebVTTBlocks parses blank-line-delimited WebVTT blocks after the header.
func parseWebVTTBlocks(lines []string) ([]Segment, error) {
	var segments []Segment
	for i := 0; i < len(lines); {
		for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
			i++
		}
		if i >= len(lines) {
			break
		}
		start := i
		for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			i++
		}
		block := lines[start:i]
		if isWebVTTMetadataBlock(block[0]) {
			continue
		}
		segment, keep, err := parseWebVTTCue(block)
		if err != nil {
			return nil, fmt.Errorf("%w: block beginning on line %d: %v", ErrInvalidInput, start+2, err)
		}
		if keep {
			segments = append(segments, segment)
		}
	}
	return segments, nil
}

// isWebVTTMetadataBlock reports whether a block is NOTE, STYLE, or REGION data.
func isWebVTTMetadataBlock(line string) bool {
	return line == "NOTE" || strings.HasPrefix(line, "NOTE ") || line == "STYLE" || line == "REGION"
}

// parseWebVTTCue converts one cue block into a normalized segment.
func parseWebVTTCue(block []string) (Segment, bool, error) {
	timingIndex := 0
	if !strings.Contains(block[0], "-->") {
		timingIndex = 1
	}
	if timingIndex >= len(block) {
		return Segment{}, false, errors.New("cue has no timing line")
	}
	matches := webVTTTimingPattern.FindStringSubmatch(block[timingIndex])
	if matches == nil {
		return Segment{}, false, errors.New("invalid cue timing")
	}
	start, err := parseWebVTTTimestamp(matches[1])
	if err != nil {
		return Segment{}, false, fmt.Errorf("invalid start timestamp: %v", err)
	}
	end, err := parseWebVTTTimestamp(matches[2])
	if err != nil {
		return Segment{}, false, fmt.Errorf("invalid end timestamp: %v", err)
	}
	if end < start {
		return Segment{}, false, errors.New("cue end precedes start")
	}
	text := normalizeWebVTTText(strings.Join(block[timingIndex+1:], "\n"))
	if text == "" {
		return Segment{}, false, nil
	}
	return Segment{Start: start, End: end, Text: text}, true, nil
}

// parseWebVTTTimestamp parses a WebVTT timestamp in mm:ss.mmm or hh:mm:ss.mmm form.
func parseWebVTTTimestamp(value string) (time.Duration, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return 0, errors.New("expected mm:ss.mmm or hh:mm:ss.mmm")
	}
	var hours, minutes int64
	var secondsPart string
	if len(parts) == 3 {
		if _, err := fmt.Sscanf(parts[0], "%d", &hours); err != nil || len(parts[0]) < 2 {
			return 0, errors.New("invalid hours")
		}
		secondsPart = parts[2]
	} else {
		secondsPart = parts[1]
	}
	if _, err := fmt.Sscanf(parts[len(parts)-2], "%d", &minutes); err != nil || len(parts[len(parts)-2]) != 2 {
		return 0, errors.New("invalid minutes")
	}
	secondPieces := strings.Split(secondsPart, ".")
	if len(secondPieces) != 2 || len(secondPieces[0]) != 2 || len(secondPieces[1]) != 3 {
		return 0, errors.New("invalid seconds")
	}
	var seconds, millis int64
	if _, err := fmt.Sscanf(secondPieces[0], "%d", &seconds); err != nil {
		return 0, errors.New("invalid seconds")
	}
	if _, err := fmt.Sscanf(secondPieces[1], "%d", &millis); err != nil {
		return 0, errors.New("invalid milliseconds")
	}
	if minutes > 59 || seconds > 59 || hours < 0 || minutes < 0 || seconds < 0 || millis < 0 {
		return 0, errors.New("timestamp component out of range")
	}
	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second + time.Duration(millis)*time.Millisecond, nil
}

// normalizeWebVTTText removes cue markup, decodes HTML entities, and collapses whitespace.
func normalizeWebVTTText(value string) string {
	value = webVTTTagPattern.ReplaceAllString(value, "")
	value = html.UnescapeString(value)
	return strings.Join(strings.Fields(value), " ")
}

// normalizeRollingSegments removes repeated leading words from overlapping,
// consecutive cues. Fully repeated cues are discarded; non-overlapping cues
// are retained unchanged.
func normalizeRollingSegments(segments []Segment) []Segment {
	result := make([]Segment, 0, len(segments))
	var previous Segment
	for _, segment := range segments {
		if len(result) == 0 || segment.Start > previous.End {
			result = append(result, segment)
			previous = segment
			continue
		}
		previousWords := strings.Fields(previous.Text)
		currentWords := strings.Fields(segment.Text)
		common := longestWordOverlap(previousWords, currentWords)
		previous = segment
		if common == len(currentWords) {
			continue
		}
		if common == 0 {
			result = append(result, segment)
			continue
		}
		segment.Text = strings.Join(currentWords[common:], " ")
		result = append(result, segment)
	}
	return result
}

// longestWordOverlap returns the largest suffix of previous that is also a
// prefix of current.
func longestWordOverlap(previous, current []string) int {
	maximum := min(len(previous), len(current))
	for size := maximum; size > 0; size-- {
		matches := true
		for i := 0; i < size; i++ {
			if previous[len(previous)-size+i] != current[i] {
				matches = false
				break
			}
		}
		if matches {
			return size
		}
	}
	return 0
}
