# 0007: yt-dlp audio acquisition safeguards

Status: accepted, 2026-08-13

## Decision

`YTDLPAudioSource` is an optional public adapter for speech-to-text composition.
It accepts a YouTube ID or supported URL, normalizes it to an ID, and uses the
internal context-aware runner without a command shell. Every controlled
`yt-dlp` invocation ignores ambient configuration.

Acquisition first inspects duration and live status. A positive duration limit
rejects live, unknown-duration, and known over-limit inputs before download.
Zero disables the limit. The download command selects `bestaudio`, writes one
artifact to an isolated temporary directory, suppresses progress output, and
passes `--max-filesize` when a positive byte limit is configured. The completed
artifact must be the only regular file, must be non-empty, and must not exceed
the limit. Its lower-case extension is exposed as a container hint, not a codec
guarantee.

`yt-dlp --max-filesize` is not a portable hard in-flight bound for every
manifest or fragmented protocol. Therefore `MaxBytes` is best-effort during
transfer and strict for the completed artifact. A future hard disk/download
budget requires separate evaluation rather than an overstated guarantee.

The caller owns directly acquired data and must close it promptly. Closing the
stream closes the file and removes its containing temporary directory;
`SpeechToTextProvider` performs this automatically and returns cleanup failures.
Acquisition errors, limit rejection, and cancellation also remove the directory
and surface cleanup failures.

## Consequences

- Caption-only consumers gain no new runtime dependency.
- Unknown/live media cannot bypass an enabled duration guard.
- Ambient `yt-dlp` configuration cannot add hooks, outputs, or postprocessing.
- Direct callers can choose their transcriber, privacy, cost, and conversion
  policy, but must honor stream ownership.
- Strict in-flight file-size enforcement remains an explicit backlog item.
