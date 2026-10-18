# 04 — Config: BASIC tab

Status: TODO
Depends on: 01

## Goal

The everyday fields, one form, flags visible in the labels.

## Spec

`QTabWidget` in the right column, tab 1 titled `BASIC`. Inside: scroll area +
`QFormLayout`. Label format: `Name  <font-small>flag</font>`.

| Field | Widget | Flag | Dummy default (cmd 1) |
|---|---|---|---|
| Model | QLineEdit (read-only) | `--model` | path of model 1 |
| Alias | QLineEdit | `--alias` | qwen3.8-27b-local |
| Host | QLineEdit | `--host` | 127.0.0.1 |
| Port | QSpinBox 1–65535 | `--port` | 8080 |
| Context | QSpinBox 1–1M | `-c` | 25556 |
| GPU layers | QLineEdit | `-ngl` | all |
| Parallel slots | QSpinBox 1–64 | `-np` | 1 |
| Flash attention | QComboBox `off`,`on` | `-fa` | on |
| K cache type | QComboBox `f16`,`q8_0`,`q4_0`,`q4_1`,`q5_1` | `-ctk` | q8_0 |
| V cache type | same combo values | `-ctv` | q8_0 |
| Jinja | QCheckBox | `--jinja` | checked |
| Reasoning format | QComboBox `none`,`deepseek` | `--reasoning-format` | deepseek |

- Panel exposes `SetConfig(cfg *dummy.Config)` / `Collect(cfg *dummy.Config)`
  operating on the ONE shared `Config` struct (`dummy/data.go`), not a
  per-form map — presets (task 06) and command-line assembly (task 07) read
  the same struct. `SetConfig` copies all fields (basic panel only touches
  basic ones; unknown fields untouched so both panels can share one cfg).
- Defaults come from the `qwen3.8-27b-local / default` preset.

## Acceptance

- All fields present, defaults as above.
- `SetValues` with the `vision+mtp` preset values changes the form correctly
  (ctx 32768 etc.), without touching ADVANCED-only fields.
