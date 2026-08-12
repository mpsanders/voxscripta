package ytdlp

import "strings"

// SelectTrack deterministically chooses a caption track. Languages are
// considered in caller order, with exact matches before base and regional
// fallbacks. Manual tracks win ties; automatic tracks require allowAutomatic.
// When languages is empty, originalLanguage is tried first and the first
// deterministic eligible track is the final fallback.
func SelectTrack(tracks []CaptionTrack, originalLanguage string, languages []string, allowAutomatic bool) (CaptionTrack, bool) {
	preferences := normalizedLanguages(languages)
	hasExplicitPreferences := len(preferences) != 0
	if len(preferences) == 0 && strings.TrimSpace(originalLanguage) != "" {
		preferences = []string{strings.TrimSpace(originalLanguage)}
	}
	for _, preference := range preferences {
		for rank := 0; rank <= 2; rank++ {
			if track, ok := selectMatchingTrack(tracks, preference, rank, allowAutomatic); ok {
				return track, true
			}
		}
	}
	if hasExplicitPreferences {
		return CaptionTrack{}, false
	}
	for _, source := range []CaptionSource{CaptionManual, CaptionAutomatic} {
		if source == CaptionAutomatic && !allowAutomatic {
			continue
		}
		for _, track := range tracks {
			if track.Source == source {
				return track, true
			}
		}
	}
	return CaptionTrack{}, false
}

// normalizedLanguages trims preferences, removes empty values, and preserves
// the first occurrence of each case-insensitive language tag.
func normalizedLanguages(languages []string) []string {
	result := make([]string, 0, len(languages))
	seen := make(map[string]struct{}, len(languages))
	for _, language := range languages {
		language = strings.TrimSpace(language)
		key := strings.ToLower(language)
		if language == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, language)
	}
	return result
}

// selectMatchingTrack returns the first source-preferred track at match rank.
func selectMatchingTrack(tracks []CaptionTrack, preference string, rank int, allowAutomatic bool) (CaptionTrack, bool) {
	for _, source := range []CaptionSource{CaptionManual, CaptionAutomatic} {
		if source == CaptionAutomatic && !allowAutomatic {
			continue
		}
		for _, track := range tracks {
			if track.Source == source && languageMatchRank(preference, track.Language) == rank {
				return track, true
			}
		}
	}
	return CaptionTrack{}, false
}

// languageMatchRank ranks an exact tag, its base tag, then another regional
// variant. A negative result means the tags do not share a base language.
func languageMatchRank(preference, candidate string) int {
	preference = strings.ToLower(strings.TrimSpace(preference))
	candidate = strings.ToLower(strings.TrimSpace(candidate))
	if preference == "" || candidate == "" {
		return -1
	}
	if preference == candidate {
		return 0
	}
	base := strings.SplitN(preference, "-", 2)[0]
	if candidate == base {
		return 1
	}
	if strings.SplitN(candidate, "-", 2)[0] == base {
		return 2
	}
	return -1
}
