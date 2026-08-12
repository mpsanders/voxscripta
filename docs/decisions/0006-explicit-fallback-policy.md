# 0006: Explicit provider fallback policy

Status: accepted, 2026-08-12

## Decision

Fallback acquisition is opt-in through `FallbackProvider`. It invokes its
fallback provider only when the primary provider returns
`ErrTranscriptUnavailable`. It passes through cancellation, invalid input,
missing dependencies, and provider failures without starting more work.

The fallback provider remains a normal `Provider`. A speech-to-text provider
can therefore be composed without adding speech-to-text dependencies to the
caption path or changing `Client`.

Amendment, 2026-08-13: separate `AudioSource` and `Transcriber` contracts are
now exported and composed by `SpeechToTextProvider`. It owns and closes acquired
audio after transcription, returns cleanup failures, and discards invalid or
partial transcriber output. Concrete transcriber-specific partial-result,
cost, concurrency, and conversion policies remain deferred until prototype
evidence exists.

## Consequences

- Callers must deliberately construct and install a fallback chain.
- A broken caption provider cannot silently trigger potentially costly or
  privacy-sensitive transcription.
- Both providers receive the same context, video ID, and acquisition options.
- More elaborate retry or multi-provider policies remain outside this small
  first contract.
