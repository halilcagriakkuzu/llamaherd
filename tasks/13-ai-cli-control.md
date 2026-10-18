# 13 — Phase 2: AI-agent / CLI control API

Status: DONE
Depends on: 12

## Goal

Let AI agents (opencode included) and scripts drive the app from the command
line — open/stop servers, add/update/remove presets — while ONE controller
instance owns the llama-server process: the GUI window reflects every CLI
change live.

## Spec

### Control API (`api/api.go`)

- HTTP API on `127.0.0.1:8517` (loopback ONLY; llama-server owns 8080).
  The GUI serves it in-process; `llamaherd daemon` runs the same core headless.
- Routes (JSON):
  - `GET /status` — full snapshot
  - `GET /models`, `GET /presets?alias=...`
  - `POST /open {alias?, preset?}` — select (fuzzy alias) + apply + Start
  - `POST /stop`, `POST /save`, `POST /load`
  - `POST /config/add | /config/update {alias,name,config?}`
    (config = full basic+advanced; partials merge in CLI)
  - `POST /config/remove {alias, name}`
- The App methods behind the API are the SAME ones the GUI bindings call
  (`app_api.go`), so events/emitState keep the window in sync. Headless mode
  (no Wails ctx) prints state/log lines to stdout instead.

### CLI (`cli.go`, same binary)

- No args → GUI (wails). Subcommands → `runCLI`:
  `list [substr]`, `status`, `open [alias] [--preset n]`, `stop`, `save`,
  `load`, `config add|update|remove <alias> <name> [--set basic.k=v ...]`,
  `help`, `daemon`.
- Alias matching: exact, else unique substring; ambiguity → error listing.
- `--set part.field=value` syntax: `basic.port=8090`,
  `advanced.spectype=draft-mtp`, booleans true/false; ints bare.
- If the control API is not reachable, the first ctl command spawns a
  detached headless daemon (Setsid) and retries for 15s.
- `config add` seeds from the alias's first preset when only overrides are
  given (must still end with a `basic.model`, either inherited or `--set`).

### Known caveat (documented)

One controller at a time owns the control port: if the headless daemon is
started while the GUI is already open, the GUI binds it first and the daemon
fails to take over — and the GUI then serves the API itself, so every ctl
command lands in the SAME app as the window. The daemon is only spawned when
NO other llamaherd instance listens.

## Acceptance

- `llamaherd list` / `status` work with GUI closed (headless daemon auto-spawn).
- `config add` → JSON file has the new preset; `update --set` changes it;
  `remove` deletes it (last preset protected).
- With the GUI running, `llamaherd open qwen3.8-27b-local default` starts the
  real server and the window shows live status.
