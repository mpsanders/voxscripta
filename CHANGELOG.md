# Changelog

All notable changes will be documented here. The project follows Semantic
Versioning once a release is published; compatibility is not promised before
v1.0.0.

## Unreleased

Planned version: v0.1.0.

### Added

- Provider-independent transcript model, WebVTT normalization, and rendering.
- yt-dlp caption discovery, deterministic selection, and isolated retrieval.
- Public acquisition client with custom-provider support.
- Development CLI with text and JSON output, stable error exits, and an
  offline dependency check.
- Human-readable duration strings in CLI JSON output.
- Offline unit and process-level smoke tests plus opt-in live integration tests.

### Changed

- Removed unused translated-caption fields and source constants from the
  pre-v1 API; future translation support will use a separately designed
  interface.

### Security

- Subprocesses avoid shell invocation, temporary output is isolated and
  cleaned up, and bounded diagnostics redact common secrets and signed URLs.
