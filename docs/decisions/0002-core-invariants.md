# 0002: Normalized transcript invariants

Status: accepted, 2026-08-12

## Decision

A valid transcript has a valid 11-character YouTube video ID, a non-empty
language code, a known source kind, and at least one segment. Each segment has
non-negative timestamps, an end not earlier than its start, and non-whitespace
text. Segments are ordered by non-decreasing start time.

Overlapping and zero-duration segments are retained. Caption formats can
legitimately contain both, and later format-specific normalization should make
any removal or merging policy explicit rather than silently losing data in the
domain model.

Plain-text rendering trims segment edges, skips blank segments defensively, and
joins segments with a newline. It does not mutate the timestamped source model.

Cancellation uses the standard `context.Canceled` and `context.DeadlineExceeded`
errors so callers can inspect failures with `errors.Is`; VoxScripta does not add
a redundant cancellation sentinel.
