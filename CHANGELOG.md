# Changelog

All notable changes to LlamaHerd are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.1] - 2026-10-21

### Added

- **llama.cpp bootstrap in the installer**: when `install.sh` finds no `llama-server`, it now
  asks (via `/dev/tty`, so the `curl … | bash` one-liner works too) whether to use an existing
  build at a path you type, clone `ggml-org/llama.cpp` and build it from source, or skip.
  The source path installs the missing tools (git, cmake, gcc/g++/make via apt/dnf/pacman/
  zypper; Xcode CLT + Homebrew cmake on macOS), reuses an existing checkout, and builds with
  the same cmake flags as `scripts/update-llamacpp.sh` (CUDA auto-detected from `nvidia-smi`).
  New flags `--llama-server PATH`, `--install-llamacpp`, `--llama-dir`,
  `--llama-build-dir`, `--llama-backend auto|cuda|vulkan|metal|cpu` and `--no-llamacpp` make
  it non-interactive; `-y`/`--dry-run`/no terminal never prompt. The build is recorded as
  `serverBin` and its checkout as `llamaDir`.

## [0.4.0] - 2026-10-21

### Added

- **App-drawn title bar on WSL**: the main window is frameless by default inside WSL, where
  the native frame is WSLg's Windows-side chrome around a Linux window. The app header is
  now the title bar (drag to move, double-click to maximise) with minimise / maximise /
  close buttons; Wails' edge detection keeps all eight resize zones working. The window's
  size and position are remembered in `<data dir>/window.json` and clamped to the available
  screens on restore (size only under WSLg, which owns window placement).
  `LLAMAHERD_FRAMELESS=0` (or `"frameless": false` in the config file) restores the native
  frame; `=1` forces the custom one on other platforms.
- **`GET /ping` identity beacon**: the control API answers a tiny unauthenticated
  `{"app":"llamaherd"}` on loopback (the `Host` guard still applies), so local tools can
  tell LlamaHerd from an unrelated listener.
- **Remembered selection**: the model/preset you last selected or started is stored in
  `<data dir>/last-run.json` and restored (selected, with its config applied) on the next
  launch. Deleting/renaming the model falls back to the normal first-model selection; the
  server is never auto-started.
- **Top-level views**: Config, Performance & benchmarks and llama.cpp are now separate
  screens behind a top tab bar, so the performance charts and benchmark table no longer
  squeeze the config form (and llama.cpp gets the full width). Logs stay collapsed by
  default and only open when the user expands them.
- **Benchmark export**: `GET /benchmarks/export?format=csv|json`, a GUI **Export** button
  next to Clear (CSV Blob download) and `llamaherd bench export [--format csv|json]
  [--out file]`. CSV is RFC 4180 with a stable header, human-readable `when` next to raw
  `at`; JSON round-trips through the benchmark store.
- **Benchmark `/metrics` counters**: new Advanced → Runtime **Metrics** toggle adds
  `--metrics`. Benchmark runs scrape `/metrics` before/after and store prompt-eval time,
  busy/active slots, queue depth and context/KV usage in `schema.Benchmark`; the table shows
  them, and older builds without the endpoint still benchmark with empty counters.
- **Non-NVIDIA GPU support**: a small GPU provider abstraction now probes NVIDIA
  (`nvidia-smi`, unchanged), AMD (`rocm-smi`), Intel (`xpu-smi`) and Apple Silicon
  (unified memory via `sysctl`). `doctor` names the active vendor; unknown values stay `—`
  and the VRAM guard keeps its conservative "no estimate" behaviour. The AMD/Intel parsers
  are unit-tested with injected output and documented as untested on real hardware.
- **Download queue**: starting a second download queues it; the queue runs one job at a
  time, supports pause/resume (continuing from the `.part` file), one retry with backoff,
  and removing a waiting job without touching the active transfer. API:
  `POST /download/pause`, `/download/resume`, `/download/retry`, `/download/queue`
  (`{id}` removes a waiting job); CLI: `download queue [rm <id>]`, `download pause|resume|retry`.
  `GET /download` and the snapshot now include `queue`, `paused` and `currentId`.
- **Refreshed top bar and notification center**: server actions stay in the header, while
  theme, diagnostics, llama.cpp update and paths moved into one **⋯** menu. Toasts (downloads,
  crashes, benchmarks, imports, errors) also land in a bell **notification center** with
  timestamps, persisted for the browser session; clearing the center does not remove the
  toast and vice versa. The model actions now read Rescan / Folder / Import / Download.
- **Host OS notifications**: server ready, server crash / not responding, download
  finished/failed and benchmark finished produce one native notification per event (Linux
  `notify-send`, macOS `osascript`, Windows toast). A per-class toggle in the llama.cpp tab
  (default on) and `GET/POST /notifications` control them; `llamaherd notifications
  [status|on|off|ready|crash|download|benchmark on|off]` does the same from the CLI. With no
  native backend the event stays in the in-app center. WSL-hosted servers notify on the
  Windows host because the GUI process owns the notifier.
- **Multi-part GGUF downloads**: selecting a `-00001-of-000NN.gguf` shard set downloads
  every part into the same folder (per-shard sha256 verification and resume), skips shards
  that already completed, records one library entry/preset whose `--model` is the first
  shard, and treats the whole set as one queue job. `GET /download/files` now includes
  `parts` and the ordered `shards` list; `GET /download` reports `shards`/`shardIndex`.
- **Model management in the GUI**: search/filter (favorites, recently used, vision, MTP),
  pin models, rescan the models directory, reveal a model in the file manager, copy its path,
  and delete the `.gguf` (the preset file is kept).
- **Open chat**: a header button opens llama-server's built-in web UI.
- **Restart** with the current form settings; restart is also available as `llamaherd restart`.
- **Native file/folder pickers** for the first-run paths, the llama.cpp tab paths, `--mmproj`
  and draft models.
- **Logs toolbar**: filter, errors-only, pause/resume auto-scroll, copy, save and clear; the
  view is capped at 5 000 lines.
- **Error toasts** for failed actions, and preset duplicate/reset/export actions.
- **Download catalog**: installed badge, "Open" for installed files and free-disk readout.
- **Model details**: a Details… action shows architecture, layers, trained context, quant,
  file size, source, sidecars and last use (plus copy path).
- **Live validation and Auto-fit**: the form reports context/trained-context, `-ngl`, port,
  sidecar and VRAM-budget problems inline; Auto-fit chooses a context and `-ngl` that pass
  the guard. API: `POST /config/autofit`; CLI: `config autofit`.
- **Performance charts**: 10-minute VRAM / CPU RAM / tokens-per-second sparklines with GPU
  utilisation and temperature when available.
- **Security**: the loopback API now validates `Host`/`Origin`, requires JSON content type for
  mutating requests, and generates a bearer token on first setup (`"apiToken": "off"` opts out).
- **Import local models**: an Import dialog copies (or moves) a `.gguf` into the models
  directory, asks whether the model has a vision projector (with a file picker), writes its
  library metadata and creates a fitted preset. API: `POST /model/import`; CLI:
  `model import <file> [--mmproj <file>] [--move]`.
- **Benchmarks**: measure the loaded model + config with a fixed prompt (`/completion`
  timings), store the result per model/preset/config fingerprint, and show a fastest-first
  table with prompt/gen tok/s and peak VRAM. Editing a config marks its old result as
  “config edited”; results survive restarts. API: `POST /benchmark`, `GET /benchmarks`,
  `POST /benchmarks/clear`; CLI: `bench [list|clear]`.
- **Reasoning budget** is now wired to `--reasoning-budget` (0 = unset/server default).
- **Persistent logs**: app/server output is mirrored into a rotating file
  (`<data>/logs/llamaherd.log`, 5 MiB × 3, `logFile: "off"` to disable), exposed as
  `GET /logs?tail=n` / `?after=seq`, `llamaherd logs [-f]`, and a **Copy diagnostics**
  action (version, paths, doctor report, last 200 lines).
- **CLI polish**: `list --json`, `status --summary` and shell completions
  (`llamaherd completion bash|zsh|fish`).
- **Onboarding detection**: a **Detect** button in the first-run dialog and the llama.cpp
  tab searches PATH, common llama.cpp build trees and (on Windows) WSL distros, then
  validates the binary with `--version`. `SetPaths` refuses a binary that does not run.
- **Import v2**: import several `.gguf` files or a whole folder at once (one shared vision
  projector), drag-and-drop files onto the Models panel, and duplicate detection (size first,
  then sha256) that reports “already imported as …” instead of copying again; an explicit
  “import duplicates anyway” option bypasses it. Results are reported per file. API:
  `POST /model/import` takes `paths` + `allowDuplicate` (the single `src` form still works)
  and `POST /model/import/check` previews; CLI: `model import <file|dir>... [--check]
  [--allow-duplicate]`.
- **Gated repositories and licences at download time**: finding a repo's files now shows its
  licence and, for gated repositories, a notice with an “Open licence page” button. A gated
  download without a Hugging Face token stops before transferring anything, and a 401/403
  on a gated repo explains that the licence must be accepted (or that manual approval is
  pending) instead of a bare “access denied”. `GET /download/files` returns `gated` and
  `license`. Downloaded models record their repo and licence in `library/<id>.json`
  (`provenance`), shown in the details drawer with an “Open source page” link. A model that
  is already on disk is never checked or warned about again.
- API: `POST /restart`, `/rescan`, `/model/delete`, `/model/favorite`, `/config/reset` and
  `GET /config/export`; CLI: `restart`, `rescan`, `model delete|favorite`, `config reset|export`.

### Security

- Hugging Face references are validated before any request: `../` traversal, encoded
  separators, backslashes, control characters and invalid `owner/name` segments are
  rejected, and the download destination must stay inside the models directory.
  `ParseRef` and the GGUF parser have fuzz targets; separately supplied file refs are validated too.
- Added [`docs/security.md`](docs/security.md) (API threat model, outbound hosts, offline use)
  and a "no telemetry" privacy section to the README.

### Fixed

- **Frameless window resize was hard to grab**: the Wails hit zone was 6 px and its
  right/bottom detection compares `window.outerWidth` with CSS-pixel mouse coordinates —
  WSLg reports the outer size in physical pixels, so those edges almost never triggered.
  The app now widens the zone to 10 px and reports the viewport size as the outer size in a
  frameless window; resize zones are also disabled while maximised (where they only showed
  a cursor and did nothing).
- **Doctor false positive on the control-API port**: the port check inspected only the
  first 4 KiB of `/status` and looked for `presetNames`, which follows the model list in
  the snapshot — on any real install it sat beyond the read window, so LlamaHerd reported
  its own API as "in use by another process". It now probes `GET /ping` first, falls back
  to a full `/status` read for older builds, and bypasses configured proxies.
- **llama.cpp view layout**: the card is now the single scroll container (no more rows cut
  off at the card edge or nested scrollbars) and uses two columns on wide windows, so the
  whole tab fits a normal window height and only scrolls inside the card when it doesn't.
- **Logs only on Config**: the log panel is hidden on the Performance and llama.cpp views.
- **"Open chat" under WSL**: `runtime.BrowserOpenURL` ends in `xdg-open`, which usually has
  no browser registered inside WSL and fails silently. Links now go through
  `settings.OpenExternalURL`, which prefers the Windows host (`wslview`, PowerShell,
  `cmd.exe`) before `xdg-open`, and reports a real error to the UI toast.
- `POST /config/update` now really merges a partial config (as documented) instead of
  replacing the preset and dropping omitted fields.
- Case-only path differences (Windows/macOS) no longer break model lookups, config
  relativisation or import containment checks.
- **Data races in `App` state**: config/model/preset reads and writes are now serialized with
  a state lock; a `-race` regression test hammers config writes and snapshots concurrently.
- **Duplicate base names** (e.g. two `model.gguf` files in different folders) no longer share
  one `configs/<alias>.json` / `library/<id>.json`; colliding ids get a stable path hash.
- **Changing paths while a server is running** is refused instead of silently orphaning the
  process.
- A start that never answers `/health` no longer hangs in `starting` forever: it becomes
  “not responding” with Retry/Stop actions and the last 20 server log lines.
- Downloaded/default presets now auto-fit the context to the card and trained window instead
  of hardcoding 51 200 tokens.
- The API rejects cross-site `text/plain` POSTs, `Host` headers that are not loopback, and
  non-loopback browser origins.

### Tooling

- Headless UI smoke test (`scripts/uishot/smoke.sh`) asserts the running model list,
  benchmark table, import/mmproj dialog and the not-responding banner; it runs in CI.
- Added a snapshot JSON contract test so frontend keys cannot be renamed silently.
- `scripts/launch-gui.sh` (what the Windows/desktop shortcuts call) now rebuilds
  `build/bin/llamaherd` when a Go source, `go.mod`/`go.sum` or an embedded frontend file
  is newer, and takes `--rebuild` to force it. Previously it only built a missing binary,
  so a shortcut kept launching a stale window after UI or startup changes.

### Changed

- New presets use a fitted context and the largest `-ngl` that fits; the VRAM guard is
  unchanged.

## [0.3.1] - 2026-10-19

### Added

- **Preset management from the Config dropdown**: `<new preset…>` saves the current form under
  a name you choose, and a `⋯` menu renames or deletes the selected preset (or discards unsaved
  edits). API: `POST /config/rename`; CLI: `config rename <alias> <old> <new>`.

### Fixed

- Deleting a preset no longer risks an out-of-range selection, and removing an earlier preset
  keeps the applied preset selected.

## [0.3.0] - 2026-10-18

### Added

- **Model catalog from the repository**: `hf/catalog.json` (embedded, refreshed from GitHub `main`,
  cached; `LLAMAHERD_CATALOG_URL` overrides) with 33 real, header-measured entries (Ornith 1.5 9B,
  Qwen3.5 9B, Qwen3.8 27B, gpt-oss 20B, Qwen3-Coder 30B-A3B and Next). Models that do not suit the
  hardware are hidden; GPU, GPU+RAM (hybrid) and CPU fits are told apart.
- **New download dialog**: a collapsible list of up to five picks (best quality, vision, longest
  context, MTP, lightweight), **Browse all models…** for the full catalog, and a Hugging Face link
  field that lists every `.gguf` of a repo with sizes (choose the file, optionally include the
  vision projector / MTP draft). Downloads continue when the dialog is closed; progress shows in
  the status bar (click to reopen). CLI: `download list [--all]`, `download files <repo>`,
  `--no-vision` / `--no-mtp`; API: `GET /download/files`, `GET /catalog?all=1`.
- **Library editing**: rename models (display name + alias; `configs/<alias>.json` moves
  with it), nested categories with create/rename/delete (empty folders persist; deleting
  keeps the models), move / remove from category. GUI (⋯ menu, right-click), CLI
  (`model rename|move`, `category create|rename|delete`) and API. Data lives in
  `library/<id>.json` (`name`, `alias`) and `library/_categories.json`.
- **Generation speed** in the Server box: average tokens/s of the managed server's finished
  requests (token-weighted, parsed from llama-server's per-request timing line);
  hover for min / max / last and the request count. Resets on every start; not
  measured for servers LlamaHerd did not start.

### Changed

- **New UI.** Card-based layout with an app bar (logo, live status pill, Start/Stop),
  light/dark theme (follows the system, header toggle), switch controls, two-column
  responsive settings grid with `--flag` hints, VRAM usage bar, terminal-style logs
  (collapsed by default; opens on start/crash), refined modals and dialogs. No
  features were removed. Default window is now 1240x820 (min 960x640).
- New orange llama icon, also embedded in `llamaherd.exe` (`rsrc_windows_amd64.syso`,
  regenerate with `scripts/make-winres.sh`) so Windows shortcuts, the taskbar and
  Explorer show it; Windows file properties now carry product name/description.
- Windows shortcut icons live in `%USERPROFILE%\LlamaHerd\` (hash-named, BMP-encoded
  ICO) — a blank icon was observed from the AppData location.

### Fixed

- A `wsl:` llama-server binary is now checked inside WSL (the GUI wrongly reported
  it missing and showed the first-run dialog).

## [0.2.0] - 2026-10-18

### Added

- Windows app can drive a `llama-server` inside WSL: `serverBin` =
  `wsl:<distro>:/path` launches via `wsl.exe`, translates Windows/UNC paths,
  stops the Linux process, and `doctor` checks the binary + warns about slow
  `/mnt/<drive>` model access.

### Changed (llama.cpp updater)

- `update-llamacpp.sh` auto-detects the CUDA architecture from `nvidia-smi`
  (falls back to `native`) instead of hard-coding `89`; `CUDA_ARCH` still overrides.

### Quality

- GPU-free unit tests for GGUF parsing, model scan, config/library load/save,
  VRAM estimate (synthetic GGUFs, faked card size) and the control API (fake
  controller + end-to-end against the real App with a fake server binary).
- CI: gofmt check, golangci-lint (errcheck/govet/staticcheck/unused/misspell),
  `go test -race`.
- Security hygiene: user config file is now written `0600`; personal paths
  removed from tracked docs/scripts; VRAM guard and llama.cpp update documented.

### Changed

- Default models directory is now `~/LlamaHerd/models` (created on first run).
- Removed the dummy/sample-data phase: no built-in fake models or presets, no
  fake Browse button; `dummy/` became `schema/`.
- Windows GUI detaches its console window on launch.
- `install.ps1` defaults to the real repo instead of an `OWNER` placeholder.

## [0.1.1] - 2026-10-18

### Fixed

- Windows release build no longer needs MSYS2 (cgo-free, fixes `go: command not found`).
- Release workflow: link `UniformTypeIdentifiers` on macOS (darwin/arm64 build
  failed), use the `macos-15-intel` runner (`macos-13` was retired), mark macOS
  and Windows as best-effort, and publish the release even if those fail.

### Planned

- Windows and macOS runtime testing of process control (see `BACKLOG.md`).

## [0.1.0] - 2026-10-18

First public release. One Go binary that is both the Wails GUI and a CLI/daemon
sharing a single loopback control API.

### Added

- **Model browser** — recursive `*.gguf` scan, grouped by category, with vision
  (mmproj), embedded-MTP, and MTP draft-sidecar badges.
- **Model download** — hardware-scored Hugging Face catalog plus any pasted
  repo/URL; resumable, size + sha256-verified downloads; auto-installs a
  ~50k-context preset and detects vision + MTP sidecars.
- **Config form / presets** — Basic / Advanced / llama.cpp tabs, built-in
  presets plus per-alias `configs/<alias>.json`, portable (models-dir-relative)
  paths, save/load.
- **Process control** — start/stop/detect-crash of `llama-server`, force-stop of
  any stray `llama-server`, live `/metrics` and `/props`, VRAM/RAM/uptime.
- **VRAM pre-flight guard** — GGUF-header-based footprint estimate; refuses a
  start above ~90% of total VRAM.
- **CLI + control API** — `list`, `status`, `open`, `stop`, `save`, `load`,
  `config`, `daemon`, `doctor`, `download`, `version`, per-subcommand `--help`;
  loopback HTTP API with an optional bearer token (`LLAMAHERD_API_TOKEN`).
- **`doctor`** — environment checks (binary, GPU, GUI libs, display, models dir,
  disk, API port) with copy-pasteable fixes, in the CLI, API, and GUI.
- **Onboarding** — portable per-OS config (`settings`), first-run setup,
  terminal installer, and a Hugging Face token stored safely (`0600`).
- **Cross-platform** — Linux (incl. WSLg), Windows 11, and macOS; platform-split
  code for process discovery, RAM, disk, signals, and detach.
- **Docs & infra** — README, `QUICKSTART.md`, `docs/architecture.md`,
  `BACKLOG.md`, CI matrix, and a tagged release workflow.
