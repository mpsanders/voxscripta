package ytdlp

import "testing"

func TestSelectTrack(t *testing.T) {
	manualEN := CaptionTrack{Language: "en", Source: CaptionManual, URL: "manual-en"}
	manualAU := CaptionTrack{Language: "en-AU", Source: CaptionManual, URL: "manual-au"}
	manualFR := CaptionTrack{Language: "fr", Source: CaptionManual, URL: "manual-fr"}
	autoEN := CaptionTrack{Language: "en", Source: CaptionAutomatic, URL: "auto-en"}
	autoUS := CaptionTrack{Language: "en-US", Source: CaptionAutomatic, URL: "auto-us"}
	tests := []struct {
		name           string
		tracks         []CaptionTrack
		original       string
		languages      []string
		allowAutomatic bool
		wantURL        string
		wantOK         bool
	}{
		{name: "exact requested language", tracks: []CaptionTrack{manualEN, manualAU}, languages: []string{"en-AU"}, wantURL: "manual-au", wantOK: true},
		{name: "base fallback before regional", tracks: []CaptionTrack{manualAU, manualEN}, languages: []string{"en-US"}, wantURL: "manual-en", wantOK: true},
		{name: "regional fallback", tracks: []CaptionTrack{manualAU}, languages: []string{"en-US"}, wantURL: "manual-au", wantOK: true},
		{name: "language order beats source", tracks: []CaptionTrack{manualFR, autoEN}, languages: []string{"en", "fr"}, allowAutomatic: true, wantURL: "auto-en", wantOK: true},
		{name: "manual wins source tie", tracks: []CaptionTrack{autoEN, manualEN}, languages: []string{"en"}, allowAutomatic: true, wantURL: "manual-en", wantOK: true},
		{name: "automatic disabled", tracks: []CaptionTrack{autoEN}, languages: []string{"en"}},
		{name: "automatic enabled", tracks: []CaptionTrack{autoEN}, languages: []string{"en"}, allowAutomatic: true, wantURL: "auto-en", wantOK: true},
		{name: "original language when empty", tracks: []CaptionTrack{manualFR, manualEN}, original: "en", wantURL: "manual-en", wantOK: true},
		{name: "provider best manual fallback", tracks: []CaptionTrack{autoUS, manualFR}, allowAutomatic: true, wantURL: "manual-fr", wantOK: true},
		{name: "provider best automatic fallback", tracks: []CaptionTrack{autoUS}, allowAutomatic: true, wantURL: "auto-us", wantOK: true},
		{name: "no fallback for explicit preference", tracks: []CaptionTrack{manualFR}, languages: []string{"de"}},
		{name: "nil tracks", tracks: nil, languages: nil},
		{name: "normalizes preferences", tracks: []CaptionTrack{manualEN}, languages: []string{" ", " EN ", "en"}, wantURL: "manual-en", wantOK: true},
		{name: "blank preferences use provider best", tracks: []CaptionTrack{manualFR}, languages: []string{" ", "\t"}, wantURL: "manual-fr", wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := SelectTrack(tt.tracks, tt.original, tt.languages, tt.allowAutomatic)
			if ok != tt.wantOK {
				t.Fatalf("SelectTrack() ok = %v, want %v", ok, tt.wantOK)
			}
			if got.URL != tt.wantURL {
				t.Errorf("SelectTrack() URL = %q, want %q", got.URL, tt.wantURL)
			}
		})
	}
}

func TestLanguageMatchRank(t *testing.T) {
	tests := []struct {
		name       string
		preference string
		candidate  string
		want       int
	}{
		{name: "exact", preference: "en-US", candidate: "en-US", want: 0},
		{name: "case insensitive", preference: "EN-us", candidate: "en-US", want: 0},
		{name: "base fallback", preference: "en-US", candidate: "en", want: 1},
		{name: "regional fallback", preference: "en-US", candidate: "en-AU", want: 2},
		{name: "different base", preference: "en", candidate: "fr", want: -1},
		{name: "empty preference", candidate: "en", want: -1},
		{name: "empty candidate", preference: "en", want: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := languageMatchRank(tt.preference, tt.candidate); got != tt.want {
				t.Errorf("languageMatchRank() = %d, want %d", got, tt.want)
			}
		})
	}
}
