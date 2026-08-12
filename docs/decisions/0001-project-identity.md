# 0001: Project identity and compatibility

Status: accepted, 2026-08-12

## Decision

- Module path: `github.com/mpsanders/VoxScripta`
- Exported package: `transcript`
- Development CLI: `ytextract`
- License: MIT
- Minimum Go version: 1.25
- Initial operating systems: Windows, Linux, and macOS
- Versioning: semantic versioning; public compatibility is not promised before v1
- Primary logo: [`docs/assets/voxscripta-logo.png`](../assets/voxscripta-logo.png)

## Rationale

The package name describes the imported capability while the repository name
provides project identity. A separate CLI name keeps commands concise. Supporting
the two current Go release families balances compatibility with maintenance.

The logo combines a caption outline, transcript lines, timestamp nodes, and a
play cue to represent video speech becoming structured text. Its simple navy and
cyan silhouette is intended to remain recognizable at repository-header and icon
sizes without relying on platform-specific branding.
