# Runtime dependencies

The normalized core uses only the Go standard library and has no external runtime dependency. Caption acquisition
will require `yt-dlp`; VoxScripta will detect it but will never install or update
it for callers.

## Installing `yt-dlp`

Use an installation method documented by the upstream project. The standalone
release binary or `pip` package are common choices. Ensure the executable is on
`PATH`, or later configure its explicit path through the library.

Verify the installation with:

```console
yt-dlp --version
```

The project supports the current `yt-dlp` nightly channel and current stable
release. Because YouTube changes independently, users should update an old
installation before reporting extraction failures. Compatibility will be
tested against a recorded version during provider integration; no minimum
version is claimed until that contract has been observed and tested.

`ffmpeg` is not required for caption-only operation.

## Supported platforms

The initial platform matrix is Windows, Linux, and macOS on Go 1.25 and 1.26.
Go 1.25 is the minimum supported language/toolchain version. Support follows
the Go project's two-current-release policy.
