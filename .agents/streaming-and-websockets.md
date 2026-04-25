# Streaming and WebSockets in LocalAI

This guide covers how to implement and work with streaming responses and WebSocket connections in LocalAI.

## Overview

LocalAI supports Server-Sent Events (SSE) for streaming completions, compatible with the OpenAI streaming API. This allows clients to receive tokens as they are generated rather than waiting for the full response.

## Streaming Architecture

### How Streaming Works

1. Client sends a request with `"stream": true`
2. The handler sets up a channel to receive tokens
3. The backend generates tokens and sends them to the channel
4. The handler writes each token as an SSE event
5. The stream is terminated with a `[DONE]` message

### SSE Format

Each streamed chunk follows the OpenAI format:

```
data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: [DONE]
```

## Implementing a Streaming Handler

### Basic Streaming Endpoint

```go
func (h *Handler) streamCompletion(c *fiber.Ctx, req *schema.OpenAIRequest, model *config.BackendConfig) error {
    // Set SSE headers
    c.Set("Content-Type", "text/event-stream")
    c.Set("Cache-Control", "no-cache")
    c.Set("Transfer-Encoding", "chunked")
    c.Set("Connection", "keep-alive")

    tokenChan := make(chan schema.OpenAIResponse)
    errChan := make(chan error, 1)

    go func() {
        defer close(tokenChan)
        if err := h.backend.GenerateStream(req, tokenChan); err != nil {
            errChan <- err
        }
    }()

    c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
        for {
            select {
            case chunk, ok := <-tokenChan:
                if !ok {
                    // Stream finished
                    fmt.Fprintf(w, "data: [DONE]\n\n")
                    w.Flush()
                    return
                }
                data, err := json.Marshal(chunk)
                if err != nil {
                    log.Error().Err(err).Msg("failed to marshal stream chunk")
                    return
                }
                fmt.Fprintf(w, "data: %s\n\n", data)
                w.Flush()
            case err := <-errChan:
                log.Error().Err(err).Msg("stream generation error")
                return
            }
        }
    }))

    return nil
}
```

### Detecting Stream Requests

```go
func handleCompletionRequest(c *fiber.Ctx) error {
    var req schema.OpenAIRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
    }

    if req.Stream {
        return streamCompletion(c, &req)
    }
    return regularCompletion(c, &req)
}
```

## Backend Streaming Support

### Implementing GenerateStream in a Backend

Backends that support streaming must implement the token channel pattern:

```go
func (b *MyBackend) GenerateStream(req *schema.OpenAIRequest, tokenChan chan<- schema.OpenAIResponse) error {
    // Set up the callback that fires for each token
    tokenCallback := func(token string, usage backend.TokenUsage) bool {
        chunk := schema.OpenAIResponse{
            ID:      req.ID,
            Created: time.Now().Unix(),
            Model:   req.Model,
            Object:  "chat.completion.chunk",
            Choices: []schema.Choice{
                {
                    Index: 0,
                    Delta: &schema.Message{
                        Role:    "assistant",
                        Content: token,
                    },
                    FinishReason: "",
                },
            },
        }
        tokenChan <- chunk
        return true // return false to stop generation
    }

    return b.model.Predict(req.Prompt, tokenCallback)
}
```

## Client-Side Usage

### JavaScript Example

```javascript
const response = await fetch('/v1/chat/completions', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    model: 'my-model',
    stream: true,
    messages: [{ role: 'user', content: 'Hello!' }]
  })
});

const reader = response.body.getReader();
const decoder = new TextDecoder();

while (true) {
  const { done, value } = await reader.read();
  if (done) break;

  const lines = decoder.decode(value).split('\n');
  for (const line of lines) {
    if (line.startsWith('data: ') && line !== 'data: [DONE]') {
      const chunk = JSON.parse(line.slice(6));
      const content = chunk.choices[0]?.delta?.content || '';
      process.stdout.write(content);
    }
  }
}
```

### Python Example with openai SDK

```python
import openai

client = openai.OpenAI(base_url="http://localhost:8080/v1", api_key="sk-xxx")

stream = client.chat.completions.create(
    model="my-model",
    messages=[{"role": "user", "content": "Tell me a story"}],
    stream=True,
)

for chunk in stream:
    content = chunk.choices[0].delta.content
    if content:
        print(content, end="", flush=True)
```

## Cancellation and Timeouts

### Handling Client Disconnects

LocalAI uses Fiber's context to detect client disconnects:

```go
// Check if client disconnected before sending next chunk
if c.Context().Done() != nil {
    select {
    case <-c.Context().Done():
        log.Debug().Msg("client disconnected, stopping stream")
        return
    default:
    }
}
```

### Configuring Stream Timeouts

In your model configuration (`models/my-model.yaml`):

```yaml
name: my-model
backend: llama-cpp
parameters:
  model: my-model.gguf
# Timeout for streaming responses (seconds)
timeout: 600
```

## Troubleshooting Streaming Issues

### Stream Cuts Off Early
- Check `timeout` setting in model config — default may be too short for long responses
- Ensure your reverse proxy (nginx, caddy) has streaming/buffering disabled
- For nginx: add `proxy_buffering off;` and `proxy_read_timeout 600;`

### Tokens Arriving in Batches Instead of One-by-One
- Some backends buffer output; check backend-specific streaming settings
- For llama.cpp, ensure `n_keep` and batch size are configured appropriately
- Verify the client is flushing the write buffer after each SSE event

### Missing `[DONE]` Terminator
- Always send `data: [DONE]\n\n` after the last chunk
- Some clients hang waiting for this terminator
- Ensure error paths also close the stream cleanly

## Related Files

- `api/openai/chat.go` — Chat completion handler with streaming
- `api/openai/completion.go` — Text completion handler with streaming  
- `pkg/schema/openai.go` — Request/response schema definitions
- `pkg/backend/` — Backend interface including streaming methods
