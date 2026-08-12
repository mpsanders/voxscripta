# 0003: WebVTT parsing and rolling-caption normalization

Status: accepted, 2026-08-12

## Decision

The first caption format is WebVTT. The public `ParseWebVTT` function accepts an
`io.Reader` and returns normalized, chronologically ordered segments. It accepts
cue identifiers and settings, ignores NOTE, STYLE, and REGION blocks, strips cue
markup, decodes HTML entities, collapses whitespace, and retains overlaps and
zero-duration cues.

For consecutive cues that overlap in time, the parser removes the longest suffix
of the prior raw cue that is also a prefix of the current cue. A fully repeated
cue is discarded. No de-duplication occurs across a timing gap. This models both
cumulative and sliding-window automatic captions while limiting changes to cues
that are evidence of a rolling display.

Malformed timing is `ErrInvalidInput`, a missing WebVTT signature is
`ErrUnsupportedFormat`, and a document without non-empty cues is
`ErrTranscriptUnavailable`.

## Rationale

`github.com/asticode/go-astisub` was evaluated as an established parser. It is a
broad subtitle conversion toolkit and introduced several transport and utility
modules for a small parsing requirement. A focused parser keeps the normalized
core standard-library-only and makes the accepted `yt-dlp` subset and rolling
caption policy explicit in fixtures. Broader WebVTT conformance can be revisited
if observed provider output exceeds this contract.
