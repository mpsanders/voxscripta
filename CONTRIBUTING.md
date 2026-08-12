# Contributing

VoxScripta is currently establishing its public API. Please discuss substantial
API or provider changes before implementation and keep changes focused.

## Development checks

Install a supported Go toolchain, then run:

```console
make check
make race
```

Use `make help` to list individual targets. The CLI can be built with
`make build` or run directly with options, for example
`make run ARGS="--version"`.

Ordinary tests must remain offline and deterministic. Live YouTube and
`yt-dlp` integration tests must be opt-in and skip with a clear reason when a
dependency is unavailable. New or modified tests should be table-driven where
practical and contain at least five meaningful cases.

## Pull requests

- Add GoDoc to exported and unexported production functions and methods.
- Update the README, planning documents, and UML when behavior or architecture changes.
- Do not commit cookies, signed caption URLs, credentials, or copyrighted subtitle data.
- Confirm that any test video and captured fixture is legal to redistribute.

Contributions are accepted under the repository's MIT License.
