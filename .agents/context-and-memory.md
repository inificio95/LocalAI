# Context and Memory Management in LocalAI

This guide covers how LocalAI handles context windows, conversation history, token counting, and memory management for long-running conversations.

## Overview

LocalAI provides several mechanisms for managing context:

1. **Static context windows** — fixed-size token buffers per model
2. **Sliding window truncation** — drop oldest messages when context is full
3. **Summarization hooks** — compress history via a secondary LLM call
4. **KV-cache management** — reuse cached prefixes across requests

---

## Context Window Configuration

Each model config (YAML) can specify context limits:

```yaml
name: my-model
backend: llama-cpp
parameters:
  model: models/my-model.gguf
context_size: 4096          # max tokens in context
mmap: true
```

The `context_size` field maps directly to `n_ctx` in llama.cpp and equivalent parameters in other backends.

### Default Fallback

If `context_size` is not set, LocalAI uses a backend-specific default (usually 512–2048 tokens). Always set this explicitly for production deployments.

---

## Message History and Token Counting

### How Messages Are Assembled

For `/v1/chat/completions`, LocalAI assembles the prompt from the `messages` array:

```
[system prompt] + [user/assistant turns] + [current user message]
```

Token counting is performed before sending to the backend. If the assembled prompt exceeds `context_size`, LocalAI applies the configured truncation strategy.

### Truncation Strategies

Set via model config or request parameter `truncation_strategy`:

| Strategy       | Behavior                                              |
|----------------|-------------------------------------------------------|
| `drop_oldest`  | Remove oldest non-system messages first (default)     |
| `drop_middle`  | Preserve first and last N turns, drop middle          |
| `error`        | Return HTTP 400 if context would be exceeded          |

Example config:

```yaml
truncation_strategy: drop_oldest
truncation_preserve_system: true   # never drop the system prompt
```

---

## Persistent Memory / Sessions

LocalAI does **not** persist conversation state between API calls by default — this is stateless, following the OpenAI API contract. The client is responsible for sending the full message history on each request.

### Session Caching (Experimental)

For backends that support KV-cache reuse (llama.cpp with `--cont-batching`), LocalAI can optionally cache the encoded prefix of a conversation to avoid re-encoding on every turn.

Enable via:

```yaml
session_cache: true
session_cache_ttl: 300   # seconds before cache entry is evicted
```

> **Warning:** Session caching is experimental. It increases memory usage proportionally to the number of active sessions × context size.

---

## Token Estimation

LocalAI exposes a `/tokenize` endpoint for token counting without running inference:

```http
POST /v1/tokenize
Content-Type: application/json

{
  "model": "my-model",
  "content": "Hello, how are you today?"
}
```

Response:

```json
{
  "tokens": [15496, 11, 703, 389, 345, 1909, 30],
  "count": 7
}
```

Use this to pre-flight check whether a conversation history fits within the model's context window before submitting a completion request.

---

## Summarization Hook

When `summarization_model` is configured, LocalAI will automatically summarize older conversation turns when the context approaches capacity:

```yaml
summarization_model: summarizer-model      # must be a loaded model
summarization_threshold: 0.85              # trigger at 85% context usage
summarization_prompt: |
  Summarize the following conversation concisely, preserving key facts:
  {{.History}}
```

The summary replaces the oldest N messages with a single synthetic `system` message containing the compressed history.

> **Note:** Summarization adds latency and an additional model call. Use only when long multi-turn conversations are a primary use case.

---

## Memory Usage Estimation

Approximate GPU/RAM usage per context token (varies by quantization):

| Quantization | Bytes per token (KV cache) |
|--------------|----------------------------|
| Q4_K_M       | ~0.5 KB                    |
| Q8_0         | ~1.0 KB                    |
| F16          | ~2.0 KB                    |
| F32          | ~4.0 KB                    |

For a 4096-token context with Q4_K_M: ~2 MB per active session.

---

## Debugging Context Issues

### Common Errors

- **`context length exceeded`** — reduce `max_tokens` or shorten message history
- **`prompt too long`** — assembled prompt exceeds `context_size`; increase `context_size` or enable truncation
- **Repetitive/degraded output** — model may be running near context limit; reduce history length

### Logging

Set `LOG_LEVEL=debug` to see token counts logged per request:

```
DEBUG assembled prompt tokens=3842 context_size=4096 utilization=93.8%
```

---

## Related

- [Model Configuration](.agents/model-configuration.md)
- [Streaming and WebSockets](.agents/streaming-and-websockets.md)
- [API Endpoints and Auth](.agents/api-endpoints-and-auth.md)
- [Debugging Backends](.agents/debugging-backends.md)
