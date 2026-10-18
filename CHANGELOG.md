# Changelog

All notable changes to LlamaHerd are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned

- Screenshot / screencast in the README.
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
