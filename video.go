package transcript

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var videoIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// ParseVideoID extracts a YouTube video ID from a raw 11-character ID or a
// supported youtube.com, youtube-nocookie.com, or youtu.be HTTP(S) URL.
func ParseVideoID(input string) (string, error) {
	input = strings.TrimSpace(input)
	if videoIDPattern.MatchString(input) {
		return input, nil
	}

	u, err := url.Parse(input)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return "", fmt.Errorf("%w: expected a YouTube video ID or HTTP(S) URL", ErrInvalidInput)
	}

	host := strings.ToLower(u.Hostname())
	var id string
	switch {
	case host == "youtu.be":
		id = firstPathPart(u.Path)
	case host == "youtube.com" || strings.HasSuffix(host, ".youtube.com"):
		id = youtubeURLVideoID(u)
	case host == "youtube-nocookie.com" || strings.HasSuffix(host, ".youtube-nocookie.com"):
		if parts := pathParts(u.Path); len(parts) == 2 && parts[0] == "embed" {
			id = parts[1]
		}
	}
	if !videoIDPattern.MatchString(id) {
		return "", fmt.Errorf("%w: URL does not contain a valid YouTube video ID", ErrInvalidInput)
	}
	return id, nil
}

// youtubeURLVideoID extracts an ID from supported paths on a YouTube URL.
func youtubeURLVideoID(u *url.URL) string {
	parts := pathParts(u.Path)
	if len(parts) == 1 && parts[0] == "watch" {
		return u.Query().Get("v")
	}
	if len(parts) == 2 {
		switch parts[0] {
		case "embed", "shorts", "live":
			return parts[1]
		}
	}
	return ""
}

// firstPathPart returns the only non-empty path component, or an empty string
// when path contains zero or multiple components.
func firstPathPart(path string) string {
	parts := pathParts(path)
	if len(parts) == 1 {
		return parts[0]
	}
	return ""
}

// pathParts splits a URL path into its non-empty slash-delimited components.
func pathParts(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool { return r == '/' })
}
