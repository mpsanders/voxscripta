package transcript

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseWebVTTFixtures(t *testing.T) {
	tests := []struct {
		name      string
		fixture   string
		want      []Segment
		wantError error
	}{
		{
			name:    "basic cues",
			fixture: "testdata/vtt/basic.vtt",
			want: []Segment{
				{Start: time.Second, End: 2500 * time.Millisecond, Text: "Hello world"},
				{Start: 3 * time.Second, End: 4 * time.Second, Text: "Second cue"},
			},
		},
		{
			name:    "unicode entities and markup",
			fixture: "testdata/vtt/unicode.vtt",
			want: []Segment{
				{End: time.Second, Text: "café & tea"},
				{Start: time.Second, End: 2 * time.Second, Text: "こんにちは 👋"},
			},
		},
		{
			name:    "rolling automatic captions",
			fixture: "testdata/vtt/rolling.vtt",
			want: []Segment{
				{End: 2 * time.Second, Text: "one two"},
				{Start: time.Second, End: 3 * time.Second, Text: "three"},
				{Start: 2500 * time.Millisecond, End: 4 * time.Second, Text: "four"},
			},
		},
		{
			name:    "out of order cues are sorted",
			fixture: "testdata/vtt/out_of_order.vtt",
			want: []Segment{
				{Start: time.Second, End: 2 * time.Second, Text: "first"},
				{Start: 3 * time.Second, End: 4 * time.Second, Text: "second"},
			},
		},
		{name: "malformed timing", fixture: "testdata/vtt/malformed.vtt", wantError: ErrInvalidInput},
		{name: "empty cues", fixture: "testdata/vtt/empty.vtt", wantError: ErrTranscriptUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := os.Open(tt.fixture)
			if err != nil {
				t.Fatalf("open fixture: %v", err)
			}
			defer file.Close()

			got, err := ParseWebVTT(file)
			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Fatalf("ParseWebVTT() error = %v, want %v", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseWebVTT() unexpected error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParseWebVTT() returned %d segments, want %d: %#v", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("segment %d = %#v, want %#v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseWebVTTInputErrors(t *testing.T) {
	tests := []struct {
		name      string
		reader    *strings.Reader
		useNil    bool
		wantError error
	}{
		{name: "nil reader", useNil: true, wantError: ErrInvalidInput},
		{name: "empty input", reader: strings.NewReader(""), wantError: ErrUnsupportedFormat},
		{name: "wrong signature", reader: strings.NewReader("SRT\n"), wantError: ErrUnsupportedFormat},
		{name: "reversed cue", reader: strings.NewReader("WEBVTT\n\n00:02.000 --> 00:01.000\ntext\n"), wantError: ErrInvalidInput},
		{name: "invalid minutes", reader: strings.NewReader("WEBVTT\n\n60:00.000 --> 61:00.000\ntext\n"), wantError: ErrInvalidInput},
		{name: "missing timing", reader: strings.NewReader("WEBVTT\n\nidentifier\ntext\n"), wantError: ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.useNil {
				_, err := ParseWebVTT(nil)
				if !errors.Is(err, tt.wantError) {
					t.Fatalf("ParseWebVTT() error = %v, want %v", err, tt.wantError)
				}
				return
			}
			_, err := ParseWebVTT(tt.reader)
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("ParseWebVTT() error = %v, want %v", err, tt.wantError)
			}
		})
	}
}
