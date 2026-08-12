package transcript

import (
	"errors"
	"testing"
	"time"
)

func TestSegmentValidate(t *testing.T) {
	tests := []struct {
		name    string
		segment Segment
		wantErr bool
	}{
		{name: "valid", segment: Segment{Start: time.Second, End: 2 * time.Second, Text: "hello"}},
		{name: "zero duration", segment: Segment{Start: time.Second, End: time.Second, Text: "hello"}},
		{name: "zero start", segment: Segment{End: time.Second, Text: "hello"}},
		{name: "unicode", segment: Segment{End: time.Second, Text: "こんにちは 👋"}},
		{name: "negative start", segment: Segment{Start: -time.Second, End: time.Second, Text: "hello"}, wantErr: true},
		{name: "reversed time", segment: Segment{Start: 2 * time.Second, End: time.Second, Text: "hello"}, wantErr: true},
		{name: "empty text", segment: Segment{End: time.Second}, wantErr: true},
		{name: "whitespace text", segment: Segment{End: time.Second, Text: " \n\t"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.segment.Validate()
			if tt.wantErr && !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("Validate() error = %v, want ErrInvalidInput", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() unexpected error = %v", err)
			}
		})
	}
}

func TestTranscriptValidate(t *testing.T) {
	valid := Transcript{
		VideoID:  "dQw4w9WgXcQ",
		Language: Language{Code: "en"},
		Source:   SourceManual,
		Segments: []Segment{{End: time.Second, Text: "hello"}},
	}
	tests := []struct {
		name    string
		change  func(*Transcript)
		wantErr bool
	}{
		{name: "valid", change: func(*Transcript) {}},
		{name: "overlap allowed", change: func(got *Transcript) {
			got.Segments = []Segment{{End: 2 * time.Second, Text: "one"}, {Start: time.Second, End: 3 * time.Second, Text: "two"}}
		}},
		{name: "same start allowed", change: func(got *Transcript) { got.Segments = append(got.Segments, Segment{End: 2 * time.Second, Text: "two"}) }},
		{name: "invalid video ID", change: func(got *Transcript) { got.VideoID = "bad" }, wantErr: true},
		{name: "empty language", change: func(got *Transcript) { got.Language.Code = "" }, wantErr: true},
		{name: "zero source", change: func(got *Transcript) { got.Source = "" }, wantErr: true},
		{name: "unknown source", change: func(got *Transcript) { got.Source = "other" }, wantErr: true},
		{name: "nil segments", change: func(got *Transcript) { got.Segments = nil }, wantErr: true},
		{name: "out of order", change: func(got *Transcript) {
			got.Segments = []Segment{{Start: 2 * time.Second, End: 3 * time.Second, Text: "one"}, {Start: time.Second, End: 2 * time.Second, Text: "two"}}
		}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := valid
			got.Segments = append([]Segment(nil), valid.Segments...)
			tt.change(&got)
			err := got.Validate()
			if tt.wantErr && !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("Validate() error = %v, want ErrInvalidInput", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() unexpected error = %v", err)
			}
		})
	}
}

func TestTranscriptText(t *testing.T) {
	tests := []struct {
		name     string
		segments []Segment
		want     string
	}{
		{name: "nil", segments: nil, want: ""},
		{name: "empty", segments: []Segment{}, want: ""},
		{name: "one", segments: []Segment{{Text: "hello"}}, want: "hello"},
		{name: "trims edges", segments: []Segment{{Text: "  hello  "}}, want: "hello"},
		{name: "multiple", segments: []Segment{{Text: "hello"}, {Text: "world"}}, want: "hello\nworld"},
		{name: "skips blank", segments: []Segment{{Text: "hello"}, {Text: " \t"}, {Text: "world"}}, want: "hello\nworld"},
		{name: "preserves unicode", segments: []Segment{{Text: "café"}, {Text: "世界"}}, want: "café\n世界"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := (Transcript{Segments: tt.segments}).Text()
			if got != tt.want {
				t.Errorf("Text() = %q, want %q", got, tt.want)
			}
		})
	}
}
