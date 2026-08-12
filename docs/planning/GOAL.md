# Project Goal

## North star

Build a dependable, library-first Go toolkit that applications can import to obtain a normalized, timestamped transcript for a public YouTube video.

The project should make the common case simple while keeping YouTube-specific volatility and external tooling behind clear boundaries. A small command-line program will exercise the public API during development and provide a useful diagnostic tool, but it is not the primary product.

## What success looks like

An importing application can:

- pass a YouTube video URL or ID and a `context.Context`;
- express language and caption-source preferences;
- receive a stable `Transcript` containing metadata and ordered timestamped segments;
- distinguish manual captions, automatic captions, and speech-to-text output;
- handle predictable, inspectable errors;
- replace or add acquisition providers without changing downstream code; and
- render the normalized result as plain text or common subtitle formats.

The default implementation should use `yt-dlp` to discover and retrieve existing captions. This delegates the frequently changing YouTube integration to a mature, actively maintained tool. Videos without captions can later be handled by explicitly configured fallback: `yt-dlp` acquires bounded audio, FFmpeg converts it only when required by the selected backend, and a replaceable local or hosted transcriber produces normalized segments.

## Product principles

1. **Library first.** Export a small, idiomatic Go API. The CLI must call that API rather than contain extraction logic.
2. **Normalized output.** Preserve timestamps and source metadata in one representation regardless of the provider or input subtitle format.
3. **Explicit dependencies.** Detect and report missing or incompatible executables clearly. Do not silently install software or require Python packages from library consumers.
4. **Context throughout.** All network and subprocess work must support cancellation and caller-defined deadlines.
5. **Deterministic selection.** Caption choice must follow documented language and source preferences, with the selected track described in the result.
6. **Provider isolation.** Keep command execution, YouTube discovery, parsing, selection, and orchestration separable and testable.
7. **Actionable failure.** Use sentinel or typed errors that support `errors.Is`/`errors.As`, while retaining diagnostic causes.
8. **Secure subprocess use.** Pass arguments without invoking a shell, use isolated temporary storage, constrain outputs, and clean up resources.
9. **Stable public surface.** Avoid exposing raw `yt-dlp` JSON or command details as the primary API.
10. **Evidence over promises.** Test parsers and selection logic with fixtures, and gate releases with unit, integration, and CLI smoke tests.

## Intended scope

### Initial scope

- Public YouTube video URLs and video IDs.
- Discovery and retrieval of creator-provided and automatically generated captions through `yt-dlp`.
- Language preference and deterministic track selection.
- Parsing and normalization of one reliable machine-readable subtitle format, with WebVTT as the likely first format.
- Timestamped transcript segments and plain-text rendering.
- A thin development CLI.
- Cross-platform support where Go and `yt-dlp` are available, beginning with Windows, Linux, and macOS.

### Later scope

- Optional speech-to-text providers for videos without captions.
- Audio acquisition through `yt-dlp` and conversion through `ffmpeg` where required.
- Local and hosted transcription adapters kept separate from YouTube-specific
  acquisition so callers can choose privacy, portability, latency, and cost
  tradeoffs without changing downstream transcript handling.
- Additional renderers such as WebVTT, SRT, Markdown, and JSON.
- Provider diagnostics, caching hooks, and observability.
- Batch acquisition with caller-controlled concurrency.

## Non-goals

- Reimplementing YouTube's private player, Innertube, signature, or caption APIs.
- Circumventing authentication, access controls, DRM, geographic restrictions, or anti-bot protections.
- Guaranteeing a transcript for every video.
- Bundling or auto-installing `yt-dlp`, `ffmpeg`, or speech-to-text models in the core library.
- Providing a hosted transcription service, downloader, media player, or general-purpose `yt-dlp` wrapper.
- Making the official YouTube Data API the default path for arbitrary public videos; caption downloads there are suited to content the authenticated user can manage.

## Proposed public API direction

Names may change during the API-design milestone, but the desired shape is intentionally small:

```go
client, err := transcript.New(
	transcript.WithYTDLPPath("yt-dlp"),
)
if err != nil {
	return err
}

result, err := client.Get(ctx, videoURL, transcript.Options{
	Languages: []string{"en-AU", "en"},
})
if err != nil {
	return err
}

fmt.Println(result.Text())
```

The normalized domain model should retain at least:

- video ID and, when available, title;
- selected language and source kind; a future translation interface must also
  identify source and target languages;
- source kind (manual, automatic, or speech-to-text);
- ordered segments with start time, end time, and text; and
- provider metadata useful for diagnosis without coupling callers to raw provider output.

Advanced users should be able to construct an orchestrator with their own providers, while the default constructor remains straightforward.

## Completion criteria

The first stable release is ready when:

- the public API has documentation and compatibility expectations;
- manual and automatic captions can be selected and normalized reliably;
- cancellation, missing dependencies, malformed input, unavailable captions, and provider failures have tested behavior;
- the CLI demonstrates the public API and supports useful structured output;
- fixtures cover caption parsing without requiring network access;
- opt-in integration tests validate supported `yt-dlp` behavior;
- supported platforms are exercised in continuous integration; and
- the README, planning documents, examples, and architecture diagrams match the implementation.
