# VRAM pre-flight guard

A runaway GPU allocation can take down a WSL distro or the whole desktop, so every start
(GUI, `open`, `POST /open`) is checked first: if the predicted GPU footprint exceeds
**90 % of total VRAM** (`vramBudgetFraction` in `control/vram.go`) the server is **not
started**. The log gets the full breakdown and the UI shows a banner; the CLI/API return
the same error.

## The estimate

`control/vram.go` adds up:

- **weights** on the GPU: file size × share set by `-ngl` (minus `--override-tensor` CPU parts),
- **KV cache**: context × full-attention layers × KV heads × head dim × bytes/element of
  `kCache`/`vCache`. Hybrid models (`full_attention_interval`) only cache a fraction of layers,
- **recurrent (SSM) state** of hybrid layers,
- a **compute buffer** allowance (grows with `ubatch` and context),
- the **mmproj** sidecar (unless `--no-mmproj-offload`) and a separate **draft model**.

Layer and head counts are read from the GGUF header (`store/gguf.go`); if that fails the
result falls back to a conservative file-size estimate. With no NVIDIA GPU visible nothing
is blocked. It is a safety net, not an exact prediction.

When a model has no preset, `SafeGpuLayers` picks the largest `-ngl` that fits, so
synthesized defaults start without tuning.

## If a start is refused

Lower `basic.context`, set the KV cache to `q8_0`/`q4_0`, or reduce `basic.gpuLayers`.

## Testing without a GPU

`internal/ggufgen` writes tiny synthetic GGUF headers (with sparse padding for realistic
sizes) and `control.SetTotalVramForTesting` fakes the card size — see `control/vram_test.go`.
