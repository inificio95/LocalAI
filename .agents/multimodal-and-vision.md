# Multimodal and Vision Support in LocalAI

This guide covers how to work with multimodal models (vision, audio, image generation) in LocalAI, including backend configuration, API usage, and adding new multimodal capabilities.

## Overview

LocalAI supports several multimodal modalities:

- **Vision / Image Understanding**: Send images alongside text prompts (e.g., LLaVA, BakLLaVA, MiniCPM-V)
- **Image Generation**: Generate images from text prompts (e.g., Stable Diffusion via `stablediffusion` backend)
- **Audio Transcription**: Transcribe audio files (e.g., Whisper)
- **Text-to-Speech**: Convert text to spoken audio (e.g., Piper, Bark)

---

## Vision / Image Understanding

### Supported Backends

| Backend       | Vision Support | Notes                          |
|---------------|---------------|--------------------------------|
| `llama-cpp`   | Yes           | Requires multimodal projector  |
| `ollama`      | Yes           | Depends on upstream model      |
| `openai`      | Yes (proxy)   | Passes through to OpenAI API   |

### Model Configuration for Vision

To enable vision support with `llama-cpp`, you need both the base model and a multimodal projector (mmproj) file.

```yaml
# models/llava-1.6.yaml
name: llava-1.6
backend: llama-cpp
parameters:
  model: llava-v1.6-mistral-7b.Q4_K_M.gguf

# Path to the multimodal projector
mmproj: llava-v1.6-mistral-7b-mmproj-model-f16.gguf

# Context size should be large enough for image tokens
context_size: 4096
```

### API Usage — Vision (Chat Completions)

Vision requests use the standard OpenAI-compatible `/v1/chat/completions` endpoint with `image_url` content parts:

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llava-1.6",
    "messages": [
      {
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "What is in this image?"
          },
          {
            "type": "image_url",
            "image_url": {
              "url": "https://example.com/image.jpg"
            }
          }
        ]
      }
    ],
    "max_tokens": 512
  }'
```

You can also pass base64-encoded images:

```json
{
  "type": "image_url",
  "image_url": {
    "url": "data:image/jpeg;base64,/9j/4AAQSkZJRgAB..."
  }
}
```

---

## Image Generation

### Supported Backends

- `stablediffusion` — Uses `go-stable-diffusion` bindings
- `diffusers` — Python-based, supports more model variants

### Model Configuration for Image Generation

```yaml
# models/stablediffusion.yaml
name: stablediffusion
backend: stablediffusion
parameters:
  model: sd-v1-5.ggml

# Image generation specific settings
stablediffusion:
  asset_dir: /models/assets
  threads: 4
```

### API Usage — Image Generation

```bash
curl http://localhost:8080/v1/images/generations \
  -H "Content-Type: application/json" \
  -d '{
    "model": "stablediffusion",
    "prompt": "A futuristic cityscape at sunset, photorealistic",
    "n": 1,
    "size": "512x512"
  }'
```

Response:

```json
{
  "created": 1699000000,
  "data": [
    {
      "url": "http://localhost:8080/generated-images/abc123.png",
      "b64_json": null
    }
  ]
}
```

To receive base64 output instead of a URL:

```json
{
  "response_format": "b64_json",
  "prompt": "..."
}
```

---

## Audio Transcription (Whisper)

### Model Configuration

```yaml
# models/whisper-1.yaml
name: whisper-1
backend: whisper
parameters:
  model: ggml-base.en.bin
```

### API Usage

```bash
curl http://localhost:8080/v1/audio/transcriptions \
  -F file=@audio.mp3 \
  -F model=whisper-1
```

---

## Text-to-Speech (TTS)

### Model Configuration (Piper)

```yaml
# models/tts-piper.yaml
name: tts-1
backend: piper
parameters:
  model: en_US-lessac-medium.onnx
```

### API Usage

```bash
curl http://localhost:8080/v1/audio/speech \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tts-1",
    "input": "Hello, this is a test of text to speech.",
    "voice": "alloy"
  }' \
  --output speech.mp3
```

---

## Troubleshooting Multimodal Issues

### Vision model returns text-only responses
- Ensure `mmproj` is set correctly in the model YAML
- Verify the mmproj file exists at the specified path
- Check logs for errors during model load: `docker logs localai 2>&1 | grep -i mmproj`

### Image generation produces blank/corrupt output
- Confirm the `asset_dir` exists and is writable
- Check available disk space and memory
- Try reducing image size (e.g., `256x256`) to rule out OOM issues

### Whisper transcription is slow
- Use a smaller model variant (e.g., `ggml-tiny.en.bin`)
- Increase thread count in model config
- Ensure audio is in a supported format (mp3, wav, m4a, webm)

### TTS produces no output
- Verify the `.onnx` model file and its associated `.onnx.json` config are both present
- Check that `espeak-ng` is installed if using Piper (required for phonemization)

---

## Adding a New Multimodal Backend

See `.agents/adding-backends.md` for the general backend addition process. For multimodal backends, additionally:

1. Implement the relevant gRPC service methods (`GenerateImage`, `AudioTranscription`, `TTS`) in your backend
2. Register the backend capabilities in `pkg/model/initializers.go`
3. Add any new config fields to `pkg/config/config.go`
4. Update the gallery model schema if new YAML fields are required
