# 12 — Phase 2: real metrics (VRAM / CPU RAM / context)

Status: DONE
Depends on: 11

## Goal

The STATUS panel's VRAM, CPU RAM and Context rows show live numbers from the
actual system, not em-dashes.

## Spec

### `control/metrics.go`

- `VramMB(pid)` — `nvidia-smi --query-compute-apps=pid,used_gpu_memory` totals
  (across all visible GPUs); nvidia-smi missing/errors → 0 (UI shows —).
- `RamMB(pid)` — `/proc/<pid>/status` `VmRSS`.
- `NCtx(addr)` — one-shot parse of llama-server `/props`
  (`default_generation_settings.n_ctx`), falling back to the config value.

### `control.Manager` sampling

- After the `starting → running` flip (health poll success), a sampler
  goroutine runs every 5s while the process lives: VRAM + RSS every tick,
  `n_ctx` fetched once at the first tick. Results land in `Info`.
- On exit the metrics reset to zero.

### App snapshot

- Context row: real n_ctx when known, else the configured value.
- VRAM / CPU RAM: formatted as `16.8 GB` style; zeros/unknown → `—`.

## Acceptance

- Start a real server with `-ngl all` → VRAM row shows ~GPU usage of the
  llama-server process; CPU RAM shows its RSS; Context shows the server's
  actual n_ctx even if the form's `-c` differs.
- After Stop, VRAM/RAM return to `—`.
