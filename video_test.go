package transcript

import (
	"errors"
	"testing"
)

func TestParseVideoID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "raw ID", input: "dQw4w9WgXcQ", want: "dQw4w9WgXcQ"},
		{name: "watch URL", input: "https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=12", want: "dQw4w9WgXcQ"},
		{name: "short URL", input: "https://youtu.be/dQw4w9WgXcQ", want: "dQw4w9WgXcQ"},
		{name: "shorts URL", input: "https://youtube.com/shorts/dQw4w9WgXcQ", want: "dQw4w9WgXcQ"},
		{name: "embed no-cookie URL", input: "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ", want: "dQw4w9WgXcQ"},
		{name: "live URL", input: "https://m.youtube.com/live/dQw4w9WgXcQ", want: "dQw4w9WgXcQ"},
		{name: "surrounding whitespace", input: "  dQw4w9WgXcQ\n", want: "dQw4w9WgXcQ"},
		{name: "empty", input: "", wantErr: true},
		{name: "short ID", input: "abc", wantErr: true},
		{name: "lookalike host", input: "https://youtube.com.example/watch?v=dQw4w9WgXcQ", wantErr: true},
		{name: "userinfo", input: "https://youtube.com@example.com/watch?v=dQw4w9WgXcQ", wantErr: true},
		{name: "unsupported scheme", input: "ftp://youtube.com/watch?v=dQw4w9WgXcQ", wantErr: true},
		{name: "extra short path", input: "https://youtu.be/dQw4w9WgXcQ/more", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVideoID(tt.input)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("ParseVideoID() error = %v, want ErrInvalidInput", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseVideoID() unexpected error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseVideoID() = %q, want %q", got, tt.want)
			}
		})
	}
}
