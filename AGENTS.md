# AGENTS.md

Go + Wails desktop GUI project (llama.cpp control panel).
Git repo: `git@github.com:halilcagriakkuzu/llamaherd.git` (branch `main`).

## Project

- Path (WSL): `~/workspace/LlamaHerd` (your checkout location)
- Windows UNC path (for file writes from Windows): `\\wsl.localhost\Ubuntu\home\<user>\workspace\LlamaHerd`
- Go module name: `llamaherd`
- GUI: **Wails v2** (v2.16.0 CLI, `go:build webkit2_41` on Ubuntu 24.04),
  vanilla HTML/JS/CSS frontend in `frontend/dist/` — no npm/framework
- Current state: Wails app. Task 13 (loopback control API + same-binary CLI) and
  a `doctor` environment check are done. The `hf` package + `llamaherd download`
  add Hugging Face model onboarding (hardware-scored catalog, resumable +
  sha256-verified downloads, GUI panel, auto-installed ~50k-context preset).
  Downloads auto-detect sidecars: vision `mmproj*.gguf` is attached, an embedded
  MTP (NextN) head sets the preset to `--spec-type draft-mtp`, and a separate
  `fastmtp*.gguf`/`*mtp-draft*` draft is downloaded + recorded (`Meta.MTPDraft`)
  and usable via `advanced.draftModel` (`--spec-draft-model`).
  Conventions: launch with `DISPLAY=:0 GSK_RENDERER=cairo`.
- System deps for Wails build (installed 2026-10-16):
  `libgtk-3-dev libwebkit2gtk-4.1-dev` (reinstall via
  `sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev` if ever missing)

## Environment

- OS: Ubuntu 24.04.5 LTS in WSL2 (Windows 11 host), x86_64, kernel `6.6.87.2-microsoft-standard-WSL2`
- Go: go1.26.5 linux/amd64
- Build tools present: gcc, g++, make
- Qt5 dev packages installed (Qt 5.15.13+dfsg-1ubuntu1):
  `qtbase5-dev`, `qtbase5-dev-tools`, `qt5-qmake`, `qt5-qmake-bin`
  - On Ubuntu 24.04 the qmake package is `qt5-qmake` (not `qmake`).
  - Install with: `sudo apt-get update && sudo apt-get install -y qtbase5-dev qtbase5-dev-tools qt5-qmake`
- Display: WSLg. `DISPLAY` is empty in WSL login shells; `/tmp/.X11-unix/X0` exists; no Wayland socket observed. Launch GUIs with `DISPLAY=:0`.

## Commands

- `CGO_ENABLED=1` is required (therecipe/qt is cgo-based).
- On first run, therecipe/qt downloads a `qtbox` helper binary into the project directory; `qtbox` is expected to exist and is not source.
- There is no lint, typecheck, or test suite configured. Verify by building and running.

## Wails notes

- Build: `wails build` (CLI at `~/go/bin/wails`, v2.16.0) or
  `go build -tags webkit2_41,desktop,production -o build/bin/llamaherd .`
- Frontend is prebuilt vanilla files in `frontend/dist` — no build step, and
  no `wailsjs` imports in `main.js` (uses `window.go.main.App.*` and
  `window.runtime.EventsOn` directly).
- Windows host shortcut (added 2026-10-16): `scripts/make-windows-shortcut.ps1`
  creates "LlamaHerd" shortcuts on the Win11 Desktop + Start Menu. They launch
  `C:\Program Files\WSL\wslg.exe -d Ubuntu --cd <repo> -- /bin/bash
  scripts/launch-gui.sh`, i.e. the GUI on the WSLg display with no console
  window (falls back to `wsl.exe` if `wslg.exe` is missing). `scripts/launch-gui.sh`
  cds to the repo, sets `DISPLAY`/`GSK_RENDERER`, builds `build/bin/llamaherd` if
  missing (log: `build/launch.log`), and refuses to start a second instance
  (the control API is a singleton). Icon: `scripts/make-icon.py` →
  `build/windows/icon.ico`.
- Backend contract (app.go): `GetState`, `SelectModel`, `SelectPreset`,
  `UpdateConfig`, `SaveConfig`, `LoadConfig`, `StartServer`, `StopServer`,
  `Log`; events: `state` (full snapshot), `log` (one line).
## llama.cpp reference

### VRAM pre-flight guard (added 2026-10-16)

`StartServer` refuses to launch when the predicted GPU footprint exceeds
`vramBudgetFraction` (90%) of total VRAM — a runaway allocation previously
OOM-killed the whole WSL distro (see `journalctl -b -1`). The estimate
(`control/vram.go`) reads the GGUF header (`store/gguf.go`) for layer/KV-head
counts and honours hybrid attention (`full_attention_interval`: qwen35 keeps a
KV cache on only 16/65 layers). Blocked starts log a reason and set
`Snapshot.Warning` (UI banner); `Open` (CLI/API) returns that error too.
Tune the threshold in `control/vram.go`; estimate vs. reality is approximate.
Presets (`configs/*.json` and the synthesized `defaultPreset`) are tuned to stay under the budget (e.g. UD-IQ4_XS uses
`-ngl 60`, vision+mtp `-ngl 51`); `TestAllPresetsFitBudget` enforces this.
The Q3/IQ4/UD-Q3 models also carry a `high-ctx` preset: full GPU offload
(`-ngl all`) with the largest context that still fits (q4_0 KV), e.g. Q3_K_S
~96k, UD-Q3_K_XL ~68k, Q3_K_P ~55k, UD-IQ4_XS ~16k.

- Server binary: `~/llama.cpp/build-latest/bin/llama-server`
- Models dir (scan recursively for `*.gguf`; `mmproj*.gguf` are vision sidecars, not loadable models): `~/llm/models/`
- Real status/control:
  - HTTP on `127.0.0.1:8080`: `/health`, `/props`, `/v1/models`, `/metrics`
  - Launch/kill the process; per-alias configs become real `configs/<alias>.json` file I/O
- User's real reference commands (flag superset the form must cover):

```bash
# 1 — qwen3.8-27b-local (UD-IQ4_XS)
llama-server --model ~/llm/models/unsloth/Qwen3.8-27B-GGUF/Qwen3.8-27B-UD-IQ4_XS.gguf \
  --alias qwen3.8-27b-local --host 127.0.0.1 --port 8080 -ngl all --fit off \
  -c 25556 -np 1 -fa on -ctk q8_0 -ctv q8_0 --jinja --reasoning-format deepseek \
  --spec-type draft-mtp --spec-draft-n-max 3 --perf --sleep-idle-seconds 300

# 2 — qwen3.8-27b-uncensored-local (TurboFCFusion, vision + MTP)
llama-server --model ~/llm/models/Qwen3.8_Uncensored/Heretic/Fable-Distilled/Qwen3.8-27B-TurboFCFusion-735-882-Here-Uncen-NEO-CODER-MAX-LOW-MTP-IQ4_XS.gguf \
  --mmproj ~/llm/models/Qwen3.8_Uncensored/Heretic/Fable-Distilled/mmproj-F16.gguf \
  --no-mmproj-offload --alias qwen3.8-27b-uncensored-local --host 127.0.0.1 --port 8080 \
  -ngl all --fit off -c 32768 -np 1 -fa on -ctk q8_0 -ctv q8_0 --jinja \
  --reasoning-format deepseek --spec-type draft-mtp --spec-draft-n-max 2 \
  --spec-draft-ngl all --spec-draft-type-k q8_0 --spec-draft-type-v q8_0 \
  --cache-ram 4096 --sleep-idle-seconds 5

# 3 — qwen3.8-27b-uncensored-local (TurboFCFusion, plain)
llama-server --model ~/llm/models/Qwen3.8_Uncensored/Heretic/Fable-Distilled/Qwen3.8-27B-TurboFCFusion-735-882-Here-Uncen-NEO-CODER-MAX-LOW-MTP-IQ4_XS.gguf \
  --alias qwen3.8-27b-uncensored-local --host 127.0.0.1 --port 8080 \
  -ngl all --fit off -c 32768 -np 1 -fa on -ctk q8_0 -ctv q8_0 --jinja \
  --reasoning-format deepseek --sleep-idle-seconds 5
```

## Files

- `main.go` — Wails window entry
- `app.go` — Wails-bound App (snapshot state machine)
- `hf/` — Hugging Face catalog + resumable, sha256-verified downloader
- OS-specific code is platform-split (build-tagged): `control/proc_{linux,darwin,unix,windows,other_unix}.go`
  (process listing/RSS/total RAM/kill/detach), `doctor/disk_{unix,other,windows}.go`,
  `daemon_attr_{unix,windows}.go`, and `settings/platform.go` (`IsWSL`, `LocalIPv4`).
  The whole module must keep cross-compiling: verify with
  `CGO_ENABLED=0 GOOS=windows go build ./...` and `GOOS=darwin go build ./...`.
- `schema/schema.go` — shared data types (`Config`, `Preset`, `Model`, `Server`)
- `frontend/dist/` — vanilla frontend (index.html, main.js, styles.css)
- `go.mod` / `go.sum` — module `llamaherd` + Wails v2 dependencies
- `opencode.json` — opencode project config (fish shell + searxng MCP only). The
  `llama` provider is NOT defined here: the global plugin
  `~/.config/opencode/plugins/llamacpp-discover.ts` discovers the model loaded in
  llama-server (127.0.0.1:8090) and registers it in every folder; run
  `/llama-refresh` in opencode after starting or switching a model.
