# Troubleshooting Common Issues in LocalAI

This guide covers the most frequently encountered issues when running, developing, or contributing to LocalAI, along with their solutions.

---

## Model Loading Failures

### Symptom
Model fails to load with an error like `failed to load model` or `backend process exited unexpectedly`.

### Causes & Fixes

1. **Incorrect model path or permissions**
   - Ensure the model file exists at the path specified in your config.
   - Check file permissions: `chmod 644 /path/to/model.gguf`

2. **Mismatched backend**
   - Verify that the `backend` field in your model config matches the actual model format.
   - Example: a `.gguf` model should use `backend: llama-cpp`, not `gpt4all`.

3. **Insufficient memory**
   - Large models require significant RAM/VRAM. Check available resources.
   - Try reducing `context_size` or enabling `mmap: true` in your model config.

4. **Missing shared libraries**
   - Run `ldd /path/to/backend-binary` to identify missing `.so` files.
   - On CUDA builds, ensure `libcuda.so` and matching driver versions are installed.

---

## API Returns 500 or Empty Response

### Symptom
Requests to `/v1/completions` or `/v1/chat/completions` return HTTP 500 or an empty body.

### Causes & Fixes

1. **Backend not running**
   - Check LocalAI logs for lines like `[ERROR] backend process crashed`.
   - Restart LocalAI and watch for startup errors.

2. **Prompt too long for context window**
   - Reduce the input prompt length or increase `context_size` in the model config.

3. **Malformed request body**
   - Validate your JSON payload. Required fields: `model`, `messages` (chat) or `prompt` (completion).
   - Use `curl -v` to inspect the raw request/response cycle.

---

## GPU Not Being Used

### Symptom
Inference is slow and GPU utilization stays at 0%.

### Causes & Fixes

1. **LocalAI not built with GPU support**
   - Check the build tags: `go build -tags cublas ...` for NVIDIA, or `-tags clblas` for OpenCL.
   - Use the pre-built Docker image with the `-cuda` suffix for NVIDIA GPUs.

2. **`gpu_layers` not set**
   - Add `gpu_layers: 35` (or more) to your model YAML config to offload layers to GPU.

3. **Driver/CUDA version mismatch**
   - Run `nvidia-smi` to confirm the driver is loaded.
   - Ensure the CUDA toolkit version matches what LocalAI was compiled against.

---

## Audio / Whisper Transcription Fails

### Symptom
`/v1/audio/transcriptions` returns an error or empty transcript.

### Causes & Fixes

1. **Wrong audio format**
   - Whisper expects 16 kHz mono WAV. Convert with:
     ```bash
     ffmpeg -i input.mp3 -ar 16000 -ac 1 output.wav
     ```

2. **Model not configured for whisper backend**
   - Set `backend: whisper` in the model YAML and point `model` to a `.bin` whisper model file.

---

## Image Generation Returns Blank or Corrupted Images

### Symptom
Stable Diffusion endpoint returns a black image or a corrupted PNG.

### Causes & Fixes

1. **Incorrect scheduler or sampler settings**
   - Try resetting to defaults: remove custom `parameters` overrides from your config.

2. **Seed value causing deterministic failure**
   - Set `seed: -1` in the request to use a random seed.

3. **VAE model path misconfigured**
   - Ensure the `vae` path in your config points to a valid `.safetensors` or `.ckpt` VAE file.

---

## Debugging Tips

- **Enable verbose logging**: Set the environment variable `DEBUG=true` before starting LocalAI.
- **Check backend logs**: Backend-specific logs are written to stderr and captured in the main LocalAI log stream.
- **Validate model configs**: Use `yamllint` on your `.yaml` model configs to catch syntax errors.
- **Test with curl**: Always isolate API issues by testing with a minimal `curl` command before blaming the client.

```bash
curl http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model": "my-model", "messages": [{"role": "user", "content": "Hello"}]}'
```

---

## Getting Further Help

- Open an issue on [GitHub](https://github.com/mudler/LocalAI/issues) with logs and your model config (redact any API keys).
- Join the community Discord and post in `#support`.
- Search existing issues — many common problems are already documented.
