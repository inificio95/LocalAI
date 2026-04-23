# Fine-Tuning and Training in LocalAI

This guide covers how LocalAI supports fine-tuning workflows, LoRA adapters, and training-related configurations for compatible backends.

## Overview

LocalAI supports loading pre-trained models with LoRA (Low-Rank Adaptation) adapters and, for some backends, initiating fine-tuning jobs. The primary backends with training/adapter support are:

- **llama.cpp** — supports LoRA adapter loading at inference time
- **transformers** (via Python backends) — supports full fine-tuning and PEFT/LoRA
- **exllama / exllamav2** — supports LoRA adapter merging

---

## LoRA Adapters

### What is LoRA?

LoRA (Low-Rank Adaptation) allows you to apply a small set of trained weights on top of a frozen base model. This is useful for:

- Domain-specific fine-tuning without retraining the full model
- Reducing memory requirements compared to full fine-tuning
- Swapping task-specific behavior at runtime

### Loading a LoRA Adapter (llama.cpp)

In your model YAML configuration:

```yaml
name: my-finetuned-model
backend: llama-cpp
parameters:
  model: models/base-llama-7b.gguf

# LoRA adapter path (relative to models directory)
lora_adapter: lora/my-domain-adapter.bin

# LoRA scaling factor (default: 1.0)
lora_scale: 0.8

# Optional: base model path if adapter was trained on a different quant
lora_base: models/base-llama-7b-f16.gguf
```

### Multiple Adapters

Some backends support stacking multiple LoRA adapters:

```yaml
lora_adapters:
  - path: lora/adapter-domain.bin
    scale: 0.7
  - path: lora/adapter-style.bin
    scale: 0.5
```

> **Note:** Multi-adapter support depends on the backend. Check backend-specific docs.

---

## Fine-Tuning via Python Backends

### Prerequisites

- A Python backend must be configured and running
- Sufficient VRAM (recommended: 24GB+ for 7B models with QLoRA)
- A dataset in JSONL format

### Dataset Format

LocalAI expects fine-tuning datasets in the following JSONL format:

```jsonl
{"instruction": "Summarize the following text.", "input": "The quick brown fox...", "output": "A fox jumps over a dog."}
{"instruction": "Translate to French.", "input": "Hello, world!", "output": "Bonjour, le monde!"}
```

Alternatively, the conversational format:

```jsonl
{"messages": [{"role": "user", "content": "What is 2+2?"}, {"role": "assistant", "content": "4"}]}
```

### Initiating a Fine-Tuning Job (API)

LocalAI exposes a fine-tuning endpoint compatible with the OpenAI API:

```bash
# Upload training file
curl -X POST http://localhost:8080/v1/files \
  -H "Authorization: Bearer $LOCALAI_API_KEY" \
  -F "purpose=fine-tune" \
  -F "file=@dataset.jsonl"

# Start fine-tuning job
curl -X POST http://localhost:8080/v1/fine_tuning/jobs \
  -H "Authorization: Bearer $LOCALAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "training_file": "file-abc123",
    "model": "llama-3-8b",
    "hyperparameters": {
      "n_epochs": 3,
      "batch_size": 4,
      "learning_rate_multiplier": 1.0
    }
  }'

# Check job status
curl http://localhost:8080/v1/fine_tuning/jobs/ftjob-xyz789 \
  -H "Authorization: Bearer $LOCALAI_API_KEY"
```

### Hyperparameter Reference

| Parameter | Default | Description |
|---|---|---|
| `n_epochs` | 3 | Number of training epochs |
| `batch_size` | 4 | Training batch size |
| `learning_rate_multiplier` | 1.0 | Multiplier applied to base LR |
| `lora_r` | 16 | LoRA rank |
| `lora_alpha` | 32 | LoRA alpha scaling |
| `lora_dropout` | 0.05 | LoRA dropout rate |
| `use_4bit` | true | Enable QLoRA (4-bit quantization) |

---

## Output and Model Registration

After a fine-tuning job completes, the adapter or merged model is saved to the `models/` directory. To use it:

1. The job result will include the output model name (e.g., `llama-3-8b:ft-2024-01-15`)
2. A YAML config is auto-generated in the `models/` directory
3. The model is immediately available via the API

To manually register the output:

```yaml
# models/my-finetuned.yaml
name: my-finetuned
backend: llama-cpp
parameters:
  model: models/llama-3-8b.gguf
lora_adapter: lora/ft-2024-01-15.bin
```

---

## Troubleshooting

### Out of Memory During Training

- Enable 4-bit quantization (`use_4bit: true`)
- Reduce `batch_size` to 1 or 2
- Use gradient checkpointing (enabled by default)
- Reduce `lora_r` to 8

### Adapter Not Loading

- Ensure the adapter was trained on the same base model architecture
- Check that `lora_adapter` path is relative to the `models/` directory
- Verify the adapter file format matches the backend expectation (`.bin` for llama.cpp, `.safetensors` for transformers)

### Training Loss Not Decreasing

- Verify dataset format is correct
- Try increasing `learning_rate_multiplier` to 2.0
- Ensure dataset has at least 100 examples for meaningful fine-tuning

---

## Related Documentation

- [Model Configuration](.agents/model-configuration.md)
- [Adding Backends](.agents/adding-backends.md)
- [llama.cpp Backend](.agents/llama-cpp-backend.md)
- [API Endpoints and Auth](.agents/api-endpoints-and-auth.md)
