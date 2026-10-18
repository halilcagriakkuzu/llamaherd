# LlamaHerd

A desktop control panel for [llama.cpp](https://github.com/ggml-org/llama.cpp)
— manage, configure, and run local GGUF models from a single GUI (with a
matching CLI and a local control API for AI agents).

> **Status: early / work in progress.** Not yet polished for distribution;
> expect rough edges and breaking changes. See [`BACKLOG.md`](BACKLOG.md).

**In a hurry? → [`QUICKSTART.md`](QUICKSTART.md).**

<!--
Add a screenshot or short screencast here, e.g.:
![LlamaHerd screenshot](docs/screenshot.png)
-->

## What is this?

LlamaHerd is a GUI front-end that sits on top of `llama-server` from
llama.cpp. Instead of memorising dozens of command-line flags, you pick a model,
tune the settings in a form, and press **Start**. It also:

- scans a models directory (recursively) for `*.gguf` files,
- manages per-model configuration presets as real JSON files,
- launches, monitors, and stops the `llama-server` process,
- shows live status and metrics (PID, listen address, VRAM/RAM, uptime),
- guards against runaway VRAM allocation before a start,
- exposes a loopback control API and a same-binary CLI so AI agents (or
  scripts) can drive it programmatically.

## Features

- **Model browser** — recursive scan of your models folder, grouped by
  category, with vision (mmproj), MTP, and MTP draft-sidecar badges.
- **Model download** — a curated, hardware-aware catalog plus any Hugging Face
  repo/URL. Resumable downloads with size + sha256 verification and live
  progress in the GUI and CLI; the model is installed with a ~50k-context
  preset. Vision projectors and separate MTP draft sidecars (`fastmtp*.gguf`)
  are detected and downloaded too, and an embedded MTP head switches the
  generated preset to `--spec-type draft-mtp` automatically.
- **Config form** — Basic / Advanced / llama.cpp tabs covering the common
  `llama-server` flags (`--model`, `--alias`, `--host`, `--port`, `-c`, `-ngl`,
  `-fa`, `-ctk`, `-ctv`, `--jinja`, `--reasoning-format`, speculative decoding
  (`--spec-type draft-mtp|draft-simple`, `--spec-draft-model`, `--spec-draft-*`),
  vision (`--mmproj`), `--fit`, `--perf`, `--sleep-idle-seconds`, `--cache-ram`,
  `-mlock`, `-no-mmap`, `-t`, `-b`, `-ub`, `-ot`, and more).
- **Presets** — a synthesized default preset per model plus per-alias `configs/<alias>.json` files you
  can save/load from the GUI.
- **Process control** — start/stop the server, detect crash exit codes, and
  force-stop any stray `llama-server` processes (including ones LlamaHerd did
  not start).
- **VRAM pre-flight guard** — reads the GGUF header and predicts the GPU
  footprint; refuses to start if it would exceed ~90% of total VRAM.
- **Live metrics** — pulled from llama.cpp's `/metrics` and `/props`.
- **CLI + control API** — drive everything from the terminal or an agent.

## Requirements

What you need depends on **how you install LlamaHerd** — a prebuilt binary
needs almost nothing, while building from source needs a Go + C toolchain and
platform GUI libraries.

### Always required (runtime)

- **`llama-server`** built from llama.cpp, plus at least one GGUF model.
- **A supported OS**: Linux (incl. WSL2 + WSLg), Windows 11, or macOS.
- **Disk space** for the models you want to run (a GGUF model is typically
  2–40 GB; `doctor` warns below 10 GB free).
- **A GPU is recommended** (CUDA/ROCm on Linux/Windows, Metal on macOS) but not
  strictly required — CPU inference works.

### Using a prebuilt binary (recommended)

- **Nothing else to install manually** — no Go toolchain and no C compiler. The
  release archive is a compiled binary; grab it with
  `scripts/install.sh --from-release` (Linux/macOS/WSL) or
  `powershell -File scripts\install.ps1 -FromRelease` (Windows).
- The GUI uses the system webview, which ships with the OS: WebKit2GTK on
  Linux (installed by the script if missing), WebView2 on Windows 11 (present
  by default), and WebKit on macOS.

### Building from source

- **Go** (see `go.mod` for the exact toolchain version).
- **A C toolchain** (`gcc`, `g++`, `make`; MSYS2/MINGW or MSVC on Windows) —
  Wails uses cgo.
- **Wails v2** CLI (for `wails build`).
- **Platform GUI build dependencies**:
  - **Linux / WSL** (Ubuntu): `libgtk-3-dev` and `libwebkit2gtk-4.1-dev`.
  - **macOS**: Xcode Command Line Tools (`xcode-select --install`).
  - **Windows 11**: the WebView2 runtime (present by default) and a C toolchain.

## Installation

LlamaHerd is installed from the terminal. Two paths are supported:

- **Prebuilt binary** — download from GitHub Releases (no Go toolchain needed).
- **From source** — clone and build with Go (needs a C toolchain + GUI libs).

### Quick install (one-liner)

```bash
curl -fsSL https://raw.githubusercontent.com/halilcagriakkuzu/llamaherd/main/scripts/install.sh \
  | bash -s -- --from-release --repo halilcagriakkuzu/llamaherd
```

The one-liner only fetches the **installer script**. The app itself is a
**prebuilt archive** (`llamaherd_<os>_<arch>.tar.gz`) attached to a GitHub
Release by `.github/workflows/release.yml` — the script downloads, extracts and
installs that compiled binary. Notes:

- Use `bash -s --`, not `sh`: the installer uses bash features (arrays, process
  substitution).
- `--repo` (or `LLAMAHERD_REPO`, or the `BUILTIN_REPO` variable in the script)
  tells it which repo to download from. `BUILTIN_REPO` lets you drop the flag.
- Requires that a release exists. To build from source instead:
  `… | bash -s -- --source-url https://github.com/halilcagriakkuzu/llamaherd.git`.

### Linux / macOS / WSL

```bash
git clone https://github.com/halilcagriakkuzu/llamaherd.git
cd LlamaHerd
scripts/install.sh                 # build from this checkout
scripts/install.sh --from-release  # …or download a prebuilt binary
```

The installer builds or downloads `llamaherd`, installs it to `~/.local/bin`,
seeds the shipped presets into `~/.local/share/llamaherd/configs`, adds a
desktop entry + GUI launcher, and detects `llama-server` / a models directory to
write `~/.config/llamaherd/config.json` (skipped if no config exists yet — the
GUI's first-run prompt will ask).

Useful options:

| Option | Meaning |
| --- | --- |
| `--prefix DIR` | install prefix (default `~/.local`) |
| `--bin-dir DIR` / `--data-dir DIR` | override the binary / data directories |
| `--from-source` / `--from-release` | force the source or release path |
| `--repo OWNER/REPO`, `--release-tag TAG` | which GitHub release to download |
| `--with-deps` / `--no-deps` | install (or never touch) system packages |
| `--no-desktop` / `--no-config` | skip the desktop entry / config file |
| `--dry-run`, `-y` | preview actions / assume yes |

```bash
scripts/install.sh --uninstall          # remove the app, keep data + config
scripts/install.sh --uninstall --purge  # also remove data + config
```

### Windows 11

```powershell
git clone https://github.com/halilcagriakkuzu/llamaherd.git
cd LlamaHerd
# Default: install inside WSL (the GUI runs on WSLg) + Start Menu/Desktop shortcuts
powershell -ExecutionPolicy Bypass -File scripts\install.ps1

# Native llamaherd.exe (experimental):
powershell -ExecutionPolicy Bypass -File scripts\install.ps1 -Mode native -FromRelease
```

Key switches: `-Mode wsl|native`, `-Distro`, `-FromRelease`, `-Tag`, `-Repo`,
`-WithDeps`, `-NoShortcuts`, `-Uninstall`, `-Purge`.

### Prebuilt releases

Tag a commit (`git tag v0.2.0 && git push origin v0.2.0`) to run
[`.github/workflows/release.yml`](.github/workflows/release.yml), which builds
Linux (amd64/arm64), macOS (amd64/arm64) and Windows (amd64) archives and
attaches them to the GitHub Release. Asset names are stable so the installer can
always use `/releases/latest/download/<asset>`.

### Building from source (manual)

**Ubuntu / WSL2**

```bash
# GUI dependencies
sudo apt-get update
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev

# Build (Wails CLI at ~/go/bin/wails)
wails build

# …or build directly with the required tags.
# Use webkit2_40 instead of webkit2_41 on distros that only ship WebKit2GTK 4.0.
go build -tags webkit2_41,desktop,production -o build/bin/llamaherd .
```

Run the built GUI (on WSLg, set the display explicitly):

```bash
DISPLAY=:0 GSK_RENDERER=cairo ./build/bin/llamaherd
```

**Windows 11 (native)**

```powershell
# Needs Go + a C toolchain (MSYS2/MINGW64 gcc). WebView2 is required at runtime.
go build -tags desktop,production -o build\bin\llamaherd.exe .
.\build\bin\llamaherd.exe
```

**macOS**

```bash
# Needs Go + Xcode Command Line Tools (xcode-select --install).
go build -tags desktop,production -o build/bin/llamaherd .
./build/bin/llamaherd
```

The GUI-free CLI/daemon builds on any platform with no extra dependencies:

```bash
CGO_ENABLED=0 go build ./...          # compile check (all packages)
go build -o llamaherd . && ./llamaherd doctor
```

### Reproducible builds

The toolchain is pinned in the repo, so a clean checkout builds deterministically:

- **Go** — the version in `go.mod` (`go 1.25.0`); CI uses
  `go-version-file: go.mod`. The `main.version` value defaults to `dev` and is
  stamped at release time with `-ldflags "-X main.version=<tag>"`, so
  `llamaherd version` reports the release tag.
- **Wails** — `github.com/wailsapp/wails/v2 v2.16.0`, pinned in `go.mod`, with
  `go.sum` locking every transitive module. The Wails **CLI** is only needed for
  `wails build`; a plain `go build -tags desktop,production` (plus
  `webkit2_41`/`webkit2_40` on Linux) is enough.
- **Frontend** — prebuilt static files in `frontend/dist`; no npm/framework, so
  there is no separate frontend build step.
- **System libraries** — the OS webview (GTK3/WebKit2GTK on Linux, WebView2 on
  Windows 11, WebKit on macOS); see Requirements.

To reproduce a release binary locally:

```bash
go build -tags webkit2_41,desktop,production \
  -ldflags "-X main.version=$(git describe --tags --always)" -o build/bin/llamaherd .
```

### Platform support

| Platform | GUI | CLI / daemon | Notes |
| --- | --- | --- | --- |
| Linux (incl. WSLg) | yes | yes | Primary target. Needs GTK3 + WebKit2GTK. |
| Windows 11 | yes | yes | Native WebView2, or run the GUI inside WSL (WSLg). |
| macOS | yes | yes | Metal is used automatically; CUDA is NVIDIA-only. |

### Windows 11 setups

| Scenario | LlamaHerd runs in | `llama-server` | Models | Setup |
| --- | --- | --- | --- | --- |
| **A. No WSL** | Windows (native) | `llama-server.exe` on Windows | Windows folder (default `%USERPROFILE%\LlamaHerd\models`) | `install.ps1 -Mode native`; point `serverBin` at the `.exe` |
| **B. WSL, all inside WSL** | WSL (GUI on WSLg) | Linux build in WSL | WSL folder (`~/LlamaHerd/models`) | `install.ps1` (default mode) — fastest loads |
| **C. Windows app + WSL server** | Windows (native) | Linux build in WSL | Windows *or* WSL folder | set `serverBin` to `wsl:<distro>:/path/to/llama-server` |

Scenario C details:

- `LLAMAHERD_SERVER_BIN=wsl:Ubuntu:/home/me/llama.cpp/build/bin/llama-server`
  (or `serverBin` in `%APPDATA%\llamaherd\config.json`; `wsl:/path` uses the default distro).
  The server is launched through `wsl.exe`, paths are translated for you
  (`C:\x` → `/mnt/c/x`, `\\wsl.localhost\Ubuntu\home\me\m` → `/home/me/m`) and
  Stop terminates the process inside WSL.
- **Models on a Windows drive** work but load slowly (WSL reads them over
  `/mnt/c`). **Models inside WSL are fast**: set the models directory to the UNC
  path `\\wsl.localhost\Ubuntu\home\me\models` — the Windows app scans it and
  the server reads them natively. `llamaherd doctor` warns about the slow case.
- `llama-server` must listen on `127.0.0.1`; WSL2's localhost forwarding makes it
  reachable from Windows.

Known limitations (see `BACKLOG.md` section 5):

- **Model directory & paths** are stored relative to `LLAMAHERD_MODELS_DIR` and
  work on every OS, but a config written on one OS may point at a
  friend-owned absolute path if it was saved outside the models dir.
- **WSL IP** is shown only inside WSL (the address the Windows host uses to
  reach the distro); it is blank elsewhere. `llama-server` binds `127.0.0.1`
  by default, so cross-boundary access needs mirrored networking or a
  non-loopback `--host`.
- **llama-server process discovery**: on Linux via `/proc`, on macOS via
  `pgrep`/`ps`, on Windows via `tasklist` (command lines / model / listen
  columns are only populated on Linux/macOS).
- **Process stop**: graceful `SIGTERM` on Linux/macOS; Windows has no SIGTERM,
  so the stop is a `TerminateProcess` (hard kill).
- **Per-process CPU RAM** (`VmRSS`/`ps rss`/working set) is reported on
  Linux/macOS and via `tasklist` on Windows.
- **llama.cpp update & GPU checks** shell out to `bash` scripts and
  `nvidia-smi`; the update path is Unix-only (CPU/GPU inference still works
  everywhere, and a CUDA GPU is optional).


## First run

1. Launch the app.
2. Select your models folder (where your `*.gguf` files live) — or press
   **Download a starter model…** to fetch a small model from Hugging Face.
3. Pick a model from the list.
4. Choose a preset (or tune the form), then press **Start**.
5. Watch the **Status** panel and **Logs** for progress and health.

> The **Models → ＋ Download a model** panel ranks the curated catalog against
> your GPU/RAM and shows each model's VRAM/RAM requirement. You can also paste
> any `owner/name` repo or `huggingface.co` URL. Downloads resume automatically
> and are verified (size + sha256) before being installed.

## Usage

### GUI

- **Left column** — server status (PID, listen address, VRAM/RAM, uptime) and
  the model browser.
- **Right column** — the config form (Basic / Advanced / llama.cpp tabs), the
  command-line preview, the VRAM estimate, and the log console.
- **Start / Stop** — control the `llama-server` process.
- **Save cfg / Load cfg** — persist or restore the current model's config.
- **llama.cpp tab** — inspect the installed `llama-server` build and any
  running processes; optionally update & rebuild llama.cpp.

### CLI

The same binary doubles as a CLI (a singleton control API on
`127.0.0.1:8517` backs both):

```bash
llamaherd list
llamaherd status
llamaherd open <alias>
llamaherd stop
llamaherd save
llamaherd load
llamaherd config add|update|remove
llamaherd daemon
llamaherd doctor [--json]
llamaherd download list
llamaherd download <repo> [<file>] [--vision <file>]   # resumable, verified
llamaherd download status | cancel
llamaherd download token [<token>|--clear]
```

> Exact arguments are being finalised — see [`BACKLOG.md`](BACKLOG.md).

### Control API

LlamaHerd exposes a loopback HTTP control API (default `127.0.0.1:8517`, port
via `LLAMAHERD_API_PORT`) so AI agents and scripts can drive it. It is bound to
localhost only. It is unauthenticated by default; set `LLAMAHERD_API_TOKEN` (or
`apiToken` in the config file) to require a bearer token. The bundled CLI sends
it automatically; other clients pass `Authorization: Bearer <token>` or
`?token=<token>`.

| Method & path | Body / query | Returns |
| --- | --- | --- |
| `GET /status` | — | full UI snapshot (same JSON the GUI renders) |
| `GET /models` | — | `{models: [...]}` |
| `GET /presets` | `?alias=<a>` | `{alias, presets: [...]}` |
| `GET /catalog` | — | hardware-scored model catalog |
| `GET /download` | — | live downloader state |
| `GET /doctor` | — | environment report (`llamaherd doctor --json`) |
| `POST /open` | `{alias, preset}` | start `llama-server` |
| `POST /stop` | — | stop the managed server |
| `POST /save` / `POST /load` | — | persist / restore the current preset |
| `POST /config/add` \| `update` \| `remove` | `{alias, name, config?}` | edit presets |
| `POST /download` | `{ref, repo, file, vision}` | start a HF download |
| `POST /download/cancel` | — | cancel the active download |
| `POST /token` | `{token}` | store/clear the HF token |

Mutating calls return `{"ok":true}` or `{"ok":false,"error":"..."}`. Example:

```bash
curl -s 127.0.0.1:8517/status | jq '.status, .alias'
curl -s -X POST 127.0.0.1:8517/open \
  -H 'content-type: application/json' -d '{"alias":"qwen3.8-27b-local"}'
```

Start it headless with `llamaherd daemon`; the CLI (`llamaherd status`, …)
talks to the same API, so GUI and CLI always share one controller.

## Configuration

Configuration is resolved by the `settings` package with this precedence for
every value:

1. an environment variable (always wins)
2. the user config file (JSON)
3. auto-discovery on this machine
4. a built-in default

### User config file

Written by the first-run setup prompt and editable by hand. Location:

| OS | Path |
| --- | --- |
| Linux/BSD | `$XDG_CONFIG_HOME/llamaherd/config.json`, else `~/.config/llamaherd/config.json` |
| Windows | `%APPDATA%\llamaherd\config.json` |
| macOS | `~/Library/Application Support/llamaherd/config.json` |

```json
{
  "serverBin": "/opt/llama.cpp/build/bin/llama-server",
  "llamaDir": "/opt/llama.cpp",
  "modelsDir": "/data/models",
  "apiPort": "8517",
  "apiToken": ""
}
```

### Environment variables

The same settings can be set as environment variables, which override the
config file:

| Variable | Purpose |
| --- | --- |
| `LLAMAHERD_SERVER_BIN` | full path to `llama-server` |
| `LLAMAHERD_LLAMA_DIR` | the llama.cpp checkout directory |
| `LLAMAHERD_LLAMA_BUILD_DIR` | build subdirectory (e.g. `build-latest`) |
| `LLAMAHERD_MODELS_DIR` | directory scanned for `*.gguf` (default `~/LlamaHerd/models`) |
| `LLAMAHERD_API_PORT` | loopback control-API port (default `8517`) |
| `LLAMAHERD_API_TOKEN` | bearer token required by the control API (empty = open) |
| `LLAMAHERD_CONFIG` | override the config-file path |
| `LLAMAHERD_DATA_DIR` | per-user data dir for `configs/` + `library/` |
| `LLAMAHERD_CONFIGS_DIR` | per-alias preset directory |
| `LLAMAHERD_LIBRARY_DIR` | per-model metadata directory |
| `LLAMAHERD_MIN_LLAMA_BUILD` | minimum llama.cpp build `doctor` accepts |
| `HF_TOKEN` / `HUGGING_FACE_HUB_TOKEN` | Hugging Face token (see below) |
| `HF_ENDPOINT` | Hugging Face mirror endpoint |

```bash
export LLAMAHERD_SERVER_BIN=/opt/llama.cpp/build/bin/llama-server
export LLAMAHERD_MODELS_DIR=~/llm/models
```

An explicit `./configs` or `./library` next to the working directory (a
checkout) wins over the per-user data dir, so development and `go test` keep
using the repo's files.

### Hugging Face token

For private or gated repos, LlamaHerd reads a token from, in order:
`HF_TOKEN` / `HUGGING_FACE_HUB_TOKEN`, a `0600` file `hf_token` next to
`config.json` (written by `llamaherd download token <token>` or the GUI panel),
then the standard Hugging Face cache (`~/.cache/huggingface/token`). The token is
never written into configs or presets. Set `HF_ENDPOINT` to use a mirror.

### VRAM pre-flight guard

Before every start (GUI, CLI `open`, API `/open`) LlamaHerd predicts the GPU
footprint and **refuses to launch above 90% of total VRAM**, because a runaway
allocation can take down the whole WSL distro or desktop. The estimate
(`control/vram.go`) is the sum of:

- model weights × the share placed on the GPU (`-ngl`; `--override-tensor` CPU
  offloads subtracted),
- KV cache = context × full-attention layers × KV heads × head dim × bytes per
  element of `kCache`/`vCache` (`f16`, `q8_0`, `q4_0`, …). Hybrid models
  (`full_attention_interval`) only keep a KV cache on a fraction of layers,
- the recurrent (SSM) state of hybrid layers,
- a compute-buffer allowance (grows with `ubatch` and context),
- the `mmproj` vision sidecar (unless `--no-mmproj-offload`) and a separate
  draft model, when set.

Layer/head counts come from the GGUF header; if it cannot be read the result is
a rough, conservative file-size estimate. With no NVIDIA GPU visible
(`nvidia-smi` missing) nothing is blocked. The estimate is approximate — treat
it as a safety net, not a guarantee. A blocked start logs the full breakdown and
sets the UI warning; `SafeGpuLayers` picks the largest `-ngl` that fits when
synthesizing default presets.

**Testing without a GPU:** `internal/ggufgen` writes tiny synthetic GGUF
headers and `control.SetTotalVramForTesting` fakes the card size, so
`go test ./control` and the end-to-end API test exercise "fits", "blocked",
"no GPU" and `SafeGpuLayers` on any machine.

### Updating llama.cpp from the GUI

The **llama.cpp** tab shows the installed build/commit and can update and
rebuild your checkout in one click (Linux/macOS/WSL only). It runs
`scripts/update-llamacpp.sh`:

1. `git pull --ff-only` in the llama.cpp checkout (never creates a merge commit),
2. `cmake -S . -B <build> -DGGML_CUDA=ON -DCMAKE_BUILD_TYPE=Release -DCMAKE_CUDA_ARCHITECTURES=<arch>`,
3. `cmake --build <build> --config Release -j<cores>`.

The log streams into the tab and a running `llama-server` is warned about (the
new binary is used after the next restart). Paths come from `llamaDir` /
`LLAMAHERD_LLAMA_DIR` and `llamaBuildDir` / `LLAMAHERD_LLAMA_BUILD_DIR`. When running
the script yourself, override with `LLAMA_DIR`, `BUILD_DIR`, `CUDA_ARCH`
(auto-detected from `nvidia-smi`'s compute capability, e.g. `89`; `native` if no GPU is detected),
`JOBS` and `GIT_PULL_ARGS`. `llamaherd doctor` warns when the build is older
than the minimum the presets need (override with `LLAMAHERD_MIN_LLAMA_BUILD`).

### Paths

- The models directory defaults to `~/LlamaHerd/models` (`%USERPROFILE%\LlamaHerd\models`
  on Windows) and is created on first run; override it with `LLAMAHERD_MODELS_DIR`
  or `modelsDir` in the config file.
- Per-alias presets live in `configs/<alias>.json`. Model / mmproj paths there
  are stored **relative to the models directory** so shipped configs stay
  portable (a saved absolute path is also accepted).

## Project layout

| Path | Purpose |
| --- | --- |
| `main.go` | Wails window entry point |
| `app.go`, `app_api.go`, `app_llama.go` | Wails-bound application state machine and actions |
| `cli.go` | Same-binary command-line interface |
| `schema/` | Shared data types (config, preset, model, server state) |
| `control/` | Process control, metrics, VRAM estimate |
| `store/` | GGUF header parsing, model scan, config persistence |
| `hf/` | Hugging Face catalog, API client, resumable verified downloads |
| `api/` | Loopback control API |
| `frontend/dist/` | Vanilla HTML/JS/CSS frontend (no framework) |
| `configs/` | Per-alias preset files |
| `~/LlamaHerd/models` | Default models directory (scanned recursively, created on first run); configurable |
| `scripts/` | Installer (`install.sh`, `install.ps1`, `uninstall.sh`), launcher, shortcuts, helpers |
| `assets/` | Icons (generated by `scripts/make-icon.py`) |
| `.github/workflows/` | CI (`ci.yml`) and release builds (`release.yml`) |
| `docs/` | Deeper docs (see [`docs/architecture.md`](docs/architecture.md)) |
| `BACKLOG.md` | Plan for making the project shareable |
| `CHANGELOG.md` | Release notes ([Keep a Changelog](https://keepachangelog.com/) + [SemVer](https://semver.org/)) |

## Troubleshooting

Run `llamaherd doctor` first — it checks the binary, GPU, GUI libraries,
display, models dir, disk space, and the control-API port, and prints a
copy-pasteable fix for anything that fails.

**"llama-server binary" fails in doctor / Start does nothing.** LlamaHerd could
not find `llama-server`. Point at your build explicitly:

```bash
LLAMAHERD_SERVER_BIN=/opt/llama.cpp/build/bin/llama-server llamaherd doctor
```

…or set `serverBin` in the config file (see [Configuration](#configuration)).
A build under the usual `~/llama.cpp/build*/bin` is discovered automatically.

**Blank window, or a WebKit/GTK error on Linux/WSL.** The GUI needs GTK3 +
WebKit2GTK:

```bash
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev
```

On WSLg, force the display and renderer:
`DISPLAY=:0 GSK_RENDERER=cairo ./build/bin/llamaherd`.

**"something is already listening on 127.0.0.1:8090 — Stop it first".** Another
`llama-server` owns the port. Press **Force stop** in the llama.cpp tab
(`ForceStopLlama`) to kill every `llama-server`, or change `basic.port`. The
control-API port (`8517`) is checked separately by `doctor`.

**Start is refused by the VRAM guard.** The pre-flight estimates the GPU
footprint and refuses to launch above ~90% of total VRAM. Lower
`basic.context`, quantize the KV cache (`kCache`/`vCache` → `q8_0`/`q4_0`), or
reduce `basic.gpuLayers`. The log shows the full estimate breakdown.

**Downloads fail or verify-check fails.** A cancelled/failed download keeps its
`.part` file and resumes on the next attempt; a sha256 mismatch removes the
corrupt `.part`. For private/gated repos set a token with
`llamaherd download token <token>` (see [Hugging Face token](#hugging-face-token)).
Check `llamaherd download status` for the live error.

**The WSL IP row is blank / I can't reach the server from Windows.** The row is
only filled inside WSL. `llama-server` binds `127.0.0.1` by default, so Windows
cannot reach it unless you enable mirrored networking or set a non-loopback
`--host` (Basic tab → Host).

## Roadmap

See [`BACKLOG.md`](BACKLOG.md) for the full list of what is needed to make this
project easy to install and share.

## Contributing

Contributions are welcome. Build and run instructions are above; a fuller
`CONTRIBUTING.md` is on the backlog. Until then, open an issue or a pull
request.

## License

[The Unlicense](LICENSE) — this is free and unencumbered software released
into the public domain. You may copy, modify, publish, use, compile, sell, or
distribute it, for any purpose, commercial or non-commercial, with no
conditions.

## Acknowledgements

Built on top of [llama.cpp](https://github.com/ggml-org/llama.cpp) and
[Wails](https://wails.io/). Not affiliated with the llama.cpp project.
