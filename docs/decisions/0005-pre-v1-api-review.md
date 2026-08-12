# 0005: Pre-v1 public API review

Status: accepted, 2026-08-12

## Decision

The caption-only public API is an acceptable pre-v1 foundation. `Client`,
`Provider`, `Options`, the normalized transcript types, parsing entry points,
functional options, and sentinel errors remain exported. Provider-specific
command execution and metadata stay internal.

The following compatibility risks are explicitly deferred until evidence from
real consumers justifies a change:

- `Options` is a value struct, so adding fields is source compatible for keyed
  literals but can break unkeyed literals. Documentation and examples use
  keyed literals; v1 should decide whether to retain the struct or move to a
  policy type before compatibility is promised.
- `Provider` returns a complete `Transcript`. Streaming, diagnostics, health
  checks, and fallback chains should use separate optional interfaces rather
  than adding methods to `Provider` and breaking implementers.
- Translation is omitted from the caption-only public API. It may be added
  later behind a distinct interface with explicit source-language and policy
  semantics rather than reserved, non-functional fields.
- CLI JSON is a presentation model separate from the public Go structs.
  Segment timestamps are human-readable Go duration strings such as `"1.25s"`
  rather than integer nanoseconds.

The CLI dependency check remains CLI-specific and uses the internal yt-dlp
client. It does not add provider-specific diagnostics to the public API.

## Consequences

Milestone 3 can close without prematurely expanding the public surface. The
first planned release is v0.1.0; compatibility is not promised until v1. A
future v1 remains subject to consumer feedback and a final exported-API review.
