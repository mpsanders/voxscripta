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

Status: release candidate. Cross-platform CI, formatting, vet, deterministic
tests, race coverage, pinned Staticcheck, and pinned govulncheck gates are
established. Responsible-use and upstream-breakage guidance, a changelog, and
a release checklist are documented. Local release-hardening checks passed on
2026-08-12. Publishing v0.1.0 remains a maintainer action after the updated CI
workflow and final live integration matrix pass on the release commit.

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
requires explicit composition. It triggers only for unavailable transcripts;
cancellation, invalid input, dependency errors, and provider failures do not
start fallback work. Audio and transcription contracts remain pending a
concrete adapter evaluation.

Extend coverage without making heavy dependencies part of the core caption path.

Deliverables:

- Define an audio-source and speech-to-text provider contract.
- Download bounded audio with `yt-dlp`; use `ffmpeg` only when the chosen provider requires conversion.
- Add at least one separately configured local or remote transcription adapter.
- Define fallback policy, cost/size/duration guards, language hints, and progress/diagnostic behavior.
- Normalize speech-to-text segments into the same `Transcript` model.

Exit criteria:

- Caption retrieval remains the default and has no new mandatory runtime dependency.
- Callers must explicitly configure or opt into potentially costly speech-to-text work.
- Tests cover fallback ordering, cancellation, partial output, and cleanup.

## Milestone 6: Ecosystem features

Add only features justified by real consumers.

Candidates include:

- SRT, WebVTT, Markdown, and richer JSON renderers.
- Cache interfaces with expiry and provider-version awareness.
- Batch APIs with bounded concurrency and per-item results.
- Custom HTTP clients, proxy/cookie configuration, and rate-limit hooks.
- Metrics, tracing, structured diagnostic events, and provider health checks.
- Additional providers for managed-channel captions or transcription services.
