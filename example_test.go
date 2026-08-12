package transcript_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	transcript "github.com/mpsanders/VoxScripta"
)

type exampleProvider struct{}

// Get returns a deterministic transcript so examples remain offline and show
// how an application can supply its own acquisition provider.
func (exampleProvider) Get(_ context.Context, videoID string, options transcript.Options) (transcript.Transcript, error) {
	language := "en"
	if len(options.Languages) > 0 {
		language = options.Languages[0]
	}
	return transcript.Transcript{
		VideoID:  videoID,
		Language: transcript.Language{Code: language},
		Source:   transcript.SourceManual,
		Provider: transcript.ProviderMetadata{Name: "example"},
		Segments: []transcript.Segment{{Start: 0, End: time.Second, Text: "Hello, world."}},
	}, nil
}

// ExampleClient_Get demonstrates basic transcript acquisition through the
// public Client API using an offline provider.
func ExampleClient_Get() {
	client, _ := transcript.New(transcript.WithProvider(exampleProvider{}))
	result, err := client.Get(context.Background(), "dQw4w9WgXcQ", transcript.Options{})
	if err != nil {
		panic(err)
	}
	fmt.Println(result.Text())
	// Output: Hello, world.
}

// ExampleClient_Get_languagePreferences demonstrates ordered caller language
// preferences and explicit automatic-caption permission.
func ExampleClient_Get_languagePreferences() {
	client, _ := transcript.New(transcript.WithProvider(exampleProvider{}))
	result, _ := client.Get(context.Background(), "dQw4w9WgXcQ", transcript.Options{
		Languages:      []string{"en-AU", "en"},
		AllowAutomatic: true,
	})
	fmt.Println(result.Language.Code)
	// Output: en-AU
}

// ExampleClient_Get_errors demonstrates classification of stable public error
// categories with errors.Is.
func ExampleClient_Get_errors() {
	client, _ := transcript.New(transcript.WithProvider(exampleProvider{}))
	_, err := client.Get(context.Background(), "invalid", transcript.Options{})
	fmt.Println(errors.Is(err, transcript.ErrInvalidInput))
	// Output: true
}

// ExampleWithProvider demonstrates constructing a client with an application-
// supplied Provider instead of the default yt-dlp implementation.
func ExampleWithProvider() {
	client, err := transcript.New(transcript.WithProvider(exampleProvider{}))
	fmt.Println(err == nil, client != nil)
	// Output: true true
}

// ExampleParseWebVTT demonstrates parsing subtitle data independently of any
// acquisition provider.
func ExampleParseWebVTT() {
	segments, err := transcript.ParseWebVTT(strings.NewReader("WEBVTT\n\n00:00.000 --> 00:01.000\nHello\n"))
	if err != nil {
		panic(err)
	}
	fmt.Println(segments[0].Start, segments[0].Text)
	// Output: 0s Hello
}
