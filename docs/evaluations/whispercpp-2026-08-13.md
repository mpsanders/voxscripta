# whisper.cpp local prototype evaluation

Date: 2026-08-13

## Configuration

- whisper.cpp: 1.9.2, locally built `whisper-cli` on Windows 11
- model: multilingual `ggml-medium.bin` (1,533,763,059 bytes)
- FFmpeg: 9.0.1 full build
- hardware observed by whisper.cpp: 8 logical CPUs, 4 worker threads, no GPU backend
- input: upstream whisper.cpp `samples/jfk.wav`, 11 seconds, mono 16 kHz

The sample is the public-domain John F. Kennedy inaugural-address excerpt
distributed by whisper.cpp. The repository does not copy the audio or model.

## Results

The adapter-compatible command produced one English segment from 0 to 11,000
milliseconds. Its text correctly transcribed the familiar "ask not" sentence,
with only inconsequential punctuation/leading-whitespace differences. Total
reported runtime was 17.1 seconds: 0.64 seconds loading, 12.66 seconds encoding,
and the remainder sampling/decoding. The model plus reported working buffers
accounted for about 1.91 GB before unreported process overhead. This is one
fixture on one CPU and is compatibility evidence, not a general accuracy or
performance benchmark.

The live output established that whisper.cpp 1.9.2 JSON offsets are
milliseconds. It exposed and led to correction of an adapter error that had
treated each unit as 10 milliseconds.

## Contract observations

- `whisper-cli` requires a model and file path; the adapter's private staging
  directory matches that contract.
- Compatible PCM WAV passed directly to whisper.cpp. The existing adapter uses
  FFmpeg only for incompatible input.
- JSON contained detected language, integer millisecond offsets, printable
  timestamps, text, model details, and system details. The adapter decodes only
  language, offsets, and text.
- Pre-cancellation is deterministic. In-flight cancellation uses Go's
  `CommandContext`, terminates the local process, discards partial JSON, and is
  covered by offline process tests; cancellation latency during a real decode
  was not benchmarked.
- Local processing sends neither audio nor model data to a hosted service.
  Model acquisition and any input acquisition remain separate caller actions.
- The medium model is portable but resource-heavy. Smaller models should be
  evaluated where latency or memory matters; accuracy was not compared here.

## Windows runtime caveat

This local MinGW/MSYS2 build initially exited with Windows status `0xC0000139`
because Git's MinGW directory preceded MSYS2 UCRT on `PATH` and supplied an
incompatible C++ runtime DLL. Prepending `C:\msys64\ucrt64\bin` fixed the
installed binary. This is a build/distribution environment issue rather than
an adapter issue, but dependency diagnostics should preserve the non-zero
process failure so callers can act on it.

## Conclusion

The local adapter is runtime-compatible with whisper.cpp 1.9.2 after correcting
offset units. It is suitable as an explicit experimental fallback with the
documented resource and installation caveats. Production qualification still
requires representative multi-language/noisy audio, smaller-model comparison,
measured in-flight cancellation, and end-to-end YouTube fallback coverage.
