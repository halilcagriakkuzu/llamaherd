# Changelog

All notable changes to LlamaHerd are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **New UI.** Card-based layout with an app bar (logo, live status pill, Start/Stop),
  light/dark theme (follows the system, header toggle), switch controls, two-column
  responsive settings grid with `--flag` hints, VRAM usage bar, terminal-style logs
  (collapsed by default; opens on start/crash), refined modals and dialogs. No
  features were removed. Default window is now 1240x820 (min 960x640).
- New orange llama icon.

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

- Windows and macOS runtime testing of process control (see `BACKLOG.md` §5).

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
