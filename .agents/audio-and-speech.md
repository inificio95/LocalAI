# Audio and Speech in LocalAI

This guide covers integrating and working with audio backends in LocalAI, including Text-to-Speech (TTS), Speech-to-Text (STT/transcription), and audio processing pipelines.

## Overview

LocalAI supports OpenAI-compatible audio endpoints:
- `/v1/audio/transcriptions` — Speech-to-Text (Whisper-compatible)
- `/v1/audio/translations` — Translate audio to English text
- `/v1/audio/speech` — Text-to-Speech synthesis

## Supported Backends

### Speech-to-Text (STT)
- **whisper.cpp** — Fast C++ inference for OpenAI Whisper models
- **faster-whisper** — Python-based, optimized with CTranslate2

### Text-to-Speech (TTS)
- **piper** — Fast, local neural TTS (recommended for CPU)
- **bark** — High-quality but slower TTS with voice cloning
- **coqui** — Multi-speaker TTS with XTTS support
- **vall-e-x** — Zero-shot voice synthesis

## Model Configuration

### Whisper (STT) Example

```yaml
name: whisper-1
backend: whisper
parameters:
  model: ggml-medium.en.bin
audio:
  # Whisper-specific settings
  language: en          # Force language (omit for auto-detect)
  translate: false      # Set true for translation endpoint
  no_timestamps: false
  threads: 4
```

### Piper (TTS) Example

```yaml
name: tts-1
backend: piper
parameters:
  model: en_US-lessac-medium.onnx
audio:
  voice: default
  # Piper outputs WAV; LocalAI converts to requested format
  sample_rate: 22050
```

### Bark (TTS) Example

```yaml
name: bark
backend: bark
parameters:
  model: bark
audio:
  voice_preset: v2/en_speaker_6
  # Bark is GPU-hungry; set threads accordingly
f16: true
```

## API Usage

### Transcription Request

```bash
curl http://localhost:8080/v1/audio/transcriptions \
  -H "Authorization: Bearer $API_KEY" \
  -F file=@audio.mp3 \
  -F model=whisper-1 \
  -F response_format=json \
  -F language=en
```

Response:
```json
{
  "text": "Hello, this is a transcription of the audio file."
}
```

### TTS Request

```bash
curl http://localhost:8080/v1/audio/speech \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tts-1",
    "input": "The quick brown fox jumps over the lazy dog.",
    "voice": "alloy",
    "response_format": "mp3",
    "speed": 1.0
  }' \
  --output speech.mp3
```

Supported `response_format` values: `mp3`, `opus`, `aac`, `flac`, `wav`, `pcm`

## Adding a New Audio Backend

Follow the general backend guide in `.agents/adding-backends.md`, with these audio-specific considerations:

1. **Implement the AudioBackend interface** in `pkg/backend/`:
   - `Transcribe(ctx, req) (*schema.TranscriptionResult, error)` for STT
   - `TTS(ctx, req) (io.Reader, error)` for TTS

2. **Register the backend** in `pkg/model/initializers.go` under the audio section.

3. **Handle audio format conversion** — LocalAI uses `ffmpeg` internally for format conversion. Backends should output raw WAV/PCM when possible and let LocalAI handle encoding.

4. **Temporary file handling** — Audio uploads are stored as temp files. Always use the path from `schema.AudioRequest.FilePath` rather than reading the multipart body directly.

## Common Issues

### Transcription returns empty text
- Check that the model file exists and is not corrupted
- Ensure the audio file is a supported format (use ffmpeg to convert if needed)
- For whisper.cpp, verify the model variant matches the binary (e.g., `ggml-` prefix)

### TTS produces no output / silent audio
- Piper requires both `.onnx` and `.onnx.json` files in the same directory
- Bark requires a GPU with sufficient VRAM (4GB+ recommended)
- Check backend logs: `LOG_LEVEL=debug localai`

### ffmpeg not found
- Install ffmpeg: `apt-get install ffmpeg` or `brew install ffmpeg`
- LocalAI requires ffmpeg for audio format conversion in the transcription pipeline
- Set `FFMPEG_PATH` env var if ffmpeg is in a non-standard location

### Slow transcription
- Use `whisper-1` with a smaller model (`tiny`, `base`) for faster inference
- Enable GPU acceleration: set `gpu_layers: 99` in the model config
- Consider `faster-whisper` backend for better CPU performance

## Gallery Models

Pre-configured audio models are available in the LocalAI gallery:

```bash
# List available audio models
curl http://localhost:8080/models/available | jq '.[] | select(.tags[] | contains("audio"))'

# Install whisper
curl http://localhost:8080/models/apply \
  -H "Content-Type: application/json" \
  -d '{"id": "whisper-base"}'

# Install piper TTS
curl http://localhost:8080/models/apply \
  -H "Content-Type: application/json" \
  -d '{"id": "piper-en-us-lessac-medium"}'
```

## Voice Mapping

OpenAI TTS uses named voices (`alloy`, `echo`, `fable`, `onyx`, `nova`, `shimmer`). LocalAI maps these to backend-specific voices via the model config:

```yaml
name: tts-1
backend: piper
voice_mapping:
  alloy: en_US-lessac-medium
  echo: en_US-ryan-medium
  nova: en_US-amy-medium
  # unmapped voices fall back to the default model
```

If no mapping is configured, the `voice` parameter is passed directly to the backend.
