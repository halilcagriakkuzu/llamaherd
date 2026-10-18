# LlamaHerd — Dummy GUI Design Tasks

Phase 1: **design-only dummy GUI**. The goal is a reviewable, working window with
hardcoded data — NOT a real control panel. The user reviews the running dummy
and steers the design. Only after approval do we build the real thing (phase 2).

## Dummy rules (strict) — PHASE 1 (relaxed since task 10)

- No `net/http`, no `os/exec` anywhere yet; real file I/O allowed ONLY in the
  `store` package (`configs/<alias>.json`, model scan) per task 10.
- Server state values come from `dummy/data.go` (`dummy.Server`).
- Start/stop/select only change UI state and append log lines.
- The assembled command line shown in logs is a **string only** — never executed.
- One typed config: `dummy/data.go` defines a single `Config` struct with a
  field per llama-server flag (basic + advanced). Both config forms and all
  presets read/write this one struct — NOT loose per-form maps. Command-line
  assembly (task 07) and phase-2 persistence (`configs/<alias>.json`) consume
  the same struct, so the dummy phase validates the real shape end to end.
- Single tick source: `main.go` owns ONE `QTimer` (500 ms) and broadcasts a
  tick; panels subscribe (uptime, later `/health` polling). Tasks must not
  create their own timers.

## Window layout (1100x720, title "LlamaHerd — llama.cpp control panel")

```
+--------------------------------------------------------------+
| [Left, fixed 340px]                [Right, stretch]          |
| +------------------------+     | [preset v] (Start) (Stop)   |
| | STATUS                 |     | [Save cfg] [Load cfg]       |
| | (green) RUNNING        |     | configs/<alias>.json        |
| | pid 749941             |     +-----------------------------+
| | listen 127.0.0.1:8080  |     | BASIC   | ADVANCED          |
| | wsl ip 172.24.96.2     |     | (config form, scroll)       |
| | model Qwen3.8-...      |     |                             |
| | ctx 25556  ngl 33/33   |     +-----------------------------+
| | vram 16.8G  ram 1.2G   |     | LOGS                        |
| | uptime 01:23:45        |     | (read-only, mono, ~160px)   |
| +------------------------+     +-----------------------------+
| +------------------------+     +-----------------------------+
| | MODELS                 |     | status bar: dummy mode      |
| | > Qwen3.8-...-IQ4_XS   |     |                             |
| |   TurboFCFusion-...    |     |                             |
| |   HauhauCS-Q3_K_P      |     |                             |
| |   mmproj-F16.gguf      |     |                             |
| +------------------------+     |                             |
+--------------------------------------------------------------+
```

Left column: STATUS (fixed height ~230) on top, MODELS (stretch) below.
Right column: action row (fixed), BASIC/ADVANCED tabs (stretch), LOGS (fixed ~160).
QMainWindow status bar at the very bottom.

## Files (dummy phase)

| File | Purpose |
|---|---|
| `main.go` | Wails entry: window options, asset embed |
| `app.go` | Wails-bound App: snapshot state machine, actions, log events |
| `dummy/data.go` | server state, presets, shared `Config` struct |
| `store/scan.go` | (task 10) recursive `*.gguf` scan, alias derivation, `configs/<alias>.json` load/save |
| `frontend/dist/index.html` | layout shell (STATUS, MODELS, ACTIONS, CONFIG tabs, LOGS) |
| `frontend/dist/styles.css` | styling |
| `frontend/dist/main.js` | UI logic: bindings calls, snapshot rendering, form collect |

Build: `wails build` (or `go build -tags webkit2_41,desktop,production`).
Run: `./build/bin/llamaherd`.

Build order: 01 -> 02..08 (any order) -> 09 -> phase 2: task 10 (persistence
+ scan), then 11 (ServerManager: real process control + `/health` status).

## Task table

| # | File | Task | Status |
|---|---|---|---|
| 01 | `01-main-window-shell.md` | window + layout skeleton | DONE (Wails) |
| 02 | `02-status-panel.md` | status panel with server state | DONE (Wails) |
| 03 | `03-model-list-panel.md` | model list (task 10: real scan) | DONE |
| 04 | `04-config-basic.md` | BASIC tab form | DONE (Wails) |
| 05 | `05-config-advanced.md` | ADVANCED tab form | DONE (Wails) |
| 06 | `06-model-config-presets.md` | presets, save/load | DONE (task 10: real I/O) |
| 07 | `07-actions-start-stop.md` | start/stop state machine | DONE (still dummy lifecycle) |
| 08 | `08-log-console.md` | log console | DONE (Wails) |
| 09 | `09-final-design-pass.md` | polish + design review | DONE (user approved) |
| 10 | `10-config-persistence-model-scan.md` | real configs/ + model scan | DONE |
| 11 | `11-process-control.md` | ServerManager: real process control + `/health` status | DONE |
| 12 | `12-metrics.md` | live VRAM / CPU RAM / real n_ctx in STATUS | DONE |
| 13 | `13-ai-cli-control.md` | control API + `llamaherd` CLI (open/stop/configs) for AI agents | DONE |

## Dummy data

### Server state (initial: RUNNING)

| Key | Value |
|---|---|
| status | running |
| pid | 749941 |
| binary | /home/halilcagriakkuzu/llama.cpp/build-latest/bin/llama-server |
| listen | 127.0.0.1:8080 |
| wsl ip | 172.24.96.2 |
| model | Qwen3.8-27B-UD-IQ4_XS.gguf |
| ctx | 25556 |
| ngl | 33/33 |
| vram | 16.8 GB |
| cpu ram | 1.2 GB |
| uptime | 01:23:45 |

### Models (list order)

| # | path | alias | note |
|---|---|---|---|
| 1 | /home/halilcagriakkuzu/llm/models/unsloth/Qwen3.8-27B-GGUF/Qwen3.8-27B-UD-IQ4_XS.gguf | qwen3.8-27b-local | |
| 2 | /home/halilcagriakkuzu/llm/models/Qwen3.8_Uncensored/Heretic/Fable-Distilled/Qwen3.8-27B-TurboFCFusion-735-882-Here-Uncen-NEO-CODER-MAX-LOW-MTP-IQ4_XS.gguf | qwen3.8-27b-uncensored-local | |
| 3 | /home/halilcagriakkuzu/llm/models/Qwen3.8-27B-Uncensored-HauhauCS-Aggressive-Q3_K_P.gguf | qwen3.8-27b-uncensored-hauhaucs | |
| 4 | /home/halilcagriakkuzu/llm/models/Qwen3.8_Uncensored/Heretic/Fable-Distilled/mmproj-F16.gguf | — | mmproj entry; selecting fills the `--mmproj` advanced field |

### Presets (per alias, stored in `configs/<alias>.json` in phase 2)

| alias | presets |
|---|---|
| qwen3.8-27b-local | `default` (command 1 below) |
| qwen3.8-27b-uncensored-local | `vision+mtp` (command 2), `plain` (command 3) |
| qwen3.8-27b-uncensored-hauhaucs | `default` (derived: common flags, ctx 32768, sleep 300) |

## Phase 2 reference (in progress)

Stack DECIDED (2026-10-16): **Wails v2** + vanilla frontend (`frontend/dist`,
no npm); therecipe/qt removed. Qt widget specs in tasks 01–05 read as their
HTML/form equivalents. See task 10 spec for the implemented persistence + scan
slice.

```go
type ServerManager interface {
    Start(cfg dummy.Config) error
    Stop() error
    Health() Status        // starting | running | stopped | crashed
    Logs() <-chan LogLine  // streamed stdout/stderr + app lines
}
```

The UI is a thin skin over `Config` + `ServerManager`; if the UI library
changes, this layer ports unchanged.

### Server binary / models

- Server binary: `/home/halilcagriakkuzu/llama.cpp/build-latest/bin/llama-server`
- Models dir to scan: `/home/halilcagriakkuzu/llm/models/` (recursive, `*.gguf`;
  `mmproj` files are vision sidecars, not loadable models)

### Real status

HTTP on 127.0.0.1:8080 — `/health`, `/props`, `/v1/models`, `/metrics`.
Polled on the single tick timer from `main.go`; timeout = not running.

### Process lifecycle (task 10+)

- Launch: `exec.CommandContext` with the child's `Stdout`/`Stderr` piped into
  the log console (raw server output visually distinct from app lines,
  e.g. gray prefix `server:`).
- Stop: SIGTERM → wait 3s → SIGKILL. Disable Start while starting; disable
  Stop while starting/stopped.
- Crash detection: non-zero exit or port never opens → status `crashed exit N`
  + a log line; do NOT silently revert UI to stopped.

### Status transitions (extend the 2-state dummy)

`stopped → starting → running → stopped`, plus `crashed` from running:

| Transition | Trigger | Log |
|---|---|---|
| starting | Start clicked, process spawned, `--model` resolved | `starting llama-server: <command line>` |
| running | first successful `/health` response | `ready in <n>s` |
| stopped | Stop clicked → clean exit | `stopped` |
| crashed | non-zero exit / never healthy | `crashed: exit status N` |

The dummy (task 07) simulates the 2s `starting` state so the real state
machine and its button-disabled rules are reviewed in the dummy design pass.

### Persistence

- `configs/<alias>.json` = serialized `dummy.Config` (same struct as dummy).
  Unsaved edits → dirty flag: asterisk on the path label
  (`configs/<alias>.json *`), cleared on save/load.
- Conflict: if a server is already listening on the configured host:port,
  offer Stop first instead of failing the launch.

### User's real reference commands (the form must cover this flag superset)

```bash
# 1 — qwen3.8-27b-local (UD-IQ4_XS)
llama-server --model .../Qwen3.8-27B-UD-IQ4_XS.gguf --alias qwen3.8-27b-local \
  --host 127.0.0.1 --port 8080 -ngl all --fit off -c 25556 -np 1 -fa on \
  -ctk q8_0 -ctv q8_0 --jinja --reasoning-format deepseek \
  --spec-type draft-mtp --spec-draft-n-max 3 --perf --sleep-idle-seconds 300

# 2 — qwen3.8-27b-uncensored-local (TurboFCFusion, vision + MTP)
llama-server --model .../Qwen3.8-27B-TurboFCFusion-...-IQ4_XS.gguf \
  --mmproj .../mmproj-F16.gguf --no-mmproj-offload \
  --alias qwen3.8-27b-uncensored-local --host 127.0.0.1 --port 8080 \
  -ngl all --fit off -c 32768 -np 1 -fa on -ctk q8_0 -ctv q8_0 --jinja \
  --reasoning-format deepseek --spec-type draft-mtp --spec-draft-n-max 2 \
  --spec-draft-ngl all --spec-draft-type-k q8_0 --spec-draft-type-v q8_0 \
  --cache-ram 4096 --sleep-idle-seconds 5

# 3 — qwen3.8-27b-uncensored-local (TurboFCFusion, plain)
llama-server --model .../Qwen3.8-27B-TurboFCFusion-...-IQ4_XS.gguf \
  --alias qwen3.8-27b-uncensored-local --host 127.0.0.1 --port 8080 \
  -ngl all --fit off -c 32768 -np 1 -fa on -ctk q8_0 -ctv q8_0 --jinja \
  --reasoning-format deepseek --sleep-idle-seconds 5
```
