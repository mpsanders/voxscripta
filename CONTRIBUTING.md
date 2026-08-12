# Contributing

VoxScripta is currently establishing its public API. Please discuss substantial
API or provider changes before implementation and keep changes focused.

## Development checks

Install a supported Go toolchain, then run:

```console
make check
make race
```

Release-hardening checks additionally use pinned analysis tools:

```console
go install honnef.co/go/tools/cmd/staticcheck@2026.1
go install golang.org/x/vuln/cmd/govulncheck@v1.6.0
make hardening
```

`make staticcheck` is deterministic for a given tool version. `make vuln`
queries the Go vulnerability database, so it requires network access and its
result reflects the database and Go toolchain at the time it runs. Update tool
pins deliberately in this file and CI together.

Use `make help` to list individual targets. `make check` remains the offline
day-to-day gate; `make hardening` includes the race detector and the two
separately installed analysis tools. The CLI can be built with
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
