# Sanitized yt-dlp metadata fixtures

These fixtures record only the metadata contract used by VoxScripta. They were
derived from `yt-dlp 2026.07.04` observations on 2026-08-12. Signed caption and
format URLs, HTTP headers, cookies, thumbnails, and unrelated volatile fields
were removed. Placeholder `https://example.invalid/` URLs keep usable track
shapes testable without retaining live credentials.

Live inventory expectations are documented in `docs/planning/TEST_VIDEOS.md`.
