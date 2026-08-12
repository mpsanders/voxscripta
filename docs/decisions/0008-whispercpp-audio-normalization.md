# 0008: Keep whisper.cpp normalization inside its adapter

Status: accepted, 2026-08-13

## Decision

The first concrete local transcriber is `WhisperCPPTranscriber`. It invokes
`whisper-cli` with a caller-supplied GGML model and consumes its JSON segment
output. The adapter is opt-in and does not change caption-only dependencies.

Portable upstream `whisper-cli` builds accept 16-bit WAV input. `Audio.Format`
is only a container hint, so the adapter inspects the staged RIFF/WAVE `fmt `
chunk. Mono, 16 kHz, 16-bit PCM passes through unchanged. Every other input is
normalized by a caller-selectable FFmpeg executable. Conversion belongs inside
this adapter because it is a transcriber requirement, not an acquisition or
domain-model requirement. A shared processor interface is deferred until a
second adapter demonstrates a common contract.

The adapter uses private temporary storage because the CLI is file-oriented. It
bounds process output, removes paths from diagnostics, honors context
cancellation, and removes staged input, converted audio, and JSON on every
return path. Partial output is discarded on failure; only a complete, valid
`Transcription` is returned.

The CLI enables the chain only with `--whisper-model`. Ordering is captions
first, then local speech-to-text. It does not discover or download models and
does not enable hosted providers from ambient credentials.

## Evidence and limitations

Offline tests cover arguments, WAV passthrough, FFmpeg conversion, JSON timing,
language hints/detection, malformed output, missing dependencies, failures, and
cleanup. On 2026-08-13 neither FFmpeg nor `whisper-cli` was installed locally,
so real-runtime accuracy, latency, memory use, version compatibility, and
cancellation latency remain unvalidated. Live evaluation remains required
before the adapter is described as production-ready.

## Consequences

- Caption consumers gain no mandatory dependency.
- Local fallback requires explicit model, executable, storage, and compute
  choices.
- Compatible PCM WAV avoids redundant conversion; other input requires FFmpeg.
- Results identify whisper.cpp, but richer model/conversion provenance remains
  planned.
