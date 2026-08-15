# Project Goal

## North star

Build a dependable, library-first Go toolkit that applications can import to
obtain a normalized, timestamped transcript from any accessible YouTube video
with usable captions or speech, on a best-effort basis.

The project makes the common case simple while keeping YouTube-specific volatility and external tooling behind clear boundaries. A small command-line program exercises the public API during development and provides a useful diagnostic tool, but it is not the primary product.

## What success looks like

An importing application can:

- pass a YouTube video URL or ID and a `context.Context`;
- express language and caption-source preferences;
- receive a stable `Transcript` containing metadata and ordered timestamped segments;
- distinguish manual captions, automatic captions, and speech-to-text output;
- handle predictable, inspectable errors;
- replace or add acquisition providers without changing downstream code; and
- render the normalized result as plain text or common subtitle formats.

The default implementation uses `yt-dlp` to discover and retrieve existing captions. This delegates the frequently changing YouTube integration to a mature, actively maintained tool. Videos without usable captions can be handled by explicitly configured fallback: `yt-dlp` acquires checked audio, the implemented local `whisper.cpp` adapter uses FFmpeg only when its input requires conversion, and replaceable transcribers produce normalized segments. The current CLI tries captions and then local `whisper.cpp` only when a model is explicitly supplied; future providers must preserve explicit ordering without silently spending money or disclosing audio through ambient credentials.

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
- Parsing and normalization of WebVTT captions.
- Timestamped transcript segments and plain-text rendering.
- A thin development CLI.
- Cross-platform support where Go and `yt-dlp` are available, beginning with Windows, Linux, and macOS.

### Implemented optional scope

- Explicit local speech-to-text fallback for videos without usable captions.
- Checked audio acquisition through `yt-dlp` and conversion through FFmpeg only
  when required by the configured `whisper.cpp` adapter.
- Transcription kept separate from YouTube-specific acquisition so callers can
  replace either side without changing downstream transcript handling.

### Later scope

- Hosted transcription adapters with explicit privacy, latency, and cost controls.
- Additional renderers such as WebVTT, SRT, Markdown, and JSON.
- Provider diagnostics, caching hooks, and observability.
- Batch acquisition with caller-controlled concurrency.

## Non-goals

- Reimplementing YouTube's private player, Innertube, signature, or caption APIs.
- Circumventing authentication, access controls, DRM, geographic restrictions, or anti-bot protections.
- Guaranteeing a transcript for private, DRM-protected, inaccessible, silent,
  corrupted, or otherwise unsupported media.
- Bundling or auto-installing `yt-dlp`, `ffmpeg`, or speech-to-text models in the core library.
- Providing a hosted transcription service, media player, or general-purpose
  downloader/`yt-dlp` wrapper; narrow temporary audio acquisition exists only
  to support transcription.
- Making the official YouTube Data API the default path for arbitrary public videos; caption downloads there are suited to content the authenticated user can manage.

## Current public API direction

Names may still change before v1, but the implemented shape is intentionally small:

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

The normalized domain model retains:

- video ID and, when available, title;
- selected language and source kind; a future translation interface must also
  identify source and target languages;
- source kind (manual, automatic, or speech-to-text);
- ordered segments with start time, end time, and text; and
- provider metadata useful for diagnosis without coupling callers to raw provider output.

Advanced users can construct an orchestrator with their own providers, while the default constructor remains straightforward.

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
