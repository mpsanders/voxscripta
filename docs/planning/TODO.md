# TODO

This is the actionable project backlog. Check an item only when its implementation, tests, and relevant documentation are complete.

## Now: establish the project

- [x] Choose the Go module/import path and initialize `go.mod`.
- [x] Choose the exported package name and the development CLI name.
- [x] Set the minimum supported Go version.
- [x] Create the initial layout: public package, focused `internal` packages, `cmd/<name>`, fixtures, examples, and `docs/uml`.
- [x] Add a license and contribution guidance.
- [x] Add CI for build, test, `go vet`, formatting checks, and the supported OS matrix.
- [x] Add a Makefile for the common build, run, test, vet, formatting, and module checks.
- [x] Document how contributors install and verify `yt-dlp` without having the library install it.
- [x] Validate and accept the public test-video matrix in `TEST_VIDEOS.md` with `yt-dlp 2026.07.04`.
- [x] Capture representative, sanitized `yt-dlp` JSON fixtures and WebVTT structures for offline tests.
- [x] Resolve the repository convention mismatch between the current `planning/` directory and the `docs/planning` path described in `agents.md`.

## Core model and parsing

- [x] Define `Transcript`, `Segment`, source-kind, language, and provider metadata types.
- [x] Specify invariants for segment ordering, zero/negative timestamps, empty text, and overlapping cues.
- [x] Implement YouTube video ID and URL parsing for documented URL forms.
- [x] Define `Provider` and orchestration contracts with `context.Context`.
- [x] Define errors for invalid input, missing dependency, unavailable transcript, unsupported format, cancellation, and provider failure.
- [x] Implement the first subtitle parser; an established library was evaluated and rejected as disproportionate for the focused WebVTT contract.
- [x] Normalize HTML entities, Unicode, line breaks, and whitespace.
- [x] Define and implement de-duplication for YouTube's rolling automatic-caption cues.
- [x] Implement plain-text rendering as a convenience over normalized segments.
- [x] Add table-driven unit tests with at least five relevant cases per new or modified test, including zero/nil-style and malformed cases where applicable.

## `yt-dlp` integration

- [x] Implement an injectable, context-aware command runner without shell invocation.
- [x] Add executable discovery and version reporting/validation.
- [x] Inspect video metadata through stable machine-readable output.
- [x] Decode only the required metadata fields and tolerate unrelated additions.
- [x] Represent manual and automatic caption tracks without exposing raw `yt-dlp` structures publicly.
- [x] Specify language matching, regional fallback, original-language behavior, and manual-versus-auto preference.
- [x] Implement deterministic track selection with extensive offline tests.
- [x] Retrieve exactly one selected subtitle track in the chosen format.
- [x] Isolate temporary files, constrain generated paths, and guarantee cleanup.
- [x] Redact signed URLs, cookies, and command secrets from bounded subprocess diagnostics; command arguments are never retained in errors.
- [x] Cover process failure, malformed JSON, missing output, non-zero exit, timeout, and cancellation.
- [x] Add opt-in integration tests for manual, automatic, multilingual, unavailable, and cancellation cases.

## Public API

- [x] Design a simple default constructor and functional options without over-configuring the common case.
- [x] Allow explicit `yt-dlp` paths and injectable providers for advanced use and testing; runner injection remains internal.
- [x] Define current defaults for languages and manual/automatic captions; translation is deliberately outside the caption-only API.
- [x] Decide whether empty language preferences mean original language, environment locale, English, or provider-best; original then provider-best is documented.
- [x] Ensure exported APIs are concurrency-safe where promised and document ownership of returned data.
- [x] Add GoDoc to every function and method, with detailed purpose and parameter documentation for production code.
- [x] Add compile-tested examples for basic use, language selection, errors, custom providers, and standalone parsing.
- [x] Review the API for semantic-versioning risks before the first release; decision 0005 records remaining pre-v1 risks.

## Development CLI

- [x] Create a thin CLI that only parses flags, calls the library, and renders results.
- [x] Accept a video URL or ID and repeatable/preferential language input.
- [x] Add manual-only source selection and a caller-controlled timeout.
- [x] Support plain-text and JSON output first.
- [x] Keep transcript output on stdout and diagnostics on stderr.
- [x] Define stable exit codes for invalid arguments, dependency problems, unavailable transcripts, and acquisition failures.
- [x] Add a dependency diagnostic command or flag (`--check` probes the configured yt-dlp executable without network access).
- [x] Add CLI smoke tests and usage examples; smoke tests build the real command and use an offline fake `yt-dlp` process.

## Documentation and release readiness

- [x] Create PlantUML package/class and acquisition sequence diagrams under `docs/uml`.
- [x] Keep `GOAL.md`, `ROADMAP.md`, `TODO.md`, and `IDEAS.md` aligned with implementation decisions through the normalized-core milestone.
- [x] Expand the README with the real module path, supported versions, installation, API examples, CLI examples, and limitations.
- [x] Document privacy, terms-of-service, copyright, rate-limit, and restricted-video considerations without claiming legal guarantees.
- [x] Document the tested `yt-dlp` version policy and response to upstream breakage.
- [x] Add changelog/release notes and a release checklist.
- [x] Run `gofmt`, unit tests, integration tests, `go vet ./...`, and the race detector where supported (verified 2026-08-12).
- [x] Choose v0.1.0 as the initial release version and human-readable duration strings for CLI JSON.
- [x] Add pinned Staticcheck 2026.1 and govulncheck v1.6.0 CI and Makefile gates.
- [x] Run the local release-hardening suite; no reachable vulnerabilities were reported on 2026-08-12.
- [ ] Confirm the updated CI workflow and live integration matrix pass on the exact release commit.
- [ ] Tag and publish v0.1.0, then perform the clean-install checks in `docs/RELEASING.md`.

## Later: optional speech-to-text

Items below are ordered by implementation dependency; do not begin production
adapter integration before the acquisition and prototype evidence it depends on.

### 1. Composition foundation

- [x] Define explicit opt-in fallback policies rather than falling back on every provider error; `FallbackProvider` falls back only for `ErrTranscriptUnavailable`.
- [x] Separate audio acquisition from transcription provider interfaces and compose them through `SpeechToTextProvider`.
- [x] Add provider-level duration and file-size guards before transcription.

### 2. Bounded `yt-dlp` audio acquisition

- [ ] Inspect video duration before downloading audio and reject configured over-limit inputs before expensive work begins.
- [ ] Download only audio with `yt-dlp` into isolated temporary storage without shell invocation.
- [ ] Enforce configured file-size bounds during acquisition where possible and verify the final artifact metadata.
- [ ] Guarantee downloaded audio and acquisition temp-file cleanup on success, failure, limit rejection, and cancellation.
- [ ] Add offline fake-process tests and opt-in live tests for audio selection, limits, cancellation, malformed/missing output, and cleanup.

### 3. Transcriber prototypes and architecture decision

- [ ] Prototype `whisper.cpp` and record executable/model setup, required input format, file-path requirements, timestamp quality, cancellation, portability, privacy, accuracy, and resource use.
- [ ] Prototype one hosted transcription provider, initially OpenAI unless evaluation selects another reference, and record accepted formats, upload/size limits, timestamps, cancellation, privacy, accuracy, latency, and cost.
- [ ] Compare the prototypes in a checked-in evaluation record using representative legal audio fixtures.
- [ ] Decide and record whether FFmpeg conversion belongs inside an audio source or behind a separate audio-processor interface.
- [ ] Document and test passthrough when downloaded audio is already compatible; require FFmpeg only for adapters that need conversion.

### 4. Production transcription adapter

- [ ] Implement at least one separately configured transcriber adapter and normalize its output into `Transcription` segments.
- [ ] Add adapter-aware cost and concurrency limits after selecting the production adapter.
- [ ] Bound and redact adapter diagnostics, credentials, request details, and temporary paths.
- [ ] Guarantee adapter and conversion temp-file cleanup on success, failure, and cancellation; the provider-level audio stream is already closed on all post-acquisition paths.

### 5. End-to-end fallback completion

- [ ] Define and document whether partial transcription results are returned or discarded on provider failure or cancellation.
- [ ] Compose the concrete speech-to-text provider with `FallbackProvider` without changing caption-first defaults.
- [ ] Test no-caption fallback, transcription failure, cancellation, duration/size/cost limits, partial-result policy, and cleanup across acquisition, optional conversion, and transcription.
