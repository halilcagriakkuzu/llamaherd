# Architecture

LlamaHerd is **one Go binary** with two entry modes that share a single controller.

```
 GUI (no args) ─►  frontend/dist (vanilla HTML/JS/CSS, embedded)
                        │ window.go.main.App.*  /  EventsOn("state" | "log")
                        ▼
 CLI / API     ─►  app*.go  (Wails-bound App — the snapshot state machine)
 (list, open,           │
  daemon, …)            ├─ settings/  paths, env vars, config file, HF token
      │ HTTP            ├─ store/     GGUF parse, model scan, configs/ + library/
      ▼ 127.0.0.1:8517  ├─ control/   llama-server lifecycle, metrics, VRAM guard, WSL bridge
   api/  ──────────────►├─ doctor/    environment checks
                        └─ hf/        Hugging Face catalog + downloads
                        │ exec
                        ▼
                  llama-server (llama.cpp)
```

## Key ideas

- **One controller.** `App` holds the only mutable state (selected model, current config,
  running process). The GUI, CLI and HTTP API reach it through the same methods, so they
  never diverge. A new capability is added once on `App`, then exposed through
  `api.Control` (+ endpoint), a CLI subcommand and the UI.
- **Snapshot + events.** `App.GetState()` returns the full `Snapshot` JSON. The GUI calls it
  once, then receives a fresh snapshot on the `state` event and log lines on the `log` event.
  Every emission goes through a single pump goroutine because the Wails runtime is not safe
  to call from arbitrary goroutines.
- **CLI = thin HTTP client.** `cli.go` talks to the loopback API; if nothing listens it spawns
  a headless daemon (`llamaherd daemon`) running the same core without a window.
- **I/O below the UI.** `net/http` and `os/exec` live in `control/`, `api/`, `hf/`,
  `doctor/`, `settings/` and `cli*.go`, never in UI-facing code.
- **Persistence is plain JSON.** `configs/<alias>.json` (presets), `library/<id>.json`
  (per-model metadata), `library/_categories.json` (empty category folders) and the user
  config file. Model paths are stored relative to the models directory.
- **Safety first.** `control/vram.go` blocks a start predicted to exceed 90 % of VRAM
  ([vram-guard.md](vram-guard.md)).
- **WSL bridge.** On Windows a `serverBin` of `wsl:<distro>:/path` is launched through
  `wsl.exe` with Windows paths translated (`control/wsl.go`).

## Package map

| Path | Responsibility |
| --- | --- |
| `main.go`, `app*.go` | window entry, Wails bindings, snapshot state machine, library editing |
| `cli*.go` | CLI subcommands and daemon |
| `api/` | loopback HTTP API (`Handler`, optional bearer token) |
| `control/` | process lifecycle, metrics (VRAM/RAM, tokens/s), WSL bridge, VRAM guard, llama.cpp updater glue |
| `store/` | GGUF header parsing, recursive scan, `configs/` and `library/` persistence, categories |
| `settings/` | path/env/config resolution, WSL detection, Hugging Face token |
| `doctor/` | environment checks (CLI, API, GUI) |
| `hf/` | Hugging Face client, hardware-scored catalog, resumable verified downloads |
| `schema/` | shared types: `Config`, `Preset`, `Model`, `Server` |
| `internal/ggufgen/` | synthetic GGUF files for tests |
| `frontend/dist/` | the UI (no framework, no build step) |
| `scripts/` | installers, llama.cpp updater, icon/Windows-resource generators, UI screenshot harness |

OS-specific code is split with build tags (`control/proc_*.go`, `doctor/disk_*.go`,
`daemon_attr_*.go`, `console_*.go`, `settings/platform.go`).

## Wails backend contract

Exported `App` methods are bound to the frontend as `window.go.main.App.<Method>`.

| Area | Methods |
| --- | --- |
| State | `GetState`, `SelectModel(index)`, `SelectPreset(name)`, `UpdateConfig(cfg)`, `SaveConfig`, `LoadConfig` |
| Server | `StartServer`, `StopServer`, `ForceStopLlama`, `Log(line)` |
| Setup | `SetPaths(serverBin, modelsDir)`, `GetDoctor`, `RunDoctor` |
| Library | `RenameModel(ref, name, alias)`, `SetModelCategory(ref, path)`, `CreateCategory(path)`, `RenameCategory(path, name)`, `DeleteCategory(path)` |
| Downloads | `GetCatalog`, `StartDownload(ref, file, vision)`, `CancelDownload`, `GetHFTokenStatus`, `SetHFToken(token)` |
| llama.cpp | `GetLlamaPanel`, `CheckLlamaUpdates`, `UpdateLlamaCpp`, `CancelLlamaUpdate` |

Events: `state` (a full `Snapshot`) and `log` (one line). Methods that can fail return an
`error`, which the frontend receives as a rejected promise. The API-facing methods
(`Open`, `Stop`, `AddPreset`, …) are the `api.Control` interface in `api/api.go`.

## Adding things

- **A preset:** add it to `configs/<alias>.json` (`store.AliasFile`) from the GUI, CLI or API.
  Models without a file get a synthesized default.
- **A model:** drop the `*.gguf` into the models folder (or download it); it is picked up on
  rescan and gets a `library/<id>.json`. mmproj and MTP sidecars next to it are attached.
- **A new `llama-server` flag:** field in `schema.Basic`/`Advanced` → `control.BuildArgs` →
  form field in `frontend/dist/index.html` (`data-key` / `data-group`) → `BASIC_KEYS` /
  `ADV_KEYS` in `main.js`.

See [`AGENTS.md`](../AGENTS.md) for conventions and how to test without a GPU.
