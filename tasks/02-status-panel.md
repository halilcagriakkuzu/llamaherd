# 02 — Status panel

Status: DONE
Depends on: 01

## Goal

Show the (dummy) llama-server state at a glance.

## Spec

`QGroupBox "STATUS"` in the left column, `QFormLayout`.

- First row (outside form): a colored dot label `●` + `RUNNING` (bold).
  Dot green when running, red + `STOPPED` when stopped (toggled by task 07).
- Form rows (label : value):

| Label | Dummy value |
|---|---|
| PID | 749941 |
| Listen | 127.0.0.1:8080 |
| WSL IP | 172.24.96.2 |
| Model | Qwen3.8-27B-UD-IQ4_XS.gguf (base name only, elided) |
| Context | 25556 |
| GPU layers | 33/33 |
| VRAM | 16.8 GB |
| CPU RAM | 1.2 GB |
| Uptime | 01:23:45 |

- Values come from `dummy.Server` struct in `dummy/data.go`.
- Expose `SetRunning(bool)` so task 07 can flip state: running -> values above;
  stopped -> dot red, `STOPPED`, uptime `00:00:00`, PID `—`.

## Acceptance

- Panel renders all rows with dummy values, no overlap at 340px width.
- `SetRunning(false)` visibly switches to the stopped presentation.

## Amendment (task 07)

Task 07 widens this to `SetState(state string)` —
`starting | running | stopped | crashed` (amber/green/red/dark-red,
hallmark: `CRASHED exit N`). Update the dot/label logic accordingly when
implementing 07; `SetRunning(bool)` stays as a thin wrapper.
