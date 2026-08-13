# Changelog

All notable changes will be documented here. The project follows Semantic
Versioning once a release is published; compatibility is not promised before
v1.0.0.

## [Unreleased]

### Added

- Separate audio-source and transcriber contracts, plus adapter-neutral
  speech-to-text composition with optional duration/file-size guards and
  surfaced audio-cleanup failures.
- A `yt-dlp` audio source with duration preflight, isolated temporary storage,
  best-effort transfer-size rejection, strict final-file verification, and
  cancellation cleanup.
- An opt-in `WhisperCPPTranscriber` with JSON timestamp normalization, verified
  PCM-WAV passthrough, conditional FFmpeg conversion, bounded path-redacted
  diagnostics, cancellation, and temporary-artifact cleanup.
- Explicit CLI local speech fallback through `--whisper-model`, ordered after
  caption acquisition and protected by duration and byte limits.
- Opt-in real-runtime whisper.cpp integration coverage and a checked-in local
  prototype evaluation for whisper.cpp 1.9.2 with FFmpeg 9.0.1.

### Fixed

- Interpret whisper.cpp JSON offsets as milliseconds, preventing speech
  segment timestamps from being inflated tenfold.

### Security

- Library-controlled `yt-dlp` calls ignore ambient configuration files and
  subprocess output capture is bounded.

## [0.1.0] - 2026-08-12

### Added

- Provider-independent transcript model, WebVTT normalization, and rendering.
- yt-dlp caption discovery, deterministic selection, and isolated retrieval.
- Public acquisition client with custom-provider support.
- Development CLI with text and JSON output, stable error exits, and an
  offline dependency check.
- Human-readable duration strings in CLI JSON output.
- Offline unit and process-level smoke tests plus opt-in live integration tests.
- Pinned Staticcheck and govulncheck release-hardening gates in CI and the
  Makefile.
- Explicit provider fallback composition that runs only when the primary
  provider reports an unavailable transcript.

### Changed

- Removed unused translated-caption fields and source constants from the
  pre-v1 API; future translation support will use a separately designed
  interface.

### Security

- Subprocesses avoid shell invocation, temporary output is isolated and
  cleaned up, and bounded diagnostics redact common secrets and signed URLs.
