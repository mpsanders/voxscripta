# Roadmap

This roadmap is ordered by dependency and learning value. Dates are deliberately omitted; a milestone advances when its exit criteria are met.

## Milestone 0: Foundation and decisions

Status: complete. Module identity, supported Go/platform policy, baseline
layout, CI, license, dependency guidance, and initial diagrams are complete.
The public test-video matrix and sanitized observed fixtures are now recorded.

Define the module and establish the contracts before committing to implementation details.

Deliverables:

- Initialize the Go module and baseline repository layout.
- Record supported Go and `yt-dlp` versions and the initial platform matrix.
- Decide the import path, package name, CLI name, and semantic-versioning policy.
- Validate `yt-dlp` metadata and subtitle output using a small, legal test-video matrix.
- Write architecture and acquisition sequence diagrams under `docs/uml`.
- Convert important design choices into short decision records where useful.

Exit criteria:

- A minimal Go project builds and tests on supported platforms.
- The first caption format and `yt-dlp` command contract are documented from observed output.
- Public domain types and errors have an agreed draft.

## Milestone 1: Normalized transcript core

Status: complete. The domain model, validation rules, errors, provider contract,
URL/ID parsing, WebVTT parsing and normalization, rolling-caption de-duplication,
and plain-text rendering are implemented and tested offline.

Build the provider-independent model and parsing foundation entirely offline.

Deliverables:

- Define `Transcript`, `Segment`, language/source metadata, and validation rules.
- Implement video ID and supported YouTube URL parsing.
- Implement the initial subtitle parser, likely WebVTT, using fixture-based tests.
- Normalize timing, whitespace, duplicate rolling-caption cues, and ordering according to documented rules.
- Add plain-text rendering without discarding the timestamped source model.
- Define typed/sentinel errors and provider interfaces.

Exit criteria:

- Core tests require neither network access nor external executables.
- Malformed, empty, overlapping, duplicated, and Unicode subtitle fixtures have defined behavior.
- The model is useful without knowledge of `yt-dlp`.

## Milestone 2: `yt-dlp` caption provider

Status: complete. Direct context-aware command execution, version probing,
narrow metadata decoding, internal manual/automatic track modeling,
deterministic language/source selection, and isolated single-track WebVTT
retrieval with guaranteed temporary-file cleanup are implemented and tested
offline. Public provider orchestration connects these pieces through parsing
and normalized results. Offline tests cover malformed metadata, missing
output, process failures, timeouts, cancellation, and cleanup. Subprocess
failures retain bounded, redacted stderr without exposing command arguments.
The live matrix validates manual, automatic, multilingual, unavailable, and
cancellation behavior with `yt-dlp 2026.07.04`.

Connect the normalized core to the primary acquisition mechanism.

Deliverables:

- Add an injectable command runner around `exec.CommandContext`.
- Validate executable availability and capture the installed `yt-dlp` version.
- Inspect videos using machine-readable JSON rather than human-oriented stdout.
- Model available manual and automatic caption tracks.
- Implement deterministic language and source selection.
- Retrieve only the selected caption track into isolated temporary storage or through a validated direct download path.
- Parse and normalize the result, preserving diagnostic causes and cleaning up temporary files.

Exit criteria:

- Unit tests mock process execution and cover success and failure paths.
- Opt-in integration tests cover manual captions, automatic captions, no captions, invalid videos, and cancellation.
- Missing or incompatible `yt-dlp` produces an actionable error.

## Milestone 3: Stable library API and development CLI

Status: complete. A default constructor, executable and custom-provider
options, validated acquisition, and a thin CLI with language, source, timeout,
text/JSON, version, and stable error exits are implemented. Full process-level
CLI smoke coverage verifies text and JSON output plus stable failure exits
without network access. A CLI-only dependency diagnostic checks the configured
yt-dlp version without expanding the public API. The pre-v1 API review is
recorded in decision 0005. Compile-tested examples cover basic use, language
selection, errors, custom providers, and standalone WebVTT parsing.

Make the library pleasant to consume and use the CLI to prove that boundary.

Deliverables:

- Finalize constructors, options, defaults, provider injection, and GoDoc.
- Add a thin CLI under `cmd/<name>` that imports the public package.
- Support URL/ID input, language preferences, source policy, timeout, and output format.
- Write transcript data to stdout and diagnostics to stderr; define useful exit codes.
- Add JSON output for automated inspection and plain text for humans.
- Add examples and CLI smoke tests.

Exit criteria:

- A separate example program can import the module without internal packages.
- The CLI contains argument handling and presentation only.
- API and CLI behavior are documented in the README.

## Milestone 4: Hardening and first release

Status: locally tagged but not published. Cross-platform CI, formatting, vet, deterministic
tests, race coverage, pinned Staticcheck, and pinned govulncheck gates are
established. Responsible-use and upstream-breakage guidance, a changelog, and
a release checklist are documented. Local release-hardening checks passed on
2026-08-13. The annotated local `v0.1.0` tag points to the caption/fallback
commit, but the configured GitLab remote had no tag on 2026-08-13. Publishing
and clean-install validation remain maintainer actions. Current speech/audio
work is post-tag and recorded under `Unreleased`.

Prepare the caption-only implementation for dependable reuse.

Deliverables:

- Test Windows, Linux, and macOS in CI.
- Add race, vet, formatting, vulnerability, and static-analysis checks as appropriate.
- Document concurrency safety, timeouts, resource limits, and privacy implications.
- Define a fixture-refresh process that avoids depending on live YouTube in normal tests.
- Add provider diagnostics that do not leak cookies, signed URLs, or other sensitive values.
- Publish migration and release guidance and tag the initial version.

Exit criteria:

- CI is repeatable and ordinary unit tests are deterministic.
- Public types and functions have GoDoc and examples.
- Known limitations and dependency requirements are explicit.

## Milestone 5: Optional speech-to-text fallback

Status: in progress. The provider-level fallback policy is implemented and
requires explicit composition. Separate audio-source and transcriber contracts
now compose through `SpeechToTextProvider`, which checks configured duration
and file-size limits, closes audio after transcription, and surfaces closure
failures. The concrete `YTDLPAudioSource` now rejects unknown/live or known over-limit
durations before downloading, asks `yt-dlp` to reject known oversized files,
strictly verifies the resulting file, and owns isolated temporary artifacts
until the audio stream is closed. An opt-in `WhisperCPPTranscriber` now stages
audio, verifies compatible PCM WAV or conditionally converts with FFmpeg,
normalizes JSON segments, bounds diagnostics, honors cancellation, and cleans
its workspace. The CLI composes captions then local speech when a model is
explicitly supplied. Offline behavior is tested, but live whisper.cpp model
evaluation, the hosted prototype, richer provenance, completeness policy, and
full process-level fallback tests remain pending.

Extend coverage without making heavy dependencies part of the core caption path.

Implementation sequence:

1. **Checked `yt-dlp` audio acquisition.** Implement an audio source that
   inspects duration before download, retrieves audio into isolated temporary
   storage, constrains output size, honors cancellation, and removes all
   temporary artifacts on success and failure.
2. **Concrete adapter prototypes.** Prototype local `whisper.cpp` and one
   hosted transcription service (initially OpenAI unless evaluation identifies
   a better reference provider). Record accepted formats, path/stream needs,
   timestamps, limits, cancellation behavior, privacy, portability, accuracy,
   and cost without treating either prototype as production-ready.
3. **Audio-processing boundary decision.** Use prototype evidence to decide
   whether FFmpeg conversion belongs inside the `AudioSource` implementation or
   behind a separate audio-processor contract. Invoke FFmpeg only when the
   selected transcriber requires conversion; pass compatible downloaded audio
   through unchanged.
4. **Production adapter and safeguards.** Implement at least one separately
   configured transcriber, normalize its segments, and add adapter-aware cost
   and concurrency controls plus safe bounded diagnostics.
5. **End-to-end fallback behavior.** Compose the audio/STT provider with
   `FallbackProvider`, decide the partial-result policy, and test no-caption
   fallback, failures, cancellation, resource limits, and cleanup across every
   stage.
6. **Completeness and CLI policy.** Define measurable caption completeness and
   whether incomplete tracks are accepted, replaced, compared, or merged with
   speech-to-text. Expose deterministic caller-selected ordering in the library.
   Make the CLI try every locally installed and explicitly enabled strategy,
   preferring captions and local transcription before hosted services. A hosted
   provider is available only through explicit CLI configuration; ambient
   credentials alone must not trigger cost or audio disclosure.

Completed foundations:

- Define separate audio-source and speech-to-text contracts.
- Define explicit fallback policy, language hints, and provider-level
  duration/file-size guards.
- Normalize valid speech-to-text segments into the same `Transcript` model.
- Acquire the best available audio-only media with duration preflight, best-effort
  transfer-size rejection, strict final-size verification, offline fake-process
  coverage, and an opt-in live suite validated with `yt-dlp 2026.07.04` on
  2026-08-13.
- Implement an explicitly configured local whisper.cpp adapter with verified
  compatible-WAV passthrough, conditional FFmpeg conversion, normalized JSON
  timestamps, bounded path-redacted diagnostics, and workspace cleanup.
- Compose caption-first local speech fallback in the CLI only when a model is
  explicitly selected, with two-hour and 200 MiB default audio guards.

Exit criteria:

- Caption retrieval remains the default and has no new mandatory runtime dependency.
- Callers must explicitly configure or opt into potentially costly speech-to-text work.
- `yt-dlp` audio acquisition rejects configured duration and final-size limits,
  documents its in-flight size limitation, and surfaces cleanup failures.
- FFmpeg is required only by configurations whose selected adapter needs it.
- At least one transcriber adapter has documented compatibility and safeguards.
- Tests cover fallback ordering, cancellation, limits, partial output, and cleanup.

## Milestone 6: Ecosystem features

Add only features justified by real consumers.

Candidates include:

- SRT, WebVTT, Markdown, and richer JSON renderers.
- Cache interfaces with expiry and provider-version awareness.
- Batch APIs with bounded concurrency and per-item results.
- Custom HTTP clients, proxy/cookie configuration, and rate-limit hooks.
- Metrics, tracing, structured diagnostic events, and provider health checks.
- Additional providers for managed-channel captions or transcription services.
