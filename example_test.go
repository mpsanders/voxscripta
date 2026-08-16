package transcript_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	transcript "github.com/mpsanders/voxscripta"
)

type exampleProvider struct{}

type unavailableExampleProvider struct{}

type exampleAudioSource struct{}

type exampleTranscriber struct{}

// Get reports that no transcript is available so the fallback example remains
// deterministic and offline.
func (unavailableExampleProvider) Get(context.Context, string, transcript.Options) (transcript.Transcript, error) {
	return transcript.Transcript{}, transcript.ErrTranscriptUnavailable
}

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

// Acquire returns deterministic in-memory audio for the fallback composition
// example; real consumers can use transcript.NewYTDLPAudioSource instead.
func (exampleAudioSource) Acquire(context.Context, string, transcript.AudioOptions) (transcript.Audio, error) {
	return transcript.Audio{Data: io.NopCloser(strings.NewReader("audio")), Format: "wav", Duration: time.Second, Size: 5}, nil
}

// Transcribe returns a deterministic speech-to-text result for the fallback
// composition example.
func (exampleTranscriber) Transcribe(context.Context, transcript.Audio, []string) (transcript.Transcription, error) {
	return transcript.Transcription{
		Language: transcript.Language{Code: "en"}, Provider: transcript.ProviderMetadata{Name: "example-stt"},
		Segments: []transcript.Segment{{Start: 0, End: time.Second, Text: "Spoken fallback."}},
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

// ExampleFallbackProvider demonstrates explicitly opting into a secondary
// provider for videos whose primary provider has no transcript.
func ExampleFallbackProvider() {
	provider := transcript.FallbackProvider{
		Primary:  unavailableExampleProvider{},
		Fallback: exampleProvider{},
	}
	client, _ := transcript.New(transcript.WithProvider(provider))
	result, _ := client.Get(context.Background(), "dQw4w9WgXcQ", transcript.Options{})
	fmt.Println(result.Text())
	// Output: Hello, world.
}

// ExampleSpeechToTextProvider demonstrates the implemented caption-first
// composition without requiring network access or a concrete transcriber.
func ExampleSpeechToTextProvider() {
	primary, _ := transcript.New(transcript.WithProvider(unavailableExampleProvider{}))
	speech := transcript.SpeechToTextProvider{
		AudioSource: exampleAudioSource{}, Transcriber: exampleTranscriber{},
		MaxDuration: time.Minute, MaxBytes: 1024,
	}
	client, _ := transcript.New(transcript.WithProvider(transcript.FallbackProvider{
		Primary: primary, Fallback: speech,
	}))
	result, _ := client.Get(context.Background(), "dQw4w9WgXcQ", transcript.Options{Languages: []string{"en"}})
	fmt.Println(result.Source, result.Text())
	// Output: speech_to_text Spoken fallback.
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
