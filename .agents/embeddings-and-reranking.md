# Embeddings and Reranking in LocalAI

This guide covers how to work with embedding models and reranking backends in LocalAI, including configuration, API usage, and adding new embedding backends.

## Overview

LocalAI supports generating text embeddings and reranking documents via dedicated backends. Embeddings are vector representations of text used for semantic search, clustering, and retrieval-augmented generation (RAG). Reranking scores a list of documents against a query to improve retrieval quality.

## Supported Backends

| Backend         | Embeddings | Reranking | Notes                          |
|-----------------|------------|-----------|--------------------------------|
| `llama-cpp`     | ✅          | ❌         | Via `embedding` flag           |
| `bert-embeddings` | ✅        | ❌         | Lightweight BERT-based         |
| `transformers`  | ✅          | ✅         | Full HuggingFace support       |
| `rerankers`     | ❌          | ✅         | Dedicated reranking backend    |

## Model Configuration for Embeddings

Create a YAML config file under your models directory:

```yaml
# models/my-embedder.yaml
name: my-embedder
backend: bert-embeddings
parameters:
  model: /models/all-MiniLM-L6-v2

# Required for embedding models
embeddings: true

# Optional: normalize output vectors
normalize: true
```

### llama-cpp Embedding Config

```yaml
name: llama-embed
backend: llama-cpp
parameters:
  model: /models/nomic-embed-text-v1.5.Q4_K_M.gguf
embeddings: true
# Disable context reuse for pure embedding workloads
no_mmap: false
```

## Model Configuration for Reranking

```yaml
# models/my-reranker.yaml
name: my-reranker
backend: rerankers
parameters:
  model: /models/bge-reranker-base

# Mark this model as a reranker
reranker: true
```

## API Usage

### Embeddings Endpoint

LocalAI implements the OpenAI-compatible `/v1/embeddings` endpoint.

```bash
curl http://localhost:8080/v1/embeddings \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "my-embedder",
    "input": "The quick brown fox jumps over the lazy dog"
  }'
```

Batch embeddings:

```bash
curl http://localhost:8080/v1/embeddings \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "my-embedder",
    "input": [
      "First document to embed",
      "Second document to embed",
      "Third document to embed"
    ]
  }'
```

Response format:

```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "index": 0,
      "embedding": [0.0023064255, -0.009327292, ...]
    }
  ],
  "model": "my-embedder",
  "usage": {
    "prompt_tokens": 9,
    "total_tokens": 9
  }
}
```

### Reranking Endpoint

LocalAI provides a `/v1/rerank` endpoint compatible with Cohere's rerank API:

```bash
curl http://localhost:8080/v1/rerank \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "my-reranker",
    "query": "What is the capital of France?",
    "documents": [
      "Paris is the capital of France.",
      "Berlin is the capital of Germany.",
      "Madrid is the capital of Spain."
    ],
    "top_n": 2
  }'
```

Response format:

```json
{
  "id": "rerank-abc123",
  "results": [
    {
      "index": 0,
      "relevance_score": 0.9871,
      "document": { "text": "Paris is the capital of France." }
    },
    {
      "index": 2,
      "relevance_score": 0.1243,
      "document": { "text": "Madrid is the capital of Spain." }
    }
  ],
  "meta": {
    "api_version": { "version": "1" }
  }
}
```

## Adding a New Embedding Backend

See `.agents/adding-backends.md` for the full backend registration process. Key points specific to embeddings:

1. **Implement the `Embeddings` method** in your backend's gRPC service:

```go
func (s *MyBackendServer) Embeddings(ctx context.Context, req *pb.PredictOptions) (*pb.EmbeddingResult, error) {
    // Tokenize and encode the input
    vectors, err := s.model.Encode(req.Prompt)
    if err != nil {
        return nil, fmt.Errorf("encoding failed: %w", err)
    }
    return &pb.EmbeddingResult{Embeddings: vectors}, nil
}
```

2. **Set the backend capability flag** in your model config so LocalAI routes embedding requests correctly:

```yaml
embeddings: true
```

3. **Register the backend** in `pkg/model/initializers.go` under the appropriate backend name.

## Performance Tips

- **Batching**: Always prefer batched requests over individual calls when embedding multiple documents. Most backends process batches more efficiently.
- **Parallelism**: Set `threads` in your model config to match your CPU core count for CPU-based embedding backends.
- **GPU offload**: For transformer-based backends, use `gpu_layers: -1` to offload all layers to GPU.
- **Model size vs. quality**: `all-MiniLM-L6-v2` (22M params) is fast; `bge-large-en-v1.5` (335M params) is more accurate but slower.

## Troubleshooting

**Empty or zero embeddings returned:**
- Verify `embeddings: true` is set in your model YAML.
- Check that the backend actually supports embeddings (see table above).
- Inspect backend logs with `DEBUG=true` for encoding errors.

**Dimension mismatch errors in downstream vector DB:**
- Each model produces a fixed embedding dimension. Ensure your vector store schema matches (e.g., 384 for MiniLM, 1536 for text-embedding-ada-002 compatible models).

**Slow embedding throughput:**
- Enable batching in your client.
- Consider a smaller quantized model (Q4_K_M) if using llama-cpp.
- Check `GOMAXPROCS` and backend thread settings.
