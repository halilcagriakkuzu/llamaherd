# Architecture

LlamaHerd is one Go binary with two entry modes that share a single controller.

```
 GUI (no args) ─►  frontend/dist (vanilla HTML/JS/CSS)
                        │ window.go.main.App.*  /  EventsOn("state"|"log")
                        ▼
 CLI / API     ─►  app.go  (Wails-bound App — the snapshot state machine)
 (list, open,           │
  daemon, …)            ├─ settings/  paths, env vars, config file, HF token
      │ HTTP            ├─ store/     GGUF parse, model scan, configs/library
      ▼ 127.0.0.1:8517  ├─ control/   llama-server lifecycle, metrics, VRAM
   api/  ──────────────►├─ doctor/    environment checks
                        └─ hf/        Hugging Face catalog + downloads
                        │ exec
                        ▼
                  llama-server (llama.cpp)
```

## Key ideas

- **One controller.** `app.go` holds the only mutable state (selected model,
  current config, running process). The GUI, the CLI and the HTTP API all reach
  it through the same methods, so they never diverge.
- **Snapshot + events.** The App exposes a full `Snapshot` JSON. The GUI calls
  `GetState()` once, then receives a fresh snapshot on the `state` event and log
  lines on the `log` event. All emissions go through one pump goroutine because
  the Wails runtime is not safe to call from arbitrary goroutines.
- **Loopback API.** `api/` serves the same `*App` over `127.0.0.1:8517`, so an
  agent or the CLI drives exactly what the GUI shows. The CLI (`cli.go`) is a
  thin HTTP client; if nothing is listening it spawns a headless daemon
  (`llamaherd daemon`), which runs the same core without a window.
- **Real I/O lives below the UI.** `net/http` and `os/exec` appear only in
  `control/`, `api/`, `hf/`, `doctor/` and `settings/` — never in the dummy/UI
  layer.

## Package map

| Package | Responsibility |
| --- | --- |
| `main` (`app*.go`, `cli.go`) | Wails bindings, snapshot state machine, CLI/daemon |
| `dummy/` | Shared `Config` struct + built-in sample data |
| `settings/` | Path/env/config resolution, WSL detection, HF token |
| `store/` | GGUF header parse, recursive model scan, configs/library persistence |
| `control/` | `llama-server` lifecycle, metrics, VRAM pre-flight guard |
| `doctor/` | Environment checks for CLI/API/GUI |
| `hf/` | Hugging Face Hub client, catalog, resumable verified downloads |
| `api/` | Loopback control API (optional bearer token) |
| `frontend/dist/` | Vanilla HTML/JS/CSS UI (no framework, no build step) |
| `tasks/` | Original design task specs |

## Adding a preset / model

- **Preset**: add an entry to a `configs/<alias>.json` (`AliasFile` in
  `store/scan.go`) or the built-in table in `dummy/data.go`; presets are full
  basic+advanced field sets (`dummy.Config`).
- **Model**: drop the `*.gguf` under the models directory and rescan; a
  `library/<id>.json` meta (`store.Meta`) records its category, mmproj and MTP
  sidecars. See the README's Configuration section.
