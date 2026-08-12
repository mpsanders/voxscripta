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
- [ ] Select stable public test videos representing manual captions, automatic captions, multiple languages, and no captions; record why each is safe to use.
- [ ] Capture representative, sanitized `yt-dlp` JSON and subtitle fixtures for offline tests.
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
- [ ] Redact signed URLs, cookies, and command secrets from diagnostics.
- [ ] Cover process failure, malformed JSON, missing output, non-zero exit, timeout, and cancellation.
- [ ] Add opt-in integration tests that skip clearly when `yt-dlp` or network access is unavailable.

## Public API

- [x] Design a simple default constructor and functional options without over-configuring the common case.
- [x] Allow explicit `yt-dlp` paths and injectable providers for advanced use and testing; runner injection remains internal.
- [x] Define current defaults for languages and manual/automatic captions; translated-caption behavior remains to be implemented.
- [x] Decide whether empty language preferences mean original language, environment locale, English, or provider-best; original then provider-best is documented.
- [x] Ensure exported APIs are concurrency-safe where promised and document ownership of returned data.
- [ ] Add GoDoc to every function and method, with detailed purpose and parameter documentation for production code.
- [ ] Add compile-tested examples for basic use, language selection, errors, and custom providers.
- [ ] Review the API for semantic-versioning risks before the first release.

## Development CLI

- [x] Create a thin CLI that only parses flags, calls the library, and renders results.
- [x] Accept a video URL or ID and repeatable/preferential language input.
- [x] Add manual-only source selection and a caller-controlled timeout; translated-source selection remains future work.
- [x] Support plain-text and JSON output first.
- [x] Keep transcript output on stdout and diagnostics on stderr.
- [x] Define stable exit codes for invalid arguments, dependency problems, unavailable transcripts, and acquisition failures.
- [ ] Add a dependency diagnostic command or flag (`--version` is implemented).
- [ ] Add CLI smoke tests and usage examples.

## Documentation and release readiness

- [x] Create PlantUML package/class and acquisition sequence diagrams under `docs/uml`.
- [x] Keep `GOAL.md`, `ROADMAP.md`, `TODO.md`, and `IDEAS.md` aligned with implementation decisions through the normalized-core milestone.
- [ ] Expand the README with the real module path, supported versions, installation, API examples, CLI examples, and limitations.
- [ ] Document privacy, terms-of-service, copyright, rate-limit, and restricted-video considerations without claiming legal guarantees.
- [ ] Document the tested `yt-dlp` version policy and response to upstream breakage.
- [ ] Add changelog/release notes and a release checklist.
- [ ] Run `gofmt`, unit tests, integration tests, `go vet ./...`, and the race detector where supported.

## Later: optional speech-to-text

- [ ] Define explicit opt-in fallback policies rather than falling back on every provider error.
- [ ] Separate audio acquisition from transcription provider interfaces.
- [ ] Add duration, file-size, cost, and concurrency limits.
- [ ] Evaluate local and hosted speech-to-text providers against timestamps, accuracy, portability, privacy, and cost.
- [ ] Implement one provider adapter and normalize its output.
- [ ] Guarantee audio/temp-file cleanup on success, failure, and cancellation.
- [ ] Document `ffmpeg` only if the selected path requires it.
- [ ] Test no-caption fallback, transcription failure, cancellation, and partial-result policy.
