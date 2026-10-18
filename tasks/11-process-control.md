# 11 — Phase 2: ServerManager (real process control)

Status: DONE
Depends on: 10

## Goal

Start/Stop buttons control the real `llama-server` process; status shows real
lifecycle; server output streams into LOGS.

## Spec

### New `control` package (`control/server.go`) — UI-agnostic

- `Manager` — owns at most one child process.
  - `Start(cfg dummy.Config) error`: refuses if the configured host:port is
    already listening (port-conflict guard, README rule); builds
    `exec.CommandContext` from `BuildArgs(cfg)`; stdout/stderr piped line-by-
    line to `onLog("server: ...")` callbacks (gray lines in the UI).
  - Graceful stop: `Cancel = SIGTERM`, `WaitDelay = 3s` → SIGKILL fallback.
  - Exit handling: clean exit after Stop → `stopped`; unexpected exit →
    `crashed exit N` (never silently reverting to stopped).
  - Health: poll `http://<host>:<port>/health` every 500ms for up to 60s;
    first 200 response flips `starting → running` and logs `ready in <n>s`.
  - `AttachExternal(host, port)`: at boot, if a `/health` listener already
    answers, state becomes `running` (external) — Stop refuses to kill a
    process LlamaHerd did not spawn.
- State machine per README: `stopped → starting → running → stopped`,
  plus `crashed` from running/start/starting.

### App integration

- `app.go` delegates lifecycle to `Manager`; snapshots pull real PID/uptime;
  listen comes from the config; VRAM/RAM rows show real or `—` until task 12
  (metrics parsing).
- Boot: config-dir check + binary existence logged; external attach attempt.
- Dummy lifecycle code and CrashDemo removed.

## Acceptance

- Start launches the real binary; window flips starting → running with the
  server's own log lines in LOGS (gray).
- Stop terminates the process cleanly (SIGTERM); crash path shows
  `crashed exit N`.
- Port busy → clear error log, UI stays stopped.
- Launching with an already-running external server attaches instead of
  failing.
