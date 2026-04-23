# Model Gallery API

This document describes how LocalAI interacts with the model gallery system, including browsing, installing, and managing gallery models via the API.

## Overview

The gallery system allows users to discover and install pre-configured models from remote or local galleries. Each gallery entry defines a model configuration, download URLs, and optional prompt templates.

## Key Data Structures

### GalleryModel

```go
type GalleryModel struct {
    ID          string            `json:"id" yaml:"id"`
    Name        string            `json:"name" yaml:"name"`
    Description string            `json:"description" yaml:"description"`
    License     string            `json:"license" yaml:"license"`
    URLs        []string          `json:"urls" yaml:"urls"`
    Icon        string            `json:"icon" yaml:"icon"`
    Tags        []string          `json:"tags" yaml:"tags"`
    Overrides   map[string]interface{} `json:"overrides" yaml:"overrides"`
    AdditionalFiles []GalleryFile  `json:"additional_files" yaml:"additional_files"`
}
```

### GalleryFile

```go
type GalleryFile struct {
    Filename string `json:"filename" yaml:"filename"`
    SHA256   string `json:"sha256" yaml:"sha256"`
    URI      string `json:"uri" yaml:"uri"`
}
```

## API Endpoints

### List Available Models

```
GET /models/available
```

Returns all models from all configured galleries.

**Response:**
```json
[
  {
    "id": "TheBloke/Mistral-7B-v0.1-GGUF/mistral-7b-v0.1.Q4_K_M.gguf",
    "name": "mistral-7b",
    "description": "Mistral 7B Q4_K_M quantization",
    "tags": ["llm", "mistral", "7b"],
    "license": "Apache 2.0"
  }
]
```

### Install a Model

```
POST /models/apply
```

**Request Body:**
```json
{
  "id": "TheBloke/Mistral-7B-v0.1-GGUF/mistral-7b-v0.1.Q4_K_M.gguf",
  "name": "my-mistral",
  "overrides": {
    "parameters": {
      "model": "mistral-7b-v0.1.Q4_K_M.gguf"
    }
  }
}
```

**Response:**
```json
{
  "uuid": "abc123",
  "status": "processing"
}
```

The `uuid` can be used to poll the job status.

### Check Job Status

```
GET /models/jobs/:uuid
```

**Response:**
```json
{
  "uuid": "abc123",
  "status": "downloading",
  "progress": 42.5,
  "error": ""
}
```

Possible status values: `waiting`, `processing`, `downloading`, `done`, `error`

### Delete a Model

```
POST /models/delete
```

**Request Body:**
```json
{
  "name": "my-mistral"
}
```

## Gallery Configuration

Galleries are configured in the LocalAI startup flags or environment:

```bash
# Single gallery
--galleries '[{"name":"localai","url":"github:mudler/LocalAI/gallery/index.yaml"}]'

# Environment variable
GALLERIES='[{"name":"localai","url":"github:mudler/LocalAI/gallery/index.yaml"}]'
```

The `url` field supports:
- `github:owner/repo/path/to/file.yaml` — fetches from GitHub
- `https://example.com/gallery.yaml` — fetches from a raw URL
- `file:///local/path/gallery.yaml` — reads from local filesystem

## Writing a Custom Gallery Index

A gallery index YAML file lists available models:

```yaml
- url: "github:mudler/LocalAI/gallery/mistral.yaml"
  name: "mistral-7b"
  description: "Mistral 7B"
  icon: "https://example.com/icon.png"
  license: "Apache 2.0"
  tags:
    - llm
    - mistral
```

Each referenced YAML defines the full model spec:

```yaml
name: mistral-7b
urls:
  - https://huggingface.co/TheBloke/Mistral-7B-v0.1-GGUF/resolve/main/mistral-7b-v0.1.Q4_K_M.gguf
files:
  - filename: mistral-7b-v0.1.Q4_K_M.gguf
    sha256: "abc123..."
    uri: https://huggingface.co/TheBloke/Mistral-7B-v0.1-GGUF/resolve/main/mistral-7b-v0.1.Q4_K_M.gguf
overrides:
  parameters:
    model: mistral-7b-v0.1.Q4_K_M.gguf
  backend: llama-cpp
  context_size: 4096
```

## Overrides

The `overrides` field merges with the base model configuration, allowing users to customize:

- `backend` — force a specific backend (e.g., `llama-cpp`, `whisper`, `stablediffusion`)
- `parameters.model` — the model filename to load
- `context_size` — context window size
- `f16` — use float16 precision
- `threads` — number of CPU threads
- `gpu_layers` — layers to offload to GPU

See [model-configuration.md](model-configuration.md) for the full list of supported fields.

## Error Handling

| Error | Cause | Resolution |
|-------|-------|------------|
| `gallery not found` | Gallery URL is unreachable | Check network and gallery URL config |
| `model already exists` | Model name conflicts | Use a different `name` in the install request |
| `sha256 mismatch` | Downloaded file is corrupt | Retry the install; check the gallery definition |
| `unsupported url scheme` | Unknown URL prefix | Use `github:`, `https://`, or `file://` |
