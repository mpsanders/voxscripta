# Release checklist

## Compatibility and documentation

- Confirm the planned initial version, v0.1.0, and summarize user-visible
  changes in `CHANGELOG.md`.
- Review exported API changes and the risks in decision 0005.
- Confirm README examples, dependency policy, diagrams, roadmap, and TODO match
  the implementation.
- Confirm CLI JSON timestamps remain documented human-readable duration
  strings and translation remains outside the caption-only API.

## Verification

- Run `make check` on a clean worktree.
- Run `make race` on a platform supported by Go's race detector.
- Run `make integration` with the documented known-good yt-dlp version.
- Confirm CI passes on the supported Go versions across Windows, Linux, and
  macOS.
- Review dependency and vulnerability reports; document or remediate findings.
- Build the CLI and run `ytextract --check` against the release environment.

## Publication

- Verify the repository has no credentials, downloaded captions, temporary
  media, or generated binaries staged for release.
- Tag the exact reviewed commit with an annotated semantic-version tag.
- Publish release notes from the changelog and include supported Go and yt-dlp
  policy plus known limitations.
- After publication, install from the tag in a clean directory and repeat the
  version and dependency checks.

Rollback is a new corrective release; published tags must not be moved.
