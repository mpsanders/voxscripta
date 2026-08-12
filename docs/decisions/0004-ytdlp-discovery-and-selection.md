# 0004: yt-dlp discovery and caption selection

Status: accepted, 2026-08-12

## Decision

The provider invokes `yt-dlp` directly through `exec.CommandContext`; it never
uses a command shell. The runner is injectable and captures stdout and stderr
separately. Cancellation is returned as `context.Canceled` or
`context.DeadlineExceeded`. Failed commands may expose a single-line stderr
diagnostic capped at 2 KiB. HTTP(S) URLs and common authorization, cookie,
proxy, and token values are redacted first. Command arguments are not retained
in the error because future configuration may place secrets in them.

Dependency probing uses `yt-dlp --version`. Discovery uses:

```console
yt-dlp --dump-single-json --skip-download --no-warnings -- VIDEO_ID
```

The decoder retains only the video ID, title, original language, and WebVTT
entries from `subtitles` and `automatic_captions`. Unknown JSON fields and
non-WebVTT formats are ignored. Caption URLs remain internal and must not appear
in errors or diagnostic metadata because they can contain signed parameters.

Requested languages are evaluated in caller order. For each preference, an
exact BCP 47 tag wins, followed by its base language, followed by another
regional variant. Manual captions win over automatic captions at the same
language match rank. Automatic captions are considered only when explicitly
allowed.

When no languages are requested, the video's reported original language is
tried first. If it is absent or unavailable, the first deterministic eligible
manual track is selected, then the first automatic track when allowed. Explicit
language preferences never fall back to an unrelated language.

Retrieval asks `yt-dlp` for only the selected language and source, fixes the
subtitle format to WebVTT, and writes into a newly created private temporary
directory. The implementation accepts exactly one regular `.vtt` file directly
inside that directory, rejects missing or ambiguous output, reads it, and
removes the entire directory on every return path. Manual and automatic tracks
use `--write-subs` and `--write-auto-subs`, respectively. The signed URL found
during discovery is deliberately not passed through diagnostics or used as the
download target.

## Consequences

Discovery, selection, isolated retrieval, and diagnostic sanitization are
deterministic and fully testable offline. Redaction is defense in depth rather
than a guarantee that arbitrary upstream prose can never contain sensitive
data, so callers should still treat provider failures as operational data. The
command and JSON shape are an implementation contract, not yet a claimed
minimum-version compatibility guarantee. Translated-track modeling, live
fixtures, and opt-in integration tests remain separate work.
