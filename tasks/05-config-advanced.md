# 05 — Config: ADVANCED tab

Status: TODO
Depends on: 01

## Goal

Everything else, grouped so the tab stays scannable.

## Spec

Tab 2 titled `ADVANCED`: scroll area with three `QGroupBox`es stacked
(plain sections, 8px apart). Same label style as task 04.

### Speculative decoding

| Field | Widget | Flag | Dummy default (cmd 1) |
|---|---|---|---|
| Spec type | QComboBox `none`,`draft-mtp` | `--spec-type` | draft-mtp |
| Draft n max | QSpinBox 0–32 (0 = off) | `--spec-draft-n-max` | 3 |
| Draft ngl | QLineEdit | `--spec-draft-ngl` | (empty) |
| Draft type K | QComboBox `f16`,`q8_0`,`q4_0`,`q4_1`,`q5_1` | `--spec-draft-type-k` | (empty = auto) |
| Draft type V | same | `--spec-draft-type-v` | (empty = auto) |

### Vision

| Field | Widget | Flag | Dummy default |
|---|---|---|---|
| mmproj | QLineEdit + `Browse…` button (button only logs `browse: dummy`) | `--mmproj` | (empty) |
| No mmproj offload | QCheckBox | `--no-mmproj-offload` | unchecked |

### Runtime

| Field | Widget | Flag | Dummy default (cmd 1) |
|---|---|---|---|
| Fit | QComboBox `off`,`on` | `--fit` | off |
| Perf | QCheckBox | `--perf` | checked |
| Sleep idle (s) | QSpinBox 0–86400 (0 = off) | `--sleep-idle-seconds` | 300 |
| Cache RAM (MB) | QSpinBox 0–1M (0 = auto) | `--cache-ram` | 0 |
| mlock | QCheckBox | `-mlock` | unchecked |
| no-mmap | QCheckBox | `-no-mmap` | unchecked |
| CPU threads | QSpinBox 0–256 (0 = auto) | `-t` | 0 |

- Panel exposes `SetConfig(cfg *dummy.Config)` / `Collect(cfg *dummy.Config)`
  on the same shared `Config` struct as task 04 (advanced panel touches only
  advanced fields). See task 04 for the pattern.
- Empty combo/lineedit entries mean "do not pass the flag" (documented in the
  command-line assembly of task 07).

## Acceptance

- All three sections render with defaults above.
- `SetValues` with the `vision+mtp` preset fills mmproj, no-mmproj-offload,
  spec-draft-ngl `all`, type-k/v `q8_0`, cache-ram 4096, sleep 5.
