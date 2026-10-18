# Backlog — turning LlamaHerd into a shareable, installable open-source project

This is a planning document only. Nothing in the codebase is changed by these
items; each line is a task to be done, reviewed, and checked off.

---

## 0. Repository hygiene / open-source readiness

- [x] Choose and add an open-source `LICENSE` — **The Unlicense** (public domain, no conditions) added as `LICENSE`.
- [x] Initialize git if not already done (`git init`, initial commit) — done; remote `git@github.com:halilcagriakkuzu/llamaherd.git`, branch `main`.
- [x] Add a `.gitignore` covering `build/`, `build/bin/`, `qtbox`, `configs/*.local.json`, logs, editor/OS junk (`.DS_Store`, `Thumbs.db`) — added (build output, models, `configs/`, `library/`, `*.gguf`, `hf_token`, OS/editor junk, `.claude/settings.local.json`).
- [x] Decide what must NOT be committed: absolute model paths, personal `configs/*.json`, `build/`, `launch.log` — `configs/`, `library/`, `models/`, `build/` are ignored; shipped presets come from `dummy/data.go`.
- [ ] Add a `CONTRIBUTING.md` (how to build, run, submit PRs, code style).
- [ ] Add a `CODE_OF_CONDUCT.md`.
- [ ] Add issue/PR templates (`.github/ISSUE_TEMPLATE/`, `PULL_REQUEST_TEMPLATE.md`).
- [x] Enable GitHub Actions CI (build + `go test ./...` on Ubuntu) — see section 8.
- [ ] Add a project description + topics on the GitHub repo page.
- [ ] Add screenshots/GIF of the GUI to the README and `docs/`.
- [x] Pick a project name without trademark/name conflicts — chose **"LlamaHerd"** (no existing product/trademark found; "LlamaHerd" is a generic phrase, and "LlamaDeck"/"Llamanager" were rejected as taken).
- [x] Rename the codebase to "LlamaHerd" everywhere: window title, binary/output name (`build/bin/llamaherd`), Go module name (`llmman` → `llamaherd`), import paths, scripts (`launch-gui.sh`, shortcuts), log/UI message text, docs (`AGENTS.md`, `tasks/`, `README.md`), and the repo directory itself (`LLM_Man_GUI` → `LlamaHerd`).

## 1. Configuration abstraction (remove hard-coded machine paths)

- [x] Move `llama-server` binary path out of hard-coded values into a config/env var (`LLAMAHERD_SERVER_BIN`) with auto-discovery fallback — done in the new `settings` package (`ServerBin()`: env → PATH lookup → common `~/llama.cpp/build*/bin` locations → informative default). The llama.cpp checkout dir and build subdir became configurable too (`LLAMAHERD_LLAMA_DIR`, `LLAMAHERD_LLAMA_BUILD_DIR`), derived from the binary path when possible.
- [x] Move the models directory out of hard-coded values into a config/env var (`LLAMAHERD_MODELS_DIR`), with a sane default of `<project-root>/models` — done via `settings.ModelsDir()` (and `settings.ProjectRoot()`, which walks up to the `go.mod` dir). `store.ModelsRoot` removed; `app.go` and `store` tests use the resolved dir.
- [x] Move the loopback control API port (`127.0.0.1:8517`) into config/env (`LLAMAHERD_API_PORT`) — `settings.APIPort()`/`settings.APIAddr()`; `api.Addr()` is now a function; `cli.go` uses it. Default stays 8517.
- [x] Add a user config file in a standard per-OS location: `~/.config/llamaherd/config.json` (Linux/XDG), `%APPDATA%\llamaherd\config.json` (Windows), `~/Library/Application Support/llamaherd/config.json` (macOS). Implemented as `settings.ConfigFile()`/`Config()`/`Save()`/`Reload()`; JSON fields `serverBin`, `llamaDir`, `llamaBuildDir`, `modelsDir`, `apiPort`. Precedence: env var > config file > auto-discovery > default (env and file work together). `LLAMAHERD_CONFIG` overrides the path.
- [x] Ensure shipped `configs/*.json` contain no machine-specific absolute paths — rewritten to paths relative to the models directory. `store.LoadAlias` resolves them via `settings.ResolveModelPath`; `store.SaveAlias` stores them relative via `settings.RelativizeModelPath` (so user saves stay portable). Built-in `dummy` presets/model list made relative and resolved at runtime too.
- [x] Add a "first run" default that prompts the user to set the binary + models paths — `App.SetPaths` writes the config file and applies it live; `Snapshot.FirstRun` drives a modal in the GUI; `computeFirstRun` triggers when no config file exists and the binary or models dir is missing.

## 2. Installer / setup flow

- [x] Design the install experience: build-from-source vs. prebuilt binary vs. script installer — chose a **terminal script installer** that supports **both** building from a local checkout and downloading a prebuilt release asset, with `env > config file > discovery > default` settings already in place.
- [x] Write an `install.sh` (Linux/macOS/WSL) that installs system deps and builds the app — `scripts/install.sh` (also `--from-release`, `--prefix`/`--bin-dir`/`--data-dir`, `--with-deps`/`--no-deps`, `--dry-run`, `-y`). Auto-detects apt/dnf/pacman/zypper, picks the `webkit2gtk-4.1`/`4.0` build tag, installs a `.desktop` entry + GUI launcher, seeds presets, and writes the user config. `scripts/uninstall.sh` wraps `--uninstall`.
- [x] Write an `install.ps1` / `.bat` (Windows 11) equivalent — `scripts/install.ps1`. Default `-Mode wsl` runs `install.sh` inside WSL and creates Start Menu + Desktop shortcuts (ported from `make-windows-shortcut.ps1`); `-Mode native` downloads/builds a native `llamaherd.exe` and edits the user PATH (experimental). `-Uninstall`/`-Purge` supported.
- [x] Provide prebuilt release binaries via GitHub Releases (Linux x86_64/arm64, Windows, macOS) — `.github/workflows/release.yml` builds stable-named assets (`llamaherd_<os>_<arch>.tar.gz` / `.zip`, including the icon + `configs/`) on tag push. arm64-Linux and native-Windows jobs are `continue-on-error` (best-effort).
- [ ] Decide on packaging: `.deb`/`.rpm` (Linux), `.msi`/winget (Windows), `.dmg`/Homebrew cask (macOS) — *script installer chosen for now; native packages deferred.*
- [x] Add a desktop entry / Start Menu shortcut / `.app` bundle created by the installer — Linux `.desktop` + icon + `llamaherd-gui` launcher; macOS `.app` bundle in `~/Applications`; Windows shortcuts via `install.ps1`.
- [x] Handle upgrades/uninstall for each platform — re-running the installer upgrades in place (existing configs and config file are preserved); `--uninstall` uses an install manifest, `--purge` also removes data + config. (Note: also made `configs/` + `library/` resolve to a per-user data dir, `settings.DataDir()`/`ConfigsDir()`/`LibraryDir()`, so an installed binary no longer writes into the current directory.)


## 3. Dependency checks (pre-flight / doctor)

Implemented in the `doctor` package (`doctor.Run()` → `Report`), surfaced by
`llamaherd doctor [--json]`, the control API (`GET /doctor`), and a GUI panel +
first-run banner (`Snapshot.Doctor`, refresh via `App.RunDoctor`). Exits 1 when
any check fails; every non-OK check carries a copy-pasteable `Fix`.

- [x] Add a `llamaherd doctor` command (or a first-run check) that reports the environment — `llamaherd doctor [--json]`, plus `GET /doctor` and the GUI panel.
- [x] Check that `llama-server` exists and is executable; if missing, print where it was expected — reports the resolved path (or "not found at <path>") and a fix.
- [x] Detect and report the `llama-server` version/commit and warn on too-old versions — parses `--version`; warns below `doctor.MinLlamaBuild` (override `LLAMAHERD_MIN_LLAMA_BUILD`).
- [x] Check GPU/CUDA availability (`nvidia-smi`), report total VRAM, warn if none found.
- [x] Check required system libraries per OS — Linux: GTK3 + WebKit2GTK via pkg-config/ldconfig (build+runtime); Windows: WebView2 registry; macOS: `xcode-select -p`. Also a display/WSLg check.
- [x] Check the models directory exists and is readable — reports counts (models + mmproj).
- [x] Check free disk space before recommending a model download — `Statfs` on the models volume; warns below 10 GiB.
- [x] Check the control-API port is free (singleton check) with a clear message — distinguishes free / held by LlamaHerd / held by another process.
- [x] Show actionable, copy-pasteable fix instructions for every failed check.
- [x] Add the same checks as a GUI panel/banner on first run — the llama.cpp tab panel + a banner when any check warns/fails. (Windows/macOS branches are implemented but unverified here.)

## 4. Model download / onboarding (Hugging Face)

Implemented in the `hf` package (direct Hub API + `resolve/main` URLs; no Python
or `huggingface-cli` needed), surfaced by `llamaherd download …`, the control
API (`GET /catalog`, `GET|POST /download`, `POST /download/cancel`,
`POST /token`), and a GUI download panel (`Snapshot.Download`, live).

- [x] Add a first-run flow that offers to download one small default model — the first-run prompt has a "Download a starter model…" button; `hf.Starter()` (Qwen2.5 0.5B, ~0.49 GB) is marked with ★.
- [x] Implement a Hugging Face downloader (e.g. via `huggingface-cli`, `hf`, or direct `resolve/main` URLs) with resume support — native Go downloader over `…/resolve/main/…`; resumes from `.part` via HTTP Range.
- [x] Let the user choose a model from a curated list, or paste a HF repo/file path — `hf.Catalog` scored against VRAM/RAM; any `owner/name`, `owner/name/file.gguf` or full hf.co URL via `hf.ParseRef`; presets target ~50k context and auto-attach mmproj.
- [x] Show download progress in the GUI and in the CLI — GUI via `Snapshot.Download` (%, bytes, speed); CLI polls `GET /download` and draws a bar.
- [x] Verify downloads (size and/or checksum) and report failures clearly — size checked against the Hub listing; sha256 against the LFS `oid`; mismatch removes the corrupt `.part` and reports both hashes.
- [x] Support authenticated/private repos via a HF token (stored safely, never committed) — `settings.HFToken()` reads `HF_TOKEN`/`HUGGING_FACE_HUB_TOKEN`, then a 0600 `hf_token` file next to config.json, then the standard HF cache; never written into configs/presets.
- [x] Save the downloaded model path into the config automatically after download — writes a tuned `configs/<alias>.json` (~50k ctx, hardware-fitted `-ngl`) and a `library/<id>.json` (category `Downloaded/<repo>`), and persists `modelsDir`.
- [x] Handle partial/failed downloads and cleanup — cancelled/failed downloads keep the `.part` for resume; a checksum mismatch removes it; retries on transient errors.
- [x] Document VRAM/RAM requirements next to each suggested model — `hf.Entry.MinVRAMGiB` / `MinRAMGiB`, shown by `llamaherd download list` and in the GUI panel.
- [x] Detect and handle vision + MTP on download — a matching `mmproj*.gguf` vision projector is attached automatically; an embedded MTP (NextN) head flips the generated preset to `--spec-type draft-mtp`; a separate `fastmtp*.gguf` / `*mtp-draft*` sidecar is downloaded, recorded in `library/<id>.json` (`mtpDraft`) and surfaced in the model list, and is usable via `advanced.draftModel` (`--spec-draft-model`).

## 5. Cross-platform support (Linux / Windows 11 / macOS)

- [x] Audit all OS-specific code (`nvidia-smi` calls, `/proc`, WSL IP detection, paths, `exec` usage) — process listing / RSS / total RAM / kill / detach / disk-free are now platform-split (`control/proc_*.go`, `doctor/disk_windows.go`, `main/daemon_attr_*.go`); the whole module cross-compiles for `linux`, `darwin` and `windows` with `CGO_ENABLED=0`.
- [x] Make WSL-specific logic (WSL IP, WSLg launch) a Linux-only code path with graceful fallback — `settings.IsWSL()` / `settings.LocalIPv4()`; the WSL IP row is populated only inside WSL and is otherwise blank ("—"). WSLg launch stays in the Windows shortcut scripts.
- [x] Verify the Wails build tags work on each OS (currently `webkit2_41` on Ubuntu 24.04) — Linux uses `webkit2_41`/`webkit2_40`; Windows/macOS build with `desktop,production` only; `install.sh`/`install.ps1`/`release.yml` already select per OS.
- [x] Provide per-OS build instructions and CI matrix — README "Building from source (manual)" + "Platform support"; `.github/workflows/ci.yml` runs `go vet` / `go test` / `go build` on `ubuntu-24.04`, `macos-14` and `windows-latest` (CGO off) plus the tagged Linux GUI build.
- [x] Document known platform limitations explicitly — README "Platform support" (paths, WSL IP, process discovery/stop, RAM reporting, update scripts).
- [ ] **Test process control on Windows** (process groups / SIGTERM vs. taskkill semantics) — code is written (TerminateProcess, `CREATE_NEW_PROCESS_GROUP`, `DETACHED_PROCESS`) but needs a real Windows run: Start/Stop a model, `ForceStopLlama`, and `llamaherd daemon` detaching from the launching console.
- [ ] **Test process control on macOS** (signals, sandbox quirks) — code is written (`pgrep`/`ps` discovery, `SIGTERM`→`SIGKILL`); needs a real macOS run: Start/Stop, `ForceStopLlama`, RSS/GPU rows, and `doctor` under Gatekeeper/notarization.
- [ ] Verify `scripts/install.sh` end-to-end on macOS (Homebrew vs. manual deps, app bundle, `launchctl`/login item) and on Windows `install.ps1 -Mode native`.
- [ ] Confirm GGUF scanning + relative model paths behave on case-insensitive filesystems (Windows/macOS).
- [ ] Sanity-check `doctor` output on a machine with an AMD/Intel GPU (only NVIDIA `nvidia-smi` is probed; Metal/CUDA-free paths should still report cleanly).


## 6. Runnable CLI/API packaging

- [x] Confirm the same binary works as CLI (`llamaherd list/status/open/stop/...`) and GUI on all platforms — verified CLI/headless-daemon end-to-end on Linux; GUI builds on Linux/Win/macOS.
- [x] Provide a headless/server-only mode (no GUI) for remote or CI use — `llamaherd daemon`; the CLI now spawns it with the `daemon` argument (previously it re-launched the GUI).
- [x] Document the loopback control API (endpoints, auth, port) for AI agents — README "Control API" table (bound to localhost, currently unauthenticated).
- [x] Consider an API auth token so the local API is not wide open — optional bearer token via `LLAMAHERD_API_TOKEN` / `apiToken`; loopback-only and open by default, enforced by middleware, sent automatically by the CLI.
- [x] Add `--help`/`--version` output for every subcommand — top-level `llamaherd --version`, per-subcommand `llamaherd <cmd> --help`, and `llamaherd version`.

## 7. README (the sharing document)

- [x] Write a top-level `README.md` (initial draft added — project details, features, requirements, usage, project layout, license). Remaining refinements:
  - [x] Project description and one-line pitch. *(initial done)*
  - [ ] Screenshot/screencast. *(placeholder only — needs a real capture)*
  - [x] Feature list.
  - [x] Requirements — split into "always required (runtime)", "prebuilt binary" (nothing extra), and "building from source" (Go + C toolchain + platform GUI libs); covers OS, Go version, GPU, disk.
  - [x] Installation — Linux (Ubuntu/WSL), Windows 11, macOS.
  - [x] First-run / setup (set paths, download a model).
  - [x] Build-from-source instructions per OS. *(Linux/WSL, Windows, macOS)*
  - [x] Usage: GUI tour + CLI examples + control API examples. *(initial)*
  - [x] Configuration reference (config file + env vars).
  - [x] Troubleshooting / FAQ (VRAM guard, missing deps, port in use, WSL quirks).
  - [x] Roadmap / link to BACKLOG.md.
  - [x] License and acknowledgements.
- [x] Add a short `QUICKSTART.md` for the impatient.
- [x] Add a `docs/` folder for deeper topics (architecture, API, packaging) — `docs/architecture.md` (component map); the API reference lives in the README.

## 8. Build, CI, and releases

- [x] Add a GitHub Actions workflow that builds on Ubuntu (and matrix Windows/macOS) — `.github/workflows/ci.yml` (Linux vet/test/build + `install.sh --dry-run`) and `.github/workflows/release.yml` (per-OS matrix).
- [x] Run `go test ./...` in CI and gate merges on it — CI now runs plain `go test ./...` (no skips) on the Linux/macOS/Windows matrix.
- [x] Add a config-consistency test that does not depend on a real GPU (current VRAM test skips when no GPU) — `TestBuiltinPresetsConsistent` (built-in preset shape + generated command line) and `TestPresetSaveLoadRoundTrip` / `TestPresetAbsolutePathsAreMadePortable` (config persistence, GPU/model-free); `TestAllPresetsFitBudget` now skips presets whose model file is absent.
- [x] Fix the currently failing `TestAllPresetsFitBudget` (two `qwen3.8-27b-uncensored-local` presets exceed the budget) or adjust presets — adjusted the machine-local presets to the safe `-ngl` for their context/KV, and made the test skip presets without local model files instead of failing spuriously.
- [x] Add a release workflow that uploads cross-platform binaries to GitHub Releases — `.github/workflows/release.yml` on `v*` tags (stable asset names).
- [x] Tag semantic versions (`v0.1.0`, …) and keep a `CHANGELOG.md` — `CHANGELOG.md` added; `v0.1.0` tagged; the version is stamped into the binary via `-ldflags -X main.version=<tag>`.
- [x] Add reproducible build notes / pinned toolchain (Go version, Wails version) — README "Reproducible builds" (Go from `go.mod`, Wails `v2.16.0` + `go.sum`, prebuilt frontend, version stamping).

## 9. Quality, testing, and safety

- [ ] Add unit tests for config load/save, model scan, GGUF parsing, VRAM estimate.
- [ ] Add integration tests for the control API.
- [ ] Add a lint step (`gofmt`, `go vet`, optionally `golangci-lint`).
- [ ] Verify the VRAM pre-flight guard behavior is documented and testable without a GPU.
- [ ] Sanity-check that no secrets, tokens, or personal paths leak into logs or commits.
- [ ] Review error messages for clarity and English-only consistency.

## 10. Documentation for contributors and users of the API

- [ ] Document the Wails backend contract (`GetState`, `SelectModel`, `StartServer`, …) and events (`state`, `log`).
- [ ] Document the shared app-control state and how GUI/CLI/API stay in sync.
- [ ] Add architecture diagram (frontend ↔ Wails ↔ control ↔ llama-server).
- [ ] Add a "how to add a new preset" guide.
- [ ] Add a "how to add a new model/library entry" guide.

## 11. Nice-to-haves after the basics work

- [ ] Auto-update mechanism for the app itself.
- [ ] Auto-update/rebuild for llama.cpp from the GUI (already partially present — document it).
- [ ] Multi-user / multi-instance support (beyond the current singleton).
- [ ] Model library curator: import, tag, and deduplicate models.
- [ ] i18n framework (keep English as default, allow translations).
- [ ] Telemetry opt-in (or explicitly none) documented in the README.
