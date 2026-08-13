<p align="center">
  <img src="docs/assets/voxscripta-logo.png" alt="VoxScripta logo" width="180">
</p>

<h1 align="center">VoxScripta</h1>

<p align="center"><strong>YouTube Transcript Extractor</strong></p>

VoxScripta is a library-first Go toolkit for acquiring normalized, timestamped transcripts from public YouTube videos. A small CLI is included as a development, testing, and diagnostic harness over the same public API.

> **Status:** caption-first development release. Caption discovery, selection,
> retrieval, WebVTT normalization, checked audio acquisition, and an opt-in
> local `whisper.cpp` adapter are implemented. The adapter has comprehensive
> offline tests and a recorded whisper.cpp 1.9.2 local-runtime evaluation.
> Public APIs may change before v1.

## Why this project exists

Applications that summarize, search, cite, or otherwise process video speech need more than a plain text blob. They need timestamps, language and source information, predictable selection rules, cancellation, and errors they can act on.

YouTube's unofficial extraction surfaces change frequently. Instead of embedding a fragile reimplementation, this project uses [`yt-dlp`](https://github.com/yt-dlp/yt-dlp) for caption discovery and retrieval, then normalizes the result behind an idiomatic Go API. An explicitly configured speech-to-text provider can cover videos that have no usable captions once a concrete transcriber is added.

## Intended capabilities

- Accept common YouTube URLs or video IDs.
- Discover creator-provided and automatically generated captions.
- Select tracks deterministically using caller-supplied language preferences.
- Preserve ordered start/end timestamps and caption-source metadata.
- Return one provider-independent transcript model.
- Render plain text through the library and structured JSON through the CLI; WebVTT, SRT, and Markdown renderers are possible later additions.
- Respect context cancellation and deadlines.
- Expose useful errors for missing dependencies, invalid input, and unavailable transcripts.
- Permit custom acquisition providers without forcing them on ordinary users.

Optional provider fallback is explicit. `transcript.FallbackProvider` invokes
its fallback only when the primary provider reports
`ErrTranscriptUnavailable`; it does not turn cancellation, invalid input,
missing dependencies, or provider failures into additional work. A future
transcriber adapter can use this composition without becoming a core runtime
dependency. The public `AudioSource` and `Transcriber` contracts are
separate, and `SpeechToTextProvider` composes them while enforcing optional
duration/file-size limits and closing acquired audio on every post-acquisition
path. Cleanup failures are returned. `YTDLPAudioSource` is the concrete audio
acquisition adapter. `WhisperCPPTranscriber` passes verified mono 16 kHz 16-bit
PCM WAV through unchanged and uses FFmpeg for other input before consuming
`whisper-cli` JSON output.

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

When explicitly configured by a library consumer:
no captions -> YTDLPAudioSource -> checked Audio -> Transcriber -> normalize
```

The CLI will remain a thin API consumer. Extraction logic will not live in `cmd/`.

## Library usage

The module is `github.com/mpsanders/VoxScripta` and its public package name is `transcript`. The default client uses `yt-dlp`:

```go
client, err := transcript.New(
	transcript.WithYTDLPPath("yt-dlp"),
)
if err != nil {
	log.Fatal(err)
}

result, err := client.Get(ctx, "https://www.youtube.com/watch?v=VIDEO_ID", transcript.Options{
	Languages:       []string{"en-AU", "en"},
	AllowAutomatic: true,
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

The returned transcript is owned by the caller. A client may be reused concurrently; custom providers must provide their own concurrency safety.

Provider process errors include at most 2 KiB of normalized stderr. URLs and
common credential-bearing values are redacted, and command arguments are never
included. Applications should still handle diagnostics as potentially
sensitive operational data rather than publishing them verbatim.

WebVTT data can already be parsed independently of a provider:

```go
segments, err := transcript.ParseWebVTT(reader)
if err != nil {
	log.Fatal(err)
}
```

## CLI usage

The development command is `ytextract`:

```console
ytextract --language en-AU --language en VIDEO_URL
ytextract --format json VIDEO_ID
ytextract --timeout 30s VIDEO_URL
ytextract --manual-only VIDEO_URL
ytextract --check
ytextract --check --yt-dlp /path/to/yt-dlp
ytextract --whisper-model /path/to/ggml-base.bin VIDEO_URL
```

Transcript data will be written to stdout and diagnostics to stderr so the command can be composed with other tools.

The CLI includes automatic captions by default. In the library, automatic captions are explicit: set `Options.AllowAutomatic` to `true`. Empty language preferences select the video's reported original language when possible, then deterministically fall back to the first eligible track. Translation is outside the caption-only API and may be added later behind a distinct interface.

CLI JSON uses human-readable Go duration strings such as `"0s"`, `"1.25s"`,
and `"2m3s"` for segment timestamps. It retains the transcript's video,
language, source, provider, and segment structure.

Supplying `--whisper-model` explicitly enables `captions -> local whisper.cpp`.
The `--whisper-cli`, `--ffmpeg`, `--max-audio-duration` (default 2h), and
`--max-audio-bytes` (default 200 MiB) flags configure that fallback. No model is
inferred or downloaded, and no hosted provider is enabled from ambient
credentials.

`--check` verifies that the configured `yt-dlp` executable starts and reports
its version. It performs no video or network acquisition.

## Runtime dependencies

The caption provider requires a compatible `yt-dlp` executable available on `PATH` or supplied explicitly in configuration.

The library will not silently install external tools. Speech-to-text audio
acquisition uses the same caller-installed `yt-dlp`. The optional whisper.cpp
adapter requires `whisper-cli`, a caller-selected GGML model, and FFmpeg unless
the input is verified as compatible PCM WAV.

The speech-to-text composition API is transcriber-neutral. Construct a
`YTDLPAudioSource` with `NewYTDLPAudioSource`, then configure it on
`SpeechToTextProvider`. A zero limit disables that limit; negative limits are
invalid. With a positive duration limit, unknown-duration and live inputs are
rejected before download. `MaxBytes` asks `yt-dlp` to reject known oversized
downloads and strictly rejects an oversized completed artifact, but upstream
manifest/fragment downloads are not hard-bounded while in flight. `Audio.Format`
is a lower-case container/file-extension hint, not a codec guarantee. Direct
`YTDLPAudioSource.Acquire` callers must close `Audio.Data` promptly to remove
the temporary artifact. `SpeechToTextProvider` closes it automatically. Cost
and concurrency controls will be designed with the first concrete adapter rather
than represented by misleading generic fields.

The current public composition is explicit: create the ordinary caption client,
use it as `FallbackProvider.Primary`, use a configured `SpeechToTextProvider`
as `Fallback`, and install that chain in an outer client. The fallback runs only
for `ErrTranscriptUnavailable`. A compile-tested offline example is included;
the CLI enables the local chain only when `--whisper-model` explicitly selects
it.

```go
captions, err := transcript.New(transcript.WithYTDLPPath("yt-dlp"))
if err != nil {
	return err
}
speech := transcript.SpeechToTextProvider{
	AudioSource: transcript.NewYTDLPAudioSource("yt-dlp"),
	Transcriber: myTranscriber,
	MaxDuration: 2 * time.Hour,
	MaxBytes:    200 << 20,
}
client, err := transcript.New(transcript.WithProvider(transcript.FallbackProvider{
	Primary: captions,
	Fallback: speech,
}))
```

## Expected caption preference

For each requested language in caller order, selection tries an exact tag,
then its base tag, then another regional variant. Manual captions beat automatic
captions only within the same preference and match rank. Automatic captions
require explicit library permission and are enabled by default in the CLI.
Speech-to-text runs only through an explicitly configured fallback provider.

The result will report what was actually selected.

## Limitations and responsible use

- A transcript cannot be guaranteed for every video. Videos may be private, removed, restricted, inaccessible, silent, or unsupported by upstream tools.
- The project will not bypass DRM, authentication, geographic restrictions, or access controls.
- YouTube and `yt-dlp` can change independently; integration behavior therefore requires a documented compatibility and update policy.
- Callers are responsible for complying with applicable terms, copyright, privacy, and data-handling requirements.
- The official YouTube captions API is not a general solution for arbitrary public videos because downloading captions requires appropriate authorization over the video.

## Development

Development requires Go 1.25 or 1.26. The normalized core has no external runtime dependency. Caption and audio acquisition require a caller-installed `yt-dlp`; see [runtime dependencies](docs/DEPENDENCIES.md).

The repository Makefile provides the common development commands:

```console
make build
make test
make integration
make vet
make staticcheck
make vuln
make hardening
make run ARGS="--version"
make check
```

Run `make help` for the complete target list, including formatting, race testing,
module tidying, and cleanup. `make check` performs the deterministic core checks
expected before submitting a change. `make hardening` additionally runs the race
detector, Staticcheck, and the network-backed Go vulnerability scan; install the
pinned tool versions documented in [CONTRIBUTING.md](CONTRIBUTING.md) first.

Network-dependent `yt-dlp` tests are opt-in so the normal unit-test suite remains deterministic. The ordinary suite includes process-level CLI smoke tests backed by a temporary fake provider executable. Run the live tests explicitly with `make integration`; this requires `yt-dlp` on `PATH` and public network access. `make test` continues to skip the live suite.

The local whisper.cpp runtime test is separately opt-in and requires explicit
executable, model, and compatible WAV sample paths through the environment
variables documented in `docs/DEPENDENCIES.md`. The recorded prototype evidence
is in `docs/evaluations/whispercpp-2026-08-13.md`.

## Project direction

- [Goal](docs/planning/GOAL.md) describes the north star, scope, and completion criteria.
- [Roadmap](docs/planning/ROADMAP.md) defines implementation milestones and exit criteria.
- [TODO](docs/planning/TODO.md) is the actionable backlog.
- [Ideas](docs/planning/IDEAS.md) holds possible future work that is not yet committed.
- [Live test-video matrix](docs/planning/TEST_VIDEOS.md) records proposed integration fixtures, provenance, validation, and replacement policy.
- [Design conversation](docs/planning/YouTube%20Transcript%20Extraction%20Methods.md) records the initial exploration of extraction approaches.
- [Responsible-use guidance](docs/RESPONSIBLE_USE.md) covers privacy, rights,
  restricted content, rate limits, retention, and caption accuracy.
- [Release checklist](docs/RELEASING.md) and [changelog](CHANGELOG.md) describe
  the pre-release verification and publication process.

## Contributing

The public API is intentionally not fixed yet. Early contributions should align with the goal and current roadmap, keep provider details isolated, include offline tests for deterministic logic, and update documentation when design decisions change.

## License

VoxScripta is available under the [MIT License](LICENSE).
