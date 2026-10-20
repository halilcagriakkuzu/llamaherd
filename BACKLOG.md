# Backlog

Open work only — finished items are removed (see `CHANGELOG.md` for what shipped).

The medium-sized, ordered implementation plan for the next batch lives in
[`TASKS.md`](TASKS.md).

- [ ] **macOS runtime verification:** the binaries build and cross-compile, but nothing has
  run on a Mac yet — Start/Stop, `ForceStopLlama`, RSS/GPU rows, `doctor`, the GUI,
  `scripts/install.sh` and the `.app` bundle (Gatekeeper / notarization behaviour).
- [ ] **Native Windows llama.cpp:** the Windows app is verified with a WSL-hosted server
  (`wsl:` bridge) and `install.ps1 -Mode native`; still untested with a native
  `llama-server.exe` (Start/Stop via `TerminateProcess`, `ForceStopLlama` via `tasklist`).
- [ ] **`install.ps1` default (WSL) mode** end-to-end on a clean machine (it runs
  `install.sh` inside the distro and creates shortcuts).
- [ ] **Case-insensitive filesystems:** confirm GGUF scanning and models-dir-relative paths
  behave on Windows/macOS (path-case differences, aliases derived from file names).
- [ ] **Non-NVIDIA GPUs:** check `doctor`, the VRAM estimate and the metrics rows on AMD /
  Intel / Apple GPUs (only `nvidia-smi` is probed today; they should degrade cleanly).
- [ ] **Code signing:** sign Windows binaries and notarize macOS builds to avoid
  SmartScreen / Gatekeeper warnings.
- [ ] **Native packages:** `.deb`/`.rpm`, winget/`.msi`, `.dmg`/Homebrew cask (the script
  installer is the supported path for now).
- [ ] **App update check:** notify (or optionally self-update) when a newer GitHub release
  exists.
- [ ] **Library curator:** import existing model files, detect duplicates (rename,
  categories and moves already exist).
- [ ] **Multi-instance:** more than one managed `llama-server` / control-API instance
  (today the control API is a singleton).
- [ ] **i18n:** translation framework, English stays the default.
- [ ] **Privacy statement:** document in the README that no telemetry is collected (or
  design an explicit opt-in).
