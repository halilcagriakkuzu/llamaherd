# LlamaHerd — task plan

Medium-sized, independently shippable tasks derived from
[`docs/improvement-report.md`](docs/improvement-report.md) and
[`BACKLOG.md`](BACKLOG.md). Work one task at a time; each ends with tests,
docs and a `CHANGELOG.md` entry.

## How to use this file

- Pick the next unchecked task (or a parallel batch below).
- Keep the scope to the task; move new ideas back into `BACKLOG.md`.
- A task is done when its **Acceptance** list passes and the repo checks are
  green: `go vet ./...`, `go test -race ./...`, `gofmt -l .`, and
  `CGO_ENABLED=0 GOOS=windows go build ./...` / `GOOS=darwin`.

## Non-goals

- No i18n — the UI stays English.
- No "start server on launch".
- No downloading/building a prebuilt llama.cpp from the onboarding screen
  (only detection of an existing binary).
- No MCP server; the CLI/API remain the agent interface.
- Do not weaken the VRAM pre-flight guard.

## Index

| ID | Task | Size | Depends on |
| --- | --- | --- | --- |
| T01 | Download ref validation + fuzz tests | S | — |
| T02 | Security and privacy documentation | S | — |
| T03 | API contract: partial update + snapshot test | M | — |
| T04 | Persistent logs, log endpoint and diagnostics | M | — |
| T05 | CLI polish: `--json`, `--summary`, completions | S | T04 (for `logs`) |
| T06 | UI smoke tests in CI | M | — |
| T07 | Case-insensitive filesystem behaviour | M | — |
| T08 | Import v2: duplicates, batch, drag-and-drop | M | — |
| T09 | Onboarding: auto-detect `llama-server` | S | — |
| T10 | Model provenance and licensing | M | — |
| T11 | Download queue with pause/resume/retry | M | — |
| T12 | Multi-part GGUF downloads | M | T11 |
| T13 | Top bar/menu refresh + in-app notifications | M | — |
| T14 | OS notifications for ready/crash/download | M | T13 |
| T15 | Non-NVIDIA GPU support | M | — |
| T16 | Benchmark `/metrics` counters | M | — |
| T17 | Benchmark export (CSV/JSON) | S | — |

S ≈ half a day, M ≈ 1–3 days. The estimates include tests and docs.

---

## T01 — Download ref validation + fuzz tests

**Status:** ✅ done (2026-10-20).

**Why:** `hf.ParseRef` accepts arbitrary strings; `../..` is split into a
repo name and later joined under the models directory. `parseGGUF` parses
untrusted files and has no fuzz target.

**Scope**
- Validate `owner`, `name` and the file path: non-empty, no leading `/`,
  no `\`, no `..` segment, no control characters; allow `A-Za-z0-9._-`.
- URL-decode before validation so `%2e%2e` cannot slip through.
- After resolving the destination, assert it stays under `ModelsDir()`.
- Add `FuzzParseRef` and `FuzzParseGGUF` with the existing tests as seeds.
- Add API tests for rejected refs (400/`ok:false`, no download started).

**Acceptance**
- `../../etc/passwd`, `owner/name/../../x.gguf`, empty and absolute paths are
  rejected with a clear error.
- Fuzz targets run out the seed corpus without panics.
- A valid `owner/name` and `owner/name/dir/file.gguf` still work.

**Likely files:** `hf/ref.go`, `hf/ref_test.go`, `hf/resolve.go`,
`app_download_install.go`, `app_download.go`, `store/gguf_test.go`.

---

## T02 — Security and privacy documentation

**Status:** ✅ done (2026-10-20).

**Why:** the security model exists in code but not in docs; "no telemetry"
is only implied.

**Scope**
- Add `docs/security.md`: loopback-only API, bearer token, `Host`/`Origin`/
  JSON guards, what the API can change, how to disable the token, local
  import path access, benchmark data (local only).
- List every outbound host: Hugging Face (downloads/API), GitHub raw
  (catalog refresh), and how to run fully offline
  (`LLAMAHERD_CATALOG_URL=off`, no downloads).
- Add a short "Privacy" section to `README.md` and link the new page.
- Check the final text against the code (no aspirational claims).

**Acceptance**
- A user can answer "what leaves my machine?" from the docs alone.
- Links work from README and the docs table.

**Likely files:** `docs/security.md`, `README.md`, `docs/configuration.md`,
`docs/cli-and-api.md`.

---

## T03 — API contract: partial update + snapshot test

**Status:** ✅ done (2026-10-20).

**Why:** `docs/cli-and-api.md` promises partial merges for `/config/update`,
but `UpdatePreset` replaces the preset wholesale. The snapshot JSON is also an
unversioned frontend contract with no test.

**Scope**
- Make `POST /config/update` merge the supplied config onto the existing
  preset (same rules as `mergeConfig` in `AddPreset`), or explicitly change
  the docs if merge is rejected. Prefer merge.
- Keep the existing behaviour that a full config still replaces cleanly.
- Add `TestSnapshotContract`: build a snapshot and assert all top-level keys
  the frontend reads (`status`, `server`, `models`, `config`, `presets`,
  `benchmarks`, `metrics`, `recentLog`, …) and their JSON types.
- Update CLI docs and add an API test for a one-field patch.

**Acceptance**
- A patch `{"basic":{"port":9000}}` keeps model, context, advanced fields.
- Snapshot contract test fails if a field is renamed or its type changes.
- Docs match the implemented behaviour.

**Likely files:** `api/api.go`, `app_api.go`, `api/api_test.go`,
`app_contract_test.go` (new), `docs/cli-and-api.md`.

---

## T04 — Persistent logs, log endpoint and diagnostics

**Status:** ✅ done (2026-10-21).

**Why:** the log view is capped in memory; after a crash there is nothing on
disk, and the CLI cannot follow logs.

**Scope**
- Write all `App.Log` / server lines to a rotating file under the data dir
  (`logs/llamaherd.log`, e.g. 5 MB × 3). Add an env/config override and an
  opt-out (`logFile: "off"`).
- Add `GET /logs?tail=n` returning recent lines (from the file, with a
  fallback to the in-memory tail when disabled).
- Add `llamaherd logs [-f] [--tail n]`; `-f` polls once per second (no SSE).
- Add a "Copy diagnostics" action (llama.cpp tab or settings): doctor report
  + version/OS + last ~200 log lines to the clipboard.
- Bound memory: keep the existing 5 000-line view cap.

**Acceptance**
- A server crash leaves a readable log file.
- `llamaherd logs -f` streams new lines; Ctrl-C exits cleanly.
- Rotation never grows unbounded; tests cover the size limit.

**Likely files:** `app.go`, `app_llama.go`, new `app_logs.go`, `api/api.go`,
`cli.go`, `settings/settings.go`, `frontend/dist/*`, docs.

---

## T05 — CLI polish

**Status:** ✅ done (2026-10-21).

**Why:** the CLI is the agent interface, but scripting output is inconsistent.

**Scope**
- `llamaherd list --json` and `llamaherd status --summary` (human-readable
  one-screen summary; `status` keeps full JSON by default).
- `llamaherd --json` for other read commands where it makes sense.
- Generated shell completions: `llamaherd completion bash|zsh|fish`.
- Keep exit codes meaningful and errors on stderr.

**Acceptance**
- `list --json` output is valid JSON and round-trips in `jq`.
- `status --summary` fits on one screen and includes status/model/preset/VRAM.
- Completion scripts load in bash/zsh/fish without errors.

**Likely files:** `cli.go`, `main.go`, docs, tests for the formatters.

---

## T06 — UI smoke tests in CI

**Status:** ✅ done (2026-10-20).

**Why:** `scripts/uishot` already renders the real frontend with a mocked
backend, but nothing asserts that the UI works.

**Scope**
- Extend `scripts/uishot` with a scripted scenario: boot, select a model,
  open the import dialog, toggle mmproj, run a mocked benchmark, show the
  failed banner, open logs.
- Assert visible text/state (not pixel comparison) and exit non-zero on
  failure; keep screenshots as artifacts.
- Add a CI job with a headless Chromium/Playwright cache; skip gracefully
  when no browser is available locally.
- Add a `make`-less command documented in `scripts/uishot/README.md`.

**Acceptance**
- The scenario fails when a button id or expected text is removed.
- CI runs it on Linux only (documented).

**Likely files:** `scripts/uishot/*`, `.github/workflows/*`, docs.

---

## T07 — Case-insensitive filesystem behaviour

**Status:** ✅ done (2026-10-20).

**Why:** Windows/macOS treat `Model.gguf` and `model.gguf` as the same file;
alias/config lookups and duplicate detection currently compare strings.

**Scope**
- Normalise paths for comparison (case-fold on Windows/macOS, exact on Linux)
  in model lookup, alias collision detection and library metadata resolution.
- Ensure two models whose names differ only by case do not share a config.
- Reject/handle an import whose destination differs only by case on a
  case-insensitive filesystem.
- Add tests using a small path-comparison helper; add a Windows/macOS CI test
  matrix if the workflow supports it easily (otherwise document the manual
  check).

**Acceptance**
- `store` and `app` tests cover case-only differences.
- Re-importing the same file with different case is refused with a clear
  message, not a silent overwrite.

**Likely files:** `store/scan.go`, `store/library.go`, `store/*_test.go`,
`app_import.go`, `app_library.go`, settings/path helpers.

---

## T08 — Import v2: duplicates, batch, drag-and-drop

**Status:** ✅ done (2026-10-21).

**Why:** basic import exists; users still re-import the same file and have to
pick files one by one.

**Scope**
- Duplicate detection: size first, then sha256 on demand; compare against
  models already in the library. Offer "use existing" / "import anyway"
  instead of a blind copy. Reuse `hf`'s hashing helper where possible.
- Batch import: `ImportModel` accepts a list (or a directory) and returns a
  per-file result; GUI dialog can add several files.
- Drag-and-drop `.gguf` files onto the Models panel (HTML drag events +
  existing binding).
- Keep mmproj pairing per file; a batch may have one common mmproj.

**Acceptance**
- Importing an identical file twice reports "already imported" with the
  existing model alias/path; no duplicate copy.
- Batch import of 3 files creates 3 models and selects the last/first
  consistently; partial failures are reported per file.
- Drag-and-drop works in the GUI smoke test (mock or real binding).

**Likely files:** `app_import.go`, `app_import_test.go`, `api/api.go`,
`cli.go`, `frontend/dist/*`, `store` hashing helpers.

---

## T09 — Onboarding: auto-detect `llama-server`

**Status:** ✅ done (2026-10-21).

**Why:** first-run still asks for an absolute path typed by hand.

**Scope**
- Search order: `PATH`, common build layouts (`~/llama.cpp/build*/bin`,
  project-relative), WSL candidates when relevant, plus an explicit
  `LLAMAHERD_SERVER_BIN`.
- Validate the candidate by running `llama-server --version` (short timeout)
  and show version/build info before saving.
- Add a "Detect" button/step to the first-run dialog and the llama.cpp tab;
  never overwrite a user-provided path silently.
- Report a clear reason when nothing is found.

**Acceptance**
- On a machine with `llama-server` on `PATH`, first-run detects it with one
  click and Start works.
- A stale/broken candidate is rejected with its error, not saved.
- No network access or downloads in this flow.

**Likely files:** `settings/settings.go`, `control/llamacpp.go`, `app_ux.go`,
`frontend/dist/*`, docs/quickstart.

---

## T10 — Model provenance and licensing

**Status:** ✅ done (2026-10-21). Scope changed: licence/gated status only matters at
download time (gated notice in the file list, clear errors for gated 401/403, repo +
licence recorded in `library/<id>.json`). Models already on disk are never checked or
warned about before Start, and there is no manual licence lookup.

**Why:** the app downloads models but does not show licence/gated status or
warn before starting a gated model.

**Scope**
- Fetch model info from the HF API (`/api/models/<repo>`) for the card,
  licence, gated flag, tags and last modified; cache it under the data dir
  and degrade to "unknown" offline.
- Persist source repo/licence in library metadata (`library/<id>.json`).
- Show provenance in the details drawer, with a link to the source URL.
- Warn before Start when the model is gated/licence is unknown, with a
  one-time dismiss or explicit confirmation.
- Keep the network call off the startup path.

**Acceptance**
- A downloaded model shows repo, licence and gated status after a refresh.
- Offline mode shows "unknown" and never blocks Start.
- The warning appears once per model/session and is logged.

**Likely files:** new `hf/modelinfo.go`, `store/library.go`, `schema/schema.go`,
`app_download_install.go`, `app_ux.go`, frontend details/start path, docs.

---

## T11 — Download queue with pause/resume/retry

**Status:** ✅ done (2026-10-21).

**Why:** one download at a time; starting another fails, and there is no
queue or retry.

**Scope**
- Add a persistent in-memory queue (`[]QueuedDownload`) to `DownloadState`
  and the snapshot; `StartDownload` enqueues when something is active.
- Process sequentially; on finish/error/cancel move to the next item.
- Pause/resume: pausing cancels the current transfer (`.part` is resumed on
  resume); queued items stay queued. Retry re-enqueues with backoff once.
- API: `POST /download/queue` (cancel item), `POST /download/pause`,
  `POST /download/resume`; CLI: `download queue|pause|resume`.
- UI: queue list with remove/cancel, current progress unchanged.
- Keep the existing status-bar badge.

**Acceptance**
- Queuing two downloads runs them one after another in order.
- Pause then resume continues from the `.part` size (hash still verified).
- Removing a queued item never touches the active download.
- Tests use a fake HF server / local HTTP test server.

**Likely files:** `app_download.go`, `app_download_start.go`, `app_download.go`
state, `hf/download.go`, `api/api.go`, `cli.go`, `frontend/dist/*`, tests.

---

## T12 — Multi-part GGUF downloads

**Status:** ✅ done (2026-10-21).

**Why:** repo listings show multi-part sets but the buttons are disabled.

**Scope**
- Resolve all shards for a selected set (`*-00001-of-000NN.gguf`), verify the
  count and download each shard into the same folder with sha256 checks.
- Create one library entry/preset whose `--model` points at the first shard
  (llama.cpp finds the siblings).
- UI: show total size, shard progress and a clear failure if one shard is
  missing; API/CLI return the shard list.
- Resume works per shard; a completed shard is skipped on retry.
- Do not add multi-part entries to the curated catalog unless a real model is
  validated (catalog tool support can follow).

**Acceptance**
- A 2-shard fake set downloads completely and the preset loads the first shard.
- Removing/corrupting one shard fails verification before install.
- Queue (T11) treats the whole set as one job.

**Likely files:** `hf/listing.go`, `hf/resolve.go`, `hf/download.go`,
`app_download_install.go`, `frontend/dist/*`, tests.

---

## T13 — Top bar/menu refresh + in-app notifications

**Status:** ✅ done (2026-10-21).

**Why:** the header is getting crowded (Start/Stop/Restart/Chat/theme) and
action results are scattered between toasts and logs.

**Scope**
- Restructure the top bar: brand + status, primary server actions, and a
  single "⋯" menu for secondary actions (theme, diagnostics, update/rebuild,
  paths).
- Group model actions (Import/Download/Rescan/Folder) consistently; keep
  element IDs stable where possible and document changed ones.
- Add an in-app notification center: recent toasts (download done, crash,
  benchmark done, import done) with timestamps, persisted for the session.
- Ensure keyboard/focus behaviour and `aria-live` still work.

**Acceptance**
- All existing actions remain reachable; nothing is cut off at 960 px width.
- Toasts also land in the notification center; dismissing one does not
  dismiss the other.
- `scripts/uishot` scenarios pass (after T06).

**Likely files:** `frontend/dist/index.html`, `main.js`, `styles.css`,
`scripts/uishot/mock.js`.

---

## T14 — OS notifications for ready/crash/download

**Status:** ✅ done (2026-10-21).

**Why:** users minimize the app while a model loads or a download runs.

**Scope**
- Emit host OS notifications for: server ready, server crashed / not
  responding, download finished, download failed, benchmark finished.
- Linux/Windows/macOS backends; if no native backend is available, fall back
  to the in-app notification center only.
- Add a settings toggle (default on) per event class; never notify for
  routine state ticks.
- WSL note: the GUI runs on the Windows host, so notifications are host
  notifications even when `llama-server` runs inside WSL; verify this and
  document any limitation.
- The chosen notification library must be small and cross-platform; evaluate
  the already-indirect `go-toast` dependency before adding a new one.

**Acceptance**
- With the window minimized, a ready/crash event produces one notification.
- Notifications respect the toggle and do not fire twice per event.
- WSL-hosted server still notifies on Windows (manual check documented).

**Likely files:** new `app_notify_*.go`, `settings/settings.go`, `app.go`,
frontend settings UI, docs.

---

## T15 — Non-NVIDIA GPU support

**Status:** ✅ done (2026-10-21).

**Why:** only `nvidia-smi` is probed; AMD/Intel/Apple machines should degrade
cleanly and, where possible, show real numbers.

**Scope**
- Introduce a small GPU provider interface: total VRAM, per-process VRAM,
  utilisation, temperature.
- Implement NVIDIA (existing), AMD (`rocm-smi`), Intel (`xpu-smi`) and Apple
  (unified memory via `sysctl`/`system_profiler`) best-effort providers.
- `doctor` reports which provider is active; unknown values stay "—" and the
  VRAM estimate falls back to the current conservative behaviour.
- All providers are unit-tested with injected command output; no real AMD/
  Intel hardware is available, so document that these paths are untested.
- Do not relax the VRAM guard when memory is unknown.

**Acceptance**
- On a machine with no NVIDIA tools, the app starts normally and shows "—"
  instead of errors; `doctor` names the detected vendor/provider.
- Provider tests cover successful output, missing binary and malformed output.

**Likely files:** `control/metrics.go`, new `control/gpu_*.go`, `doctor`,
`control/vram.go`, tests, docs/troubleshooting.

---

## T16 — Benchmark `/metrics` counters

**Status:** ✅ done (2026-10-21).

**Why:** the benchmark table only stores tok/s and VRAM; llama-server exposes
prompt-eval, slots and queue depth via `/metrics`.

**Scope**
- Add an advanced toggle (or default-on flag) for `--metrics` in the config
  form/`BuildArgs`; document the extra endpoint.
- During a benchmark, scrape `/metrics` before/after and record prompt-eval
  time, slots active/busy, queue depth and (when present) context usage.
- Store the new fields in `schema.Benchmark`; show the most useful ones as
  extra table columns/tooltips.
- Handle older builds without `/metrics` gracefully (fields empty, no error).

**Acceptance**
- A benchmark against a `--metrics` server stores counters; against an older
  build it still completes with empty counters.
- Table/snapshot JSON round-trips the new fields.

**Likely files:** `control/bench.go`, `control/metrics.go`, `schema/schema.go`,
`store/bench.go`, `app_bench.go`, frontend table, docs, tests.

---

## T17 — Benchmark export (CSV/JSON)

**Status:** ✅ done (2026-10-21).

**Why:** results are useful outside the app (spreadsheets, issue reports).

**Scope**
- `GET /benchmarks/export?format=csv|json` returning stable columns/keys.
- GUI: Export button next to Clear; save via a Blob download.
- CLI: `llamaherd bench export [--format csv|json] [--out file]`.
- CSV: RFC 4180 quoting, header row, human-readable timestamps and raw epoch
  in separate columns if useful.

**Acceptance**
- Exported CSV opens in a spreadsheet with one row per result.
- JSON export round-trips through `store.LoadBenchmarks`/`AllBenchmarks`.
- No personal paths beyond what is already stored in the result.

**Likely files:** `app_bench.go`, `api/api.go`, `cli.go`, `store/bench.go`,
frontend bench section, docs, tests.

---

## Suggested batches

**Batch A — hardening (can run in parallel):** T01, T02, T03, T06, T07  
**Batch B — observability and CLI:** T04 → T05, plus T09  
**Batch C — model manager:** T08, T10  
**Batch D — downloads (sequential):** T11 → T12  
**Batch E — desktop UX (sequential):** T13 → T14  
**Batch F — hardware and benchmark:** T15, T16, T17

## Definition of done (every task)

- Tests cover the new behaviour (GPU-free, model-free, no real network).
- `go vet ./...`, `go test -race ./...`, `gofmt -l .` clean.
- Windows/macOS cross-build still works.
- `CHANGELOG.md` updated; CLI/API docs updated when an interface changed.
- No personal paths, tokens or machine-specific defaults in the repo.
