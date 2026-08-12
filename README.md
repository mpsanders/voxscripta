# YouTube Transcript Extractor

VoxScripta is a library-first Go toolkit for acquiring normalized, timestamped transcripts from public YouTube videos. A small CLI is included as a development, testing, and diagnostic harness over the same public API.

> **Status:** early development. The normalized domain model, validation, errors, provider contract, and YouTube URL/ID parsing are implemented. Caption parsing and `yt-dlp` acquisition are next. Public APIs may change before v1.

## Why this project exists

Applications that summarize, search, cite, or otherwise process video speech need more than a plain text blob. They need timestamps, language and source information, predictable selection rules, cancellation, and errors they can act on.

YouTube's unofficial extraction surfaces change frequently. Instead of embedding a fragile reimplementation, this project will initially use [`yt-dlp`](https://github.com/yt-dlp/yt-dlp) for caption discovery and retrieval, then normalize the result behind an idiomatic Go API. An optional speech-to-text provider can later cover videos that have no captions.

## Intended capabilities

- Accept common YouTube URLs or video IDs.
- Discover creator-provided and automatically generated captions.
- Select tracks deterministically using caller-supplied language preferences.
- Preserve ordered start/end timestamps and caption-source metadata.
- Return one provider-independent transcript model.
- Render plain text initially, with JSON, WebVTT, SRT, and Markdown planned.
- Respect context cancellation and deadlines.
- Expose useful errors for missing dependencies, invalid input, and unavailable transcripts.
- Permit custom acquisition providers without forcing them on ordinary users.

## Architecture

The public package owns domain types, options, orchestration, and stable behavior. Provider-specific details remain internal or behind narrow interfaces.

```text
Importing Go application                Development CLI
             |                                |
             +---------- public API ----------+
                              |
                    selection/orchestration
                              |
                       yt-dlp provider
                              |
              manual or automatic captions
                              |
                     parse and normalize
                              |
             timestamped Transcript result

Later, when explicitly configured:
no captions -> audio acquisition -> speech-to-text provider -> normalize
```

The CLI will remain a thin API consumer. Extraction logic will not live in `cmd/`.

## Proposed library usage

The module is `github.com/mpsanders/VoxScripta` and its public package name is `transcript`. Acquisition usage is expected to resemble the following once the default client is implemented:

```go
client, err := transcript.New(
	transcript.WithYTDLPPath("yt-dlp"),
)
if err != nil {
	log.Fatal(err)
}

result, err := client.Get(ctx, "https://www.youtube.com/watch?v=VIDEO_ID", transcript.Options{
	Languages: []string{"en-AU", "en"},
})
if err != nil {
	log.Fatal(err)
}

fmt.Print(result.Text())
```

The canonical result will retain segment timing rather than reducing the transcript to text immediately:

```go
type Segment struct {
	Start time.Duration
	End   time.Duration
	Text  string
}
```

Exact exported types will be finalized before implementation is declared stable.

## Proposed CLI usage

The development command is `ytextract`. Transcript acquisition flags are planned to support workflows like:

```console
ytextract --language en-AU --language en VIDEO_URL
ytextract --format json VIDEO_ID
ytextract --timeout 30s VIDEO_URL
```

Transcript data will be written to stdout and diagnostics to stderr so the command can be composed with other tools.

## Runtime dependencies

The first caption provider will require a compatible `yt-dlp` executable available on `PATH` or supplied explicitly in configuration.

The library will not silently install external tools. A future speech-to-text fallback may also require `ffmpeg`, a local model/runtime, or credentials for a remote service, but those dependencies will remain optional and explicit.

## Expected caption preference

The exact configurable policy will be validated during implementation. The working preference is:

1. manual captions matching the requested language;
2. manual captions matching an allowed language fallback;
3. automatic captions matching the requested language;
4. automatic captions matching an allowed language fallback;
5. translated captions only when enabled; and
6. speech-to-text only when explicitly configured and appropriate.

The result will report what was actually selected.

## Limitations and responsible use

- A transcript cannot be guaranteed for every video. Videos may be private, removed, restricted, inaccessible, silent, or unsupported by upstream tools.
- The project will not bypass DRM, authentication, geographic restrictions, or access controls.
- YouTube and `yt-dlp` can change independently; integration behavior therefore requires a documented compatibility and update policy.
- Callers are responsible for complying with applicable terms, copyright, privacy, and data-handling requirements.
- The official YouTube captions API is not a general solution for arbitrary public videos because downloading captions requires appropriate authorization over the video.

## Development

Development requires Go 1.25 or 1.26. The normalized core has no external runtime dependency. Caption acquisition will require a caller-installed `yt-dlp`; see [runtime dependencies](docs/DEPENDENCIES.md).

Once code exists, the baseline development checks will include:

```console
gofmt -w .
go test ./...
go vet ./...
```

Network-dependent `yt-dlp` tests will be opt-in so the normal unit-test suite remains fast and deterministic.

## Project direction

- [Goal](docs/planning/GOAL.md) describes the north star, scope, and completion criteria.
- [Roadmap](docs/planning/ROADMAP.md) defines implementation milestones and exit criteria.
- [TODO](docs/planning/TODO.md) is the actionable backlog.
- [Ideas](docs/planning/IDEAS.md) holds possible future work that is not yet committed.
- [Design conversation](docs/planning/YouTube%20Transcript%20Extraction%20Methods.md) records the initial exploration of extraction approaches.

## Contributing

The public API is intentionally not fixed yet. Early contributions should align with the goal and current roadmap, keep provider details isolated, include offline tests for deterministic logic, and update documentation when design decisions change.

## License

VoxScripta is available under the [MIT License](LICENSE).
