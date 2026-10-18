# 06 — Model config: presets, save/load

Status: TODO
Depends on: 03, 04, 05

## Goal

Per-alias named presets (the user's 3 real commands become 3 presets), with
dummy save/load.

## Spec

### Preset data (`dummy/data.go`)

Map: alias -> []preset. Each preset = full basic+advanced field set.

| alias | preset | source |
|---|---|---|
| qwen3.8-27b-local | `default` | command 1: ctx 25556, spec draft-mtp n-max 3, perf on, sleep 300 |
| qwen3.8-27b-uncensored-local | `vision+mtp` | command 2: ctx 32768, mmproj path, no-mmproj-offload, spec draft-mtp n-max 2, spec-draft-ngl all, type-k/v q8_0, cache-ram 4096, sleep 5, perf off |
| qwen3.8-27b-uncensored-local | `plain` | command 3: ctx 32768, sleep 5, no spec, no mmproj |
| qwen3.8-27b-uncensored-hauhaucs | `default` | derived: common flags, ctx 32768, sleep 300, no spec |

### UI (lives in the action row, task 07 owns the row layout)

- Preset `QComboBox` (left of buttons).
- Config path label: `configs/<alias>.json` (right of buttons, small gray text).

### Behavior (all dummy)

- Model selected (task 03): combo gets that alias's preset names; first preset
  applied to the forms; path label updated.
- Preset changed: both forms filled from the preset. Log `preset: <alias>/<name>`.
- Dirty flag: any form edit after preset selection marks the config
  unsaved — path label shows a trailing `*` (`configs/<alias>.json *`);
  selecting another preset or Save/Load clears it. (Phase 2 will persist
  this exact state, so it's part of the design review.)
- `Save cfg` button: log `saved configs/<alias>.json [preset <name>] (dummy)`
  and clear the dirty `*`. No file written.
- `Load cfg` button: log `loaded configs/<alias>.json [preset <name>] (dummy)`
  and re-applies the current preset to the forms. No file read.

## Acceptance

- Selecting each model shows the right preset names; switching presets fills
  both tabs correctly (spot-check: `vision+mtp` has mmproj + cache-ram 4096;
  `plain` has neither).
- Editing a field shows the `*` dirty marker on the path label; switching
  presets or Save/Load clears it.
- Save/Load only produce log lines; path label always matches the selected
  alias.
