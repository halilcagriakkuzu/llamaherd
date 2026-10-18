# 10 — Phase 2: config persistence + model scan

Status: DONE
Depends on: 09 (design approved)

## Goal

First real-I/O phase-2 slice: `configs/<alias>.json` persistence and the
recursive model scan. NO process control yet (that's task 11).

## Spec

### New `store` package

- `store/scan.go`:
  - `ScanModels(root)` — recursive walk of `/home/halilcagriakkuzu/llm/models`,
    one entry per `*.gguf`; `mmproj*.gguf` (case-insensitive) flagged
    `IsMMProj` (vision sidecar, never a loadable model); sidecars listed right
    after their sibling models; unlabelled aliases left to the app;
    unreadable subtrees skipped silently.
  - `DerivedAlias(path)` — file-name-derived alias (lowercase, `.` → `-`).
  - `ModelsRoot` const.
- `store/presets.go` shapes:
  - `AliasFile{Alias, Presets}` — on-disk shape of `configs/<alias>.json`
    (preset = name + full basic+advanced field set).
  - `LoadAlias` / `SaveAlias(dir, alias, presets)` — real file I/O; save
    refuses empty preset lists; unit-tested in `store/scan_test.go`.

### App integration (`app.go`)

- Startup: scan replaces the hardcoded dummy list (dummy list kept only as
  fallback when the scan is empty); per-alias preset table = built-in presets
  overridden by any `configs/<alias>.json` found, plus a synthesized
  `default` preset for scanned models nothing claims.
- Alias resolution: a built-in preset claiming the model path wins; otherwise
  `DerivedAlias`.
- Save cfg: real write — upserts the current config under the current preset
  name; log lines no longer say `(dummy)`. Save failure logs `save FAILED ...`.
- Load cfg: real read — replaces the alias's presets in memory; missing file
  logs `loaded ... FAILED: ... (nothing changed)`.
- mmproj rows still fill the `--mmproj` advanced field without changing the
  model selection.

## Acceptance

- `go test ./store/` passes against the real models directory.
- Model list shows the real scanned files (known preset models keep their
  preset tables and order-independent aliases).
- Save/load round-trips a modified field through `configs/<alias>.json`
  (verify on disk), and load restores the persisted values after restart.
