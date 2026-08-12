# Live YouTube test-video matrix

Live YouTube state is mutable: videos can disappear, captions can be
regenerated, and access can vary by region. Ordinary unit tests therefore use
sanitized repository fixtures without network access.

## Validated matrix

Validated on 2026-08-12 with `yt-dlp 2026.07.04`. All cases passed through the
public `Client.Get` API.

| Case | Video | Observed contract | Status |
| --- | --- | --- | --- |
| Manual English | `O8G5Mkzhe4s`, NASA Goddard's *Roman Space Telescope and the Journey to Space* (75 seconds) | Uploaded `en-US` WebVTT returns `SourceManual`. | Accepted; official NASA Goddard channel. |
| Automatic captions | `4IVomi9s4BA`, Google for Developers' *Continuous learning with Google in Udacity with spanish subtitles [spanish]* (43 seconds) | Spanish automatic WebVTT returns `SourceAutomatic`. | Accepted; automatic state remains mutable. |
| Multiple manual languages | `W01c2-2NubU`, Wikimedia Foundation's *Behind The Screen- The Global Edition* (5:04) | Uploaded `ar,en,es,fr,hi,id,ru,sw,uk`; requesting `fr` returns manual French. | Accepted; official Wikimedia Foundation channel. |
| No captions | `aqz-KE-bpKQ`, Blender Foundation's *Big Buck Bunny* (10:35) | No manual or automatic tracks; returns `ErrTranscriptUnavailable`. | Accepted; open movie, but absence remains mutable. |

The replaced candidates `ptfLfrW1648` and `BaW_jenozKc` were unavailable.
`TcP3jk0yJLM` worked but was rejected because it is a four-hour livestream.

## Refresh procedure

```console
yt-dlp --version
yt-dlp --dump-single-json --skip-download --no-warnings -- VIDEO_ID
yt-dlp --skip-download --list-subs -- VIDEO_ID
VOXSCRIPTA_YTDLP_INTEGRATION=1 go test -run TestYTDLPIntegration -v .
```

Record the date, version, identity, duration, and expected track tags. Remove
signed URLs, headers, cookies, tokens, and volatile fields before committing
JSON. A changed inventory should trigger review rather than silently selecting
another track.

## Replacement policy

If a candidate changes repeatedly, create short project-controlled videos: one
with uploaded English, one with English plus another language, one with
confirmed automatic captions, and one silent captionless clip. Keep scripts,
caption sources, licenses, and expected inventories in the repository.
