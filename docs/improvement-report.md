# LlamaHerd — improvement report

_Review date: 2026-10-20 · code at `5f0255a` (0.3.1) · review scope: architecture, GUI/UX,
control API/CLI, reliability, security, distribution._

> **Implementation status (2026-10-20):** the top five reliability/security items
> (§2.1–§2.5, §3.1), all eleven UX quick wins (§4.1), the three follow-ups
> (model details, validation/auto-fit, live charts), the start-timeout fix (§2.6), the
> reasoning-budget wiring (§2.8), local model import (with mmproj) and stored benchmarks are
> implemented; they are marked ✅ below. The remaining items are still open.

## How this was produced

- Read the controller (`app*.go`), `control/`, `store/`, `hf/`, `doctor/`, `settings/`,
  `api/`, the CLI and the whole frontend (`index.html`, `main.js`, `styles.css`).
- Cross-checked the README, `docs/*`, `BACKLOG.md` and screenshots.
- Ran `go vet ./...` and `go test ./...` — all packages pass.
- Probed concurrency with a temporary `-race` test (removed afterwards): the race detector
  reports real data races in `App` state (details in §2.1).

The project is in good shape: a clear single-controller design, useful docs, GPU-free tests,
and a genuinely nice safety feature (the VRAM pre-flight). The recommendations below are
ordered by impact, not by how easy they are.

---

## 1. What is already strong (keep it)

- **One controller, three front ends.** GUI, CLI and API share `App`; a change made by curl
  appears in the window immediately.
- **The VRAM guard.** A rare, opinionated safety feature. Keep the 90 % rule and the tests.
- **Zero-dependency frontend** embedded with `go:embed` — fast, no node/npm toolchain.
- **GGUF header parsing without llama.cpp**, used for MTP detection, KV sizing and the VRAM
  estimate. This is the foundation for many features below.
- **Resumable, sha256-verified downloads** with a hardware-scored catalog.
- **Honest backlog** and OS-specific files behind build tags.
- **Docs** are better than most hobby projects: quickstart, configuration, WSL, VRAM guard.

---

## 2. Reliability and correctness

### 2.1 Serialize `App` state (P0 — correctness) — ✅ done

**Evidence:** `App` fields (`cfg`, `models`, `presets`, `selModel`, `selPreset`, `dirty`,
`warn`, `runCfg`, `cats`) are read and written from several goroutines without a lock:

- Wails binding calls and HTTP API handlers mutate them (`UpdateConfig`, `Open`, …).
- `control.Manager` callbacks (`emitState` on health flip / every 5 s metrics tick) call
  `snapshot()` concurrently (`control/server.go:354`, `app.go:587`).
- The download install path calls `initModels` / `initPresets` from the download goroutine
  (`app_download_install.go:122`).

A `-race` probe that only called `UpdateConfig` and `snapshot` concurrently produced
`WARNING: DATA RACE` at `app.go:396/584/591/611`. The existing global `libMu` covers only
library edits, not the rest.

**Impact:** corrupt snapshots, inconsistent presets, random crashes under normal use (a
metrics tick every 5 s is enough). This is the highest-priority engineering fix.

**Suggested fix:** add a `stateMu sync.RWMutex` (or a single owner goroutine with a command
channel — the "actor" variant removes this class of bug permanently):

- Exported mutating methods take the write lock, copy state, release, then `emitState`.
- `snapshot()` takes the read lock and copies everything it needs.
- Never call `emitState` while holding the lock (it calls `snapshot`).
- Add a `TestConcurrentAPI` that hammers `GET /status` while calling `/open`, `/config/*`,
  `/model/*`; run it under the existing CI `-race`.

### 2.2 Alias collisions silently overwrite data (P1) — ✅ done

`store.DerivedAlias` (`store/scan.go:144`) and library IDs (`store/library.go:75`) use only
the file **base name**. Two models named `model.gguf` in different folders map to the same
`library/<id>.json` and `configs/<alias>.json`; the second silently overwrites the first.
This is also the root of the "case-insensitive filesystem" backlog item.

**Fix:** derive the ID from the path relative to the models dir (sanitized + short hash), or
detect duplicate aliases during scan and disambiguate (`-2`, `-3`) with a one-time migration
for existing `library/` entries. Add a test with two same-named files in different folders.

### 2.3 Fix or fail fast on path changes while running (P1) — ✅ done

`SetPaths` → `reloadSettings` creates a **new** `control.Manager` (`app.go:166`) while the old
one may own a running `llama-server`. The process keeps running but is orphaned: the UI no
longer sees or can stop it.

**Fix:** refuse path changes while status is `starting`/`running` (with a clear message), or
carry the live manager/process handle over. Add an integration test: start a fake server,
call `SetPaths`, assert Stop still works.

### 2.4 Auto-fit context, not a fixed 51 200 (P1) — ✅ done

`buildDownloadPreset` hardcodes `Context: 51200` (`app_download_install.go:148`) and then only
tunes `-ngl` (`SafeGpuLayers`). On a smaller card this can force GPU layers to a tiny number
(or still fail the guard) even though a 16k context would run entirely on the GPU. The catalog
already computes `MaxContext` (`hf/recommend.go:86`) but the preset ignores it.

**Fix:** when creating any default/download preset, binary-search the largest context that
passes `EstimateVram` with the selected `-ngl`, capped by the GGUF's trained context. Show the
choice in the UI ("32k context, all layers on GPU"). Add a test with synthetic GGUFs and a
faked card size.

### 2.5 No model rescan and no metadata cleanup (P1) — ✅ done

`initModels` is called at boot and after a download only. Dropping a `.gguf` into the models
folder while the app runs requires a restart; there is no UI button, CLI command or API
endpoint. Conversely, deleted files leave stale `library/` meta forever.

**Fix:** `App.RescanModels()` → `POST /rescan` → `llamaherd rescan` → refresh icon in the
Models header. Optionally watch the directory (fsnotify) or rescan on window focus. Add an
"orphaned metadata" cleanup and show missing-file rows instead of dropping them silently.

### 2.6 A start can hang in `starting` forever (P2) — ✅ done

`awaitHealth` gives up after 60 s but only logs (`control/server.go:320`); the manager stays
`starting`, Start is disabled, Stop is disabled in the UI, and there is no per-process force
stop (only "Force stop all"). A slow first load looks like a freeze.

**Fix:** add a `StatusFailed`/"not responding" transition or expose a `Retry`/`Cancel start`
action; make Stop available in `starting` with a force option. Surface a toast/banner with
the last 20 server log lines.

### 2.7 Bound logs and batch UI updates (P2)

- `appendLog` never removes lines (`main.js:128`): a long-running server grows the DOM
  indefinitely. Cap at ~5 000 lines, add Clear/Copy/Export/Search, and an auto-scroll pause.
- Every snapshot rebuilds the whole model list, preset select and form (`render()` →
  `renderModels`), including every 5 s from the metrics tick and on every keystroke. This
  causes flicker, lost scroll position and occasionally lost focus. Add a state revision /
  dirty-section flags, or diff by key, and throttle rendering to `requestAnimationFrame`.
- Persist logs to a rotating file under the data dir (opt-in) so a crash can be diagnosed
  after the window closes.

### 2.8 Dead config field (P3) — ✅ done (wired to `--reasoning-budget`; 0 = unset)

`schema.Basic.ReasoningBudget` exists but is not in the form, `BASIC_KEYS`, or `BuildArgs`.
Either wire it to `--reasoning-budget` or remove it; a persisted-but-ignored field is
confusing when presets are hand-edited.

---

## 3. Security

### 3.1 The loopback API is CSRF-able by default (P0/P1) — ✅ done

The API has no token by default (`withAuth`, `api/api.go:283`) and does not validate `Origin`,
`Host` or `Content-Type`. Any web page the user visits can issue a "simple" cross-origin
`fetch` with `Content-Type: text/plain` and a JSON body; `json.Decoder` ignores the content
type, so `POST /open`, `/stop`, `/config/remove`, `/download` all succeed. The response is
unreadable cross-origin, but the side effects are real (start/stop servers, delete presets,
download files).

**Fix (cheap, no UX regression):**

1. Reject requests with a non-empty `Origin` that is not `http://127.0.0.1:<port>` /
   `http://localhost:<port>`.
2. Require `Content-Type: application/json` for POST (forces a CORS preflight).
3. Validate the `Host` header against loopback addresses (DNS-rebinding defence).
4. Prefer `Authorization: Bearer`; deprecate `?token=` (it leaks into shell history/logs).
5. On first GUI run, generate a random API token into the `0600` config and have the CLI read
   it. Keep "no token" as an explicit `apiToken: "off"` choice if backward compatibility matters.

Add an API test for each rejection.

### 3.2 Validate download refs and destinations (P2)

`hf.ParseRef` accepts arbitrary `owner/name` strings; `../../x` is split into a repo that is
then joined under the models dir (`app_download_install.go:23`). `filepath.Base` protects the
file name, but validating `owner`/`name` (letters, digits, `._-`) and checking the final
destination stays under `ModelsDir()` is cheap defence in depth. Also fuzz `parseGGUF`
(untrusted files) — it already guards implausible counts, a fuzz target would lock that in.

### 3.3 Document the security model (P2)

Add `docs/security.md`: loopback-only API, token, what the API can do, how to disable it, and
what is sent to Hugging Face/GitHub (catalog refresh and downloads). This pairs naturally
with the backlog's privacy statement ("no telemetry").

---

## 4. UX improvements

### 4.1 Quick wins (≈1–3 days each)

| # | Change | Why it matters | Status |
| --- | --- | --- | --- |
| 1 | **"Open chat" button** (`runtime.BrowserOpenURL` to `http://host:port/`) | llama-server ships a full web UI. Today LlamaHerd starts a server and leaves the user to find a client. This is one line and a huge completion of the workflow. | ✅ |
| 2 | **File/folder pickers** for `llama-server` binary and models dir (`runtime.OpenFileDialog` / `OpenDirectoryDialog`) | The first-run screen asks users to type absolute paths. Desktop apps should browse. Also add a picker for `--mmproj`, `--spec-draft-model`. | ✅ |
| 3 | **Rescan models** button / API / CLI (see §2.5) | Basic model management; currently a restart is the only way. | ✅ |
| 4 | **Delete / reveal / copy path** in the model ⋯ menu | Downloaded a wrong quant? There is no way to remove it or open its folder from the app. Confirm, optionally keep presets, move to trash where possible. | ✅ |
| 5 | **Search + filter + favorites** in the model list | Category drill-down does not scale past a dozen models. Add a search box (name/alias/path), filter chips (vision, MTP, quant, fits), "recently used", star/pin. Persist in `library/`. | ✅ |
| 6 | **Restart button** and "config changed — apply & restart?" | Editing while running is common; today you must Stop then Start. Show which preset/model is actually running (not just selected). | ✅ (Restart uses the current form; the button tooltip says so) |
| 7 | **Error toasts** instead of log-only failures | `call()` swallows rejections into the log (`main.js:117`). Users miss "port in use", "preset exists", "download failed". A small toast/snackbar with the message is enough. | ✅ |
| 8 | **Logs toolbar**: clear, copy, download, filter (errors/server), pause auto-scroll, line cap | The log panel is a terminal with no controls. | ✅ |
| 9 | **Installed badge + disk space** in the download catalog | Rows do not say "already installed"; users can re-download. Show installed size, free disk, and a one-click "open folder". | ✅ |
| 10 | **Preset duplicate / reset to default / export** | Presets are the core workflow; cloning is the most natural action and is missing. Export/import JSON makes sharing configs possible. | ✅ (import is not included) |
| 11 | **Keyboard shortcuts + focus trap** | Ctrl/Cmd+Enter start, Ctrl+S save, Ctrl+F search, Esc (exists). Trap focus inside modals and add `aria-live` for status/log so the app is usable with a keyboard/screen reader. | ✅ |

### 4.2 Medium (≈1–2 weeks each)

- **Model details drawer.** ✅ `store.GGUFInfo` already parses architecture, block count,
  trained context, head counts, KV geometry, file size. Show them with the quant, source, path,
  alias, paired mmproj/MTP, last used, and a "copy path" action. This turns the app from a
  launcher into a model manager and justifies the library layer.
- **Validation + live budget.** ✅ Show warnings inline: context above the model's trained
  context, `-ngl` above layer count, invalid port, missing sidecars, and "this preset will
  exceed VRAM by X". Today the only feedback is after pressing Start (or the guard banner).
  Also expose "Auto-fit" next to `-ngl`/context (uses §2.4).
- **Live charts.** ✅ The status box shows one number per metric. Keep a 10-minute ring buffer
  of VRAM, RAM and tok/s (and GPU util/temp via `nvidia-smi`) and draw sparklines. Use
  `--metrics` / `/metrics` for prompt-eval, slots and queue depth instead of parsing log lines
  only. (Ring buffer + sparklines are done; `/metrics`-specific counters remain a follow-up.)
- **Download queue + multi-part support.** One download at a time, and multi-part GGUFs are
  explicitly disabled (`main.js:731`). Add a queue, pause/resume (the `.part` machinery
  supports it), retry, per-file destination category, and shard assembly with hash verification.
- **Download/install from local files.** ✅ "Import model…" that copies or moves a `.gguf` into
  the models dir, creates the library meta and a fitted preset. Detect duplicates by size +
  sha256 on demand. This is the backlog's "library curator" and a common onboarding path. (Implemented with an
  optional mmproj picker; the API/CLI expose `POST /model/import` and `model import`.)
- **System integration.** Minimize-to-tray while the server runs, "start server on launch",
  and OS notifications for ready/crash/download done. Desktop users expect the app not to die
  when the window closes if a server is running.
- **Onboarding v2.** Besides path selection: detect `llama-server` in PATH/common locations,
  offer "download a prebuilt llama.cpp" with GPU variant detection, "build from source", and a
  starter model. Validate before saving; show the resulting model count and free disk.

### 4.3 Bigger bets

1. **Router / multi-model support (strategic).** llama.cpp now has router mode
   (`--models-dir`, model presets, dynamic load/unload) alongside the classic one-model server.
   LlamaHerd's manager is deliberately single-process/single-port. A router mode would deliver
   the backlog's "multi-instance" goal with far less memory churn: one server, many models,
   switch without restart. Suggested spike: a `serverMode: single|router` setting, `GET
   /v1/models` for loaded models, and per-model presets in the INI. If router mode is too
   invasive, at least add multi-instance with automatic port allocation (default 8090
   collides today).
2. **Built-in playground (or embedded web UI).** Start with the "Open chat" button; later a
   small Chat tab that calls `/v1/chat/completions` with streaming, system prompt, temperature
   and a token/s readout. This makes "pick model → test it" a single window and is the most
   requested feature in comparable tools (LM Studio, Jan, llama.app).
3. **Benchmark and preset comparison.** ✅ "Benchmark this preset" runs a fixed prompt set at 2–3
   context sizes and stores prompt tok/s, generation tok/s and peak VRAM per model/preset.
   Compare presets side by side. This differentiates LlamaHerd from launchers and directly uses
   the metrics you already collect. (Implemented: fixed-prompt `/completion` probe, stored per
   model/preset/config fingerprint, fastest-first table.)
4. **Agent integration.** The project explicitly targets AI agents. Add an OpenAPI document
   (`GET /openapi.json`) and a small **MCP server** that exposes `list_models`, `status`,
   `start`, `stop`, `download`, `set_preset`. An MCP server is a natural fit and a strong
   README feature; also add SSE (`GET /events`) so agents/CLI can stream logs instead of
   polling `/status`.
5. **Model provenance and licensing.** Show the HF license/gated flag and the source URL in
   the details drawer; warn before starting gated models. This is a small trust feature that
   matters for a downloader.

---

## 5. API and CLI

- Add missing verbs: `POST /restart`, `POST /rescan`, `GET /logs?tail=n`, `GET
  /models/{alias}` (details), `DELETE /models/{alias}` (file), `POST /open` returning the
  resolved URL.
- **Version the API** (`/v1/...` or a `schemaVersion` in `/status`) before third-party clients
  depend on it. Keep the old paths as aliases.
- **`GET /openapi.json`** generated from a small spec, plus a typed client. The API is already
  the most script-friendly part of the product; document it as a first-class interface.
- **CLI**: `llamaherd chat` (talk to the running server), `logs -f`, `restart`, `rescan`,
  `bench`, `open --browser`, `--json` on `list`/`status`, shell completions. Today `status`
  dumps the full snapshot; a `--summary` is friendlier for humans.
- **Fix the documented partial-update behaviour.** `docs/cli-and-api.md` says a partial
  config is merged for `/config/update`, but `UpdatePreset` replaces the preset wholesale
  (`app_api.go:150`). Either merge with `mergeConfig` (as `AddPreset` does) or correct the doc.
- **Stable error shape.** `{ok:false,error}` is good; add an `errorCode` (e.g.
  `vram_blocked`, `port_in_use`, `not_found`) so clients can react without parsing text.

---

## 6. Engineering quality

- **Concurrency tests** (§2.1) and `go test -race` in CI already exists — add the missing
  concurrent API test.
- **Fuzz tests** for `store.parseGGUF` and `hf.ParseRef`; both parse untrusted input.
- **Snapshot contract test**: marshal a snapshot and assert the JSON keys/shape, so the
  frontend contract cannot drift silently.
- **UI smoke test**: the `scripts/uishot` harness already mocks the backend. Extend it to click
  Start/Stop, open dialogs and assert visible state; run it in CI with a headless Chromium.
- **Structured logging**: one log line per event with a `component:` prefix exists; add a
  rotating file sink and a "copy diagnostics" button (doctor report + last logs + versions).
- **Benchmarks** for `ScanModels`/`ReadGGUFInfo` on a large fake library; the GGUF cache is
  path-keyed and never invalidated, which is fine for immutable files but should be documented
  (or keyed by size+mtime).

---

## 7. Distribution, docs and trust

- **App update check** (already in the backlog): query the GitHub releases API once per day,
  show a dismissible "0.4.0 available — release notes" banner. Optional self-update later.
- **Signing/notarization**: Windows SmartScreen and macOS Gatekeeper are the first-run
  experience. Sign the exe and notarize the `.app` before promoting the project.
- **Packages**: `.deb`/`.rpm`, winget/`.msi`, Homebrew cask/`.dmg`. The script installers are
  good, but package managers are how desktop users install.
- **Privacy/security docs**: "no telemetry", exactly which hosts are contacted (HF, GitHub raw
  catalog), and how to run fully offline (`LLAMAHERD_CATALOG_URL=off`).
- **Docs**: refresh screenshots after 4.1; add an FAQ ("Where is the chat?", "How do I delete a
  model?", "Why did Start get blocked?"), a comparison table vs LM Studio/Ollama/llama.app, and
  a short "connect from Open WebUI/Continue/Cline" page with copy-paste snippets.
- **i18n** is in the backlog; the frontend is small and dependency-free, so a tiny
  `data-i18n` dictionary is enough when the UI text stabilizes. Do it after the UX churn above.

---

## 8. Suggested sequence

### v0.4 — "trust and polish" (highest value / lowest risk)

1. Serialize `App` state + concurrent API test (§2.1).
2. API hardening: Origin/Host/Content-Type + token default (§3.1).
3. Quick wins: Open chat, file pickers, rescan, delete/reveal, search, restart, toasts, log
   toolbar (§4.1).
4. Fix `SetPaths` while running and auto-fit context (§2.3, §2.4).
5. Update check + security/privacy docs (§7, §3.3).

### v0.5 — "model manager"

- Import local files, duplicate/disk-usage reporting, download queue + multi-part, installed
  badges, preset export/import.
- Alias-collision migration (§2.2) before users accumulate libraries.

### v0.6 — "multi-model and agents"

- Router-mode spike or multi-instance with port allocation.
- Playground/chat, benchmark + preset comparison.
- OpenAPI + MCP server + SSE events.

---

## 9. Success metrics to watch

- Time from install to first successful response (target: under 5 minutes, no terminal).
- Start failure rate and percentage of failures with an actionable message.
- Downloads that complete vs. fail/cancel; resume success rate.
- Weekly active users who run at least one server, and median models per library.
- Crash-free sessions / race-detector clean CI.
- GitHub issues about "can't find model / can't stop / can't connect".

---

## 10. Things to avoid

- **Do not weaken the VRAM guard** to make a start "just work"; instead make the fit visible
  and tunable (§2.4). An opt-in override should be explicit, per-launch and logged.
- **Do not add a frontend framework** for these features. The existing vanilla stack is an
  asset; small helpers and targeted rendering are enough.
- **Do not make the API token mandatory without giving the CLI/daemon a way to share it** —
  that breaks the "one binary, three front ends" promise.
- **Do not build a full chat client before shipping the "Open chat" link** — llama-server's
  built-in UI already solves 80 % of it.
