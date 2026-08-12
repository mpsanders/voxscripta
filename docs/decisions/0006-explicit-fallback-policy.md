# 0006: Explicit provider fallback policy

Status: accepted, 2026-08-12

## Decision

Fallback acquisition is opt-in through `FallbackProvider`. It invokes its
fallback provider only when the primary provider returns
`ErrTranscriptUnavailable`. It passes through cancellation, invalid input,
missing dependencies, and provider failures without starting more work.

The fallback provider remains a normal `Provider`. A future speech-to-text
adapter can therefore be composed without adding speech-to-text dependencies
to the caption path or changing `Client`. Audio acquisition and transcription
contracts are deliberately deferred until an adapter is selected and their
resource ownership, limits, and partial-result semantics can be tested.

## Consequences

- Callers must deliberately construct and install a fallback chain.
- A broken caption provider cannot silently trigger potentially costly or
  privacy-sensitive transcription.
- Both providers receive the same context, video ID, and acquisition options.
- More elaborate retry or multi-provider policies remain outside this small
  first contract.
