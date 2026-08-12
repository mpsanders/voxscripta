# Responsible use and operational considerations

VoxScripta retrieves captions that are available to the caller through
`yt-dlp`. It does not grant rights to copy, publish, train on, or otherwise use
video or caption content. Callers are responsible for checking applicable law,
copyright licences, platform terms, and organizational policy. This document
is operational guidance, not legal advice.

## Privacy and sensitive content

Transcripts can contain personal, confidential, health, financial, or other
sensitive information even when a video is publicly reachable. Minimize what
you retain, restrict access, encrypt storage and transport where appropriate,
and define deletion periods. Do not send transcripts to another service unless
that disclosure is permitted and understood by the affected users.

Provider failures contain bounded, normalized stderr with common URLs and
credential-shaped values redacted. Redaction is defense in depth, not a
guarantee. Treat diagnostics as potentially sensitive and avoid publishing
them or storing them indefinitely.

Speech-to-text acquisition temporarily stores the video's full selected audio
stream, which can be more sensitive and rights-restricted than captions alone.
Direct audio-source callers must close the returned stream promptly so its
isolated temporary directory can be removed, and must handle cleanup errors.
Before configuring a future hosted transcriber, treat the upload as disclosure
of the source audio to another service and review its retention and training
policies, data location, credentials, cost, and the permissions applicable to
the recording.

## Platform access and restrictions

The project does not bypass authentication, DRM, geographic restrictions, or
other access controls. Private, age-restricted, member-only, removed, or
region-restricted videos may fail. Do not use cookies or credentials belonging
to another person, and do not attempt to evade an access decision.

YouTube and other relevant services may change their terms and technical
controls. Review the terms that apply to your use case rather than assuming
that technical availability implies permission.

## Rate limits and reliability

Repeated requests can trigger upstream throttling or anti-abuse controls.
Applications should use caller-controlled timeouts, conservative concurrency,
backoff outside the library when retrying transient failures, and caching only
when content rights and retention policy allow it. Do not retry invalid,
unavailable, restricted, or cancelled requests as though they were transient.

Caption accuracy varies. Automatic captions and translated text can contain
material errors, so do not use them as authoritative records without suitable
review, especially for safety-critical or consequential decisions.
