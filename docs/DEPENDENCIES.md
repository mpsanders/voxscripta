# Runtime dependencies

The normalized core uses only the Go standard library and has no external
runtime dependency. Caption and audio acquisition require `yt-dlp`; VoxScripta
detects it but never installs or updates it for callers.

## Installing `yt-dlp`

Use an installation method documented by the upstream project. The standalone
release binary or `pip` package are common choices. Ensure the executable is on
`PATH`, or configure its explicit path through the library.

Verify the installation with:

```console
yt-dlp --version
```

The project supports the current `yt-dlp` nightly channel and current stable
release. Because YouTube changes independently, users should update an old
installation before reporting extraction failures. The live caption and audio
integration contracts were validated with `yt-dlp 2026.07.04` on 2026-08-13.
This is a known-good version, not a claimed minimum version. Check an installation with
`ytextract --check`; this invokes only `yt-dlp --version` and performs no video
or network acquisition.

After an upstream update, run `make integration`. This target sets
`VOXSCRIPTA_YTDLP_INTEGRATION=1` and runs only the live integration suite; it
requires network access. If a previously passing test fails, first reproduce
with the recorded known-good version. For a new-version regression, capture
sanitized metadata and command behavior, update the internal adapter and
offline fixtures, rerun the full matrix, and record the newly validated version
here and in `TEST_VIDEOS.md`. Security fixes can supersede a known-good version;
do not recommend a vulnerable pin merely to preserve extraction behavior.

`ffmpeg` is not required for caption-only or audio-acquisition operation. The
speech-to-text fallback's `YTDLPAudioSource` uses `yt-dlp` for checked audio
acquisition. The planned policy is for FFmpeg to remain an optional, explicitly
detected dependency and run only when the configured transcriber cannot consume
the downloaded audio directly. Milestone 5 includes `whisper.cpp` and
hosted-provider prototypes to determine that boundary before
a production adapter contract is fixed.

## Supported platforms

The initial platform matrix is Windows, Linux, and macOS on Go 1.25 and 1.26.
Go 1.25 is the minimum supported language/toolchain version. Support follows
the Go project's two-current-release policy.
