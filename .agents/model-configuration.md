# Model Configuration Guide

This document describes how to configure models in LocalAI, including YAML configuration files, parameters, and backend-specific settings.

## Overview

LocalAI uses YAML configuration files to define model behavior, backend selection, and inference parameters. Each model can have its own configuration file stored in the models directory.

## Configuration File Structure

Model configuration files are placed in the `models/` directory and follow this naming convention:
- `models/my-model.yaml` — explicit config file
- `models/my-model.bin` — auto-detected by file extension

### Minimal Configuration

```yaml
name: my-model
backend: llama
parameters:
  model: my-model.gguf
```

### Full Configuration Reference

```yaml
# Model identity
name: my-model                    # Name used in API requests
backend: llama                    # Backend to use (llama, whisper, stablediffusion, etc.)
description: "My custom model"    # Optional description

# File paths
parameters:
  model: my-model.gguf            # Model file relative to models directory
  # For models with multiple files:
  # model: path/to/model.gguf

# Context and generation settings
context_size: 4096                # Maximum context window size
f16: true                         # Use 16-bit floats (recommended)
mmap: true                        # Memory-map model file
mlock: false                      # Lock model in RAM

# Thread and batch settings
threads: 4                        # Number of CPU threads
batch_size: 512                   # Batch size for prompt processing

# Default inference parameters
parameters:
  temperature: 0.9
  top_k: 40
  top_p: 0.95
  max_tokens: 512
  repeat_penalty: 1.1

# GPU settings
gpu_layers: 0                     # Number of layers to offload to GPU (0 = CPU only)
tensor_split: ""                  # Split layers across multiple GPUs

# Template configuration
template:
  chat: |
    {{.Input}}
  completion: |
    {{.Input}}
  chat_message: |
    {{if eq .RoleName "user"}}### Human: {{.Content}}
    {{else if eq .RoleName "assistant"}}### Assistant: {{.Content}}
    {{end}}

# Stop words — generation stops when these tokens are produced
stop_words:
  - "### Human:"
  - "<|endoftext|>"

# System prompt injected at the start of every conversation
system_prompt: "You are a helpful assistant."
```

## Backend-Specific Settings

### llama (llama.cpp)

```yaml
backend: llama
parameters:
  model: model.gguf
f16: true
gpu_layers: 35          # Set > 0 to enable GPU acceleration
rope_scaling:
  type: linear
  factor: 2.0
```

### whisper (speech-to-text)

```yaml
backend: whisper
parameters:
  model: whisper-base.en.bin
language: en
translate: false
```

### stablediffusion (image generation)

```yaml
backend: stablediffusion
parameters:
  model: sd-v1-5.bin
step: 20
cfg_scale: 7.0
width: 512
height: 512
```

## Environment Variable Overrides

Some configuration values can be overridden via environment variables at startup:

| Variable | Description |
|---|---|
| `MODELS_PATH` | Directory where model files are stored |
| `THREADS` | Default number of CPU threads |
| `CONTEXT_SIZE` | Default context window size |
| `F16` | Enable 16-bit floats globally |
| `DEBUG` | Enable verbose backend logging |

## Validating a Configuration

Use the LocalAI CLI to validate a config file before loading:

```bash
./local-ai validate --config models/my-model.yaml
```

Or check loaded models via the API:

```bash
curl http://localhost:8080/v1/models
```

## Troubleshooting

- **Model not found**: Ensure the `model` path is relative to `MODELS_PATH`.
- **Backend mismatch**: Confirm the backend name matches a compiled-in backend (see `local-ai list-backends`).
- **OOM errors**: Reduce `context_size`, `batch_size`, or `gpu_layers`.
- **Slow inference**: Increase `threads` or enable `f16: true`.

## See Also

- [Adding Backends](.agents/adding-backends.md)
- [llama.cpp Backend](.agents/llama-cpp-backend.md)
- [Debugging Backends](.agents/debugging-backends.md)
