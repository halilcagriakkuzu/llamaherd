# AGENTS.md

Guidance for AI coding agents and human contributors working on **LlamaHerd**, a
desktop control panel (GUI + CLI + loopback API) for
[llama.cpp](https://github.com/ggml-org/llama.cpp)'s `llama-server`.
See [`README.md`](README.md) for users and [`docs/architecture.md`](docs/architecture.md)
for the design; this file is the short "how to work here" list.

## Stack

- Go (version in `go.mod`), one binary that is both the GUI and the CLI/daemon.
- GUI: **Wails v2** with a hand-written vanilla HTML/JS/CSS frontend in
  `frontend/dist/` (no npm, no framework, no build step; it is embedded with `go:embed`).
  The frontend talks to Go through `window.go.main.App.*` and `window.runtime.EventsOn`.
- Local runtime data lives outside the repo (see "Paths"); `configs/`, `library/`,
  `models/` and `build/` are git-ignored.

## Build, run, test

```bash
# GUI build (Linux needs libgtk-3-dev and libwebkit2gtk-4.1-dev)
CGO_ENABLED=1 go build -tags webkit2_41,desktop,production -o build/bin/llamaherd .
# macOS / Windows: -tags desktop,production

go vet ./...
go test ./...                 # GPU-free and model-free; run with -race before pushing
golangci-lint run ./...       # config in .golangci.yml; CI runs it too
gofmt -l .                    # must print nothing

# The whole module must keep cross-compiling (CI checks this):
CGO_ENABLED=0 GOOS=windows go build ./...
CGO_ENABLED=0 GOOS=darwin  go build ./...
```

Run the GUI with no arguments, the CLI with a subcommand (`llamaherd help`), or the
headless core with `llamaherd daemon`. `llamaherd doctor` checks the environment.

## Code map

| Path | Role |
| --- | --- |
| `app*.go` | The Wails-bound `App`: the single controller and snapshot state machine |
| `cli*.go` | CLI subcommands (thin HTTP client of the control API; spawns the daemon if needed) |
| `api/` | Loopback control API (`api.Handler`, optional bearer token) |
| `control/` | `llama-server` lifecycle, metrics, tokens/s, WSL bridge, VRAM guard |
| `store/` | GGUF header parsing, model scan, `configs/<alias>.json` and `library/` persistence |
| `settings/` | Paths, env vars, user config file, HF token, WSL detection |
| `doctor/` | Environment checks |
| `hf/` | Hugging Face catalog and resumable, sha256-verified downloads |
| `schema/` | Shared data types (`Config`, `Preset`, `Model`, `Server`) |
| `internal/ggufgen/` | Writes tiny synthetic GGUF files for tests |
| `scripts/` | Installers, updater, icon/resource generators, UI screenshot harness |

OS-specific code is split with build tags (`control/proc_*.go`, `doctor/disk_*.go`,
`daemon_attr_*.go`, `console_*.go`, `settings/platform.go`).

## Conventions

- **One controller.** State changes go through `App` methods so the GUI, CLI and API
  never diverge. A new capability usually means: an `App` method → an `api.Control`
  method + endpoint (+ test with the fake in `api/api_test.go`) → a CLI subcommand →
  UI wiring → a `CHANGELOG.md` entry and README / `docs/cli-and-api.md` update.
- **Frontend:** keep element IDs stable (`main.js` addresses them directly), keep it
  dependency-free, and keep both themes working (colours come from CSS variables).
- **I/O boundaries:** `net/http` and `os/exec` belong in `control/`, `api/`, `hf/`,
  `doctor/`, `settings/` and `cli*.go`, not in UI-facing code.
- **Messages:** user-facing text and errors are English; error strings are lower-case
  without trailing punctuation (staticcheck ST1005).
- **No personal data in the repo:** no absolute home paths, usernames, tokens or
  machine-specific defaults. Presets store model paths relative to the models dir.
- **Safety:** the VRAM pre-flight guard (`control/vram.go`) refuses to start a server
  predicted to exceed 90% of GPU memory. Do not weaken or bypass it; if you change the
  estimate, update the tests.
- Don't leave dead code or commented-out blocks; match the surrounding style.

## Testing without a GPU or real models

- Build GGUF headers with `internal/ggufgen` (`PadBytes` makes sparse, realistic file
  sizes) and fake the card size with `control.SetTotalVramForTesting`.
- Tests that start servers use a `/bin/sh` script as a fake `llama-server`
  (see `app_integration_test.go`).
- **Isolate settings in tests**: set `LLAMAHERD_CONFIG`, `_MODELS_DIR`, `_DATA_DIR`,
  `_CONFIGS_DIR` and `_LIBRARY_DIR` (see `libApp` in `app_library_test.go`). `ConfigsDir()`
  and `LibraryDir()` prefer a relative `./configs` / `./library` when it exists, so an
  unpinned test can write into a developer's real data.
- Frontend changes: `scripts/uishot/shoot.sh <outdir> [light|dark]` renders
  `frontend/dist` with a mocked backend (needs a Chromium; Playwright's cache is
  auto-detected) — use it to review UI changes and to refresh `docs/screenshots/`.

## Paths and configuration

Resolution order is always: environment variable → user config file → built-in default.
See [`docs/configuration.md`](docs/configuration.md) for the full table. Notable defaults: models dir
`~/LlamaHerd/models`, control API `127.0.0.1:8517`, per-user data dir
(`configs/`, `library/`) under the OS data directory. A `serverBin` of the form
`wsl:<distro>:/path` lets the Windows app drive a `llama-server` inside WSL.

## Updating the model catalog

`hf/catalog.json` is the curated download list. It is embedded in the binary **and** fetched
by running apps from the `main` branch (`hf.DefaultCatalogURL`), so merging a change to it
updates every install without a release.

1. Edit `hf/catalog.json` (exact repo + file path, `family` to group quants) and bump `updated`.
   `quality` ranks entries for the "best quality" pick (higher wins; only the order matters). The
   shipped values are parameters × a precision factor (1.0 at ≥ 4.5 bits/weight, 0.1 at ≤ 1.8);
   overwrite them freely, `0`/missing falls back to that formula.
2. Get real numbers instead of guessing: `go run ./internal/catalogtool` reads each model's
   GGUF header over HTTP Range and prints the file size, minimum VRAM/RAM (51200-token
   context, q8_0 KV cache, 90 % VRAM guard) and `kvMiBPer1k` / `trainedContext`.
   Add new repos/quants to the `families` list in that tool first.
3. `go test ./hf` validates the file (`TestEmbeddedCatalogValid`).

Only single-file quants can be listed (the downloader does not handle multi-part shards).
Models needing a custom llama.cpp fork are not listed.

## Releases

1. Update `CHANGELOG.md` (move *Unreleased* under the new version).
2. Regenerate assets if needed: `python3 scripts/make-icon.py`,
   `scripts/make-winres.sh` (Windows exe icon + version info).
3. Tag `vX.Y.Z` on `main` and push the tag; `.github/workflows/release.yml` builds
   Linux/macOS/Windows archives with stable asset names
   (`llamaherd_<os>_<arch>.tar.gz|zip`) and publishes the GitHub Release. macOS and
   Windows builds are best-effort and never block the release.
