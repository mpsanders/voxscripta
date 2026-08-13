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
speech-to-text fallback's `YTDLPAudioSource` uses `yt-dlp` for checked audio.
The optional `WhisperCPPTranscriber` requires a `whisper-cli` executable and a
caller-selected GGML model. It passes verified mono 16 kHz 16-bit PCM WAV
through unchanged and invokes the configured FFmpeg executable for other input.
Use `--whisper-model` to enable this chain in the CLI; the library uses
`NewWhisperCPPTranscriber`. Current setup guidance is maintained by the
upstream whisper.cpp project. Real-runtime compatibility was validated with
whisper.cpp 1.9.2, FFmpeg 9.0.1, and the multilingual medium model on Windows
on 2026-08-13. See `docs/evaluations/whispercpp-2026-08-13.md`; this is a
known-good configuration, not a minimum-version claim.

The opt-in live adapter test additionally requires
`VOXSCRIPTA_WHISPER_INTEGRATION=1`, `VOXSCRIPTA_WHISPER_CLI`,
`VOXSCRIPTA_WHISPER_MODEL`, and `VOXSCRIPTA_WHISPER_SAMPLE`. The sample must be
a compatible mono 16 kHz 16-bit PCM WAV. On MinGW/MSYS2 Windows builds, ensure
the matching runtime DLL directory precedes unrelated MinGW installations on
`PATH`; an immediate `0xC0000139` exit indicates an incompatible DLL was found.

## Supported platforms

The initial platform matrix is Windows, Linux, and macOS on Go 1.25 and 1.26.
Go 1.25 is the minimum supported language/toolchain version. Support follows
the Go project's two-current-release policy.
