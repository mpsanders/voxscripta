# Ideas

Ideas are deliberately separate from the committed roadmap. Promote an idea only after a concrete consumer need, scope, and maintenance cost are understood.

## Acquisition and provider ideas

- A Go-native caption provider as an optional fast path, retaining `yt-dlp` as the compatibility fallback.
- An authenticated YouTube Data API provider for captions belonging to channels the caller can manage.
- Adapters for local `whisper.cpp` and hosted transcription services.
- User-supplied provider chains with rules based on language, duration, privacy, latency, or cost.
- Provider capability discovery so applications can explain what is available before acquisition.
- Cookies, proxy, and custom HTTP transport support through explicit secure configuration.

## Transcript quality ideas

- Confidence scores where the source supplies them.
- Speaker diarization and speaker labels without pretending inferred labels are authoritative.
- Sentence reconstruction and punctuation cleanup as optional transforms that preserve original segments.
- Word-level timestamps alongside the simpler segment model.
- Transcript quality indicators based on source type, language match, cue continuity, and coverage.
- Separate raw and normalized views for auditability.
- Chapter detection and alignment with YouTube chapters.

## Output and downstream use

- SRT, WebVTT, Markdown, and JSON renderers.
- Timestamped Markdown links back to the exact point in the video.
- Streaming or iterator-style access for very large transcripts.
- Search, slicing, and time-range utilities.
- Chunking helpers designed for summarization, embeddings, and retrieval-augmented generation while preserving citation timestamps.
- Optional translation behind a distinct interface, clearly identifying translated output.

## Operations and performance

- Cache interfaces rather than a mandatory cache implementation.
- Cache keys incorporating video ID, language policy, source, normalization version, and provider version.
- Singleflight request coalescing for duplicate concurrent acquisitions.
- Batch retrieval with bounded concurrency, rate limiting, and individual error results.
- Structured diagnostic events, OpenTelemetry hooks, and acquisition timing metrics.
- A health/doctor API that reports executable versions and provider readiness.
- Reproducible test fixtures with a small tool for intentional refresh and review.

## Distribution and developer experience

- Prebuilt CLI binaries while keeping `yt-dlp` an explicit external dependency.
- Container examples for reproducible integration environments.
- A compatibility matrix for Go, `yt-dlp`, operating systems, and optional transcription backends.
- A separate examples repository or integration examples once the public API stabilizes.
- Fuzz tests for URL, metadata, WebVTT, and SRT parsers.

## Questions to resolve through prototypes

- Is direct subtitle URL retrieval more reliable and testable than asking `yt-dlp` to write a subtitle file?
- Which machine-readable subtitle format best preserves YouTube timing while minimizing rolling-caption duplication?
- Should default language selection prefer the video's original language or a documented application default?
- Should translated caption tracks be opt-in because their quality and proliferation differ from source tracks?
- What minimum `yt-dlp` version can be supported without creating brittle version branches?
- How much provider diagnostic metadata can be retained without leaking ephemeral signed URLs or credentials?
- Does a streaming API provide meaningful value when `yt-dlp` discovery and caption retrieval are naturally whole-result operations?
