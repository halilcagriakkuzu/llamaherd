# 07 — Actions: start / stop

Status: TODO
Depends on: 02, 06

## Goal

The action row and the two lifecycle buttons (dummy transitions).

## Spec

### Action row (fixed height ~56, top of right column)

Horizontal layout: preset combo | `Start` | `Stop` | `Save cfg` | `Load cfg` |
stretch | config path label.

- `Start`: green-ish. Disabled while running.
- `Stop`: red-ish. Disabled while stopped.
- Initial state: server dummy-RUNNING (task 02), so `Stop` enabled, `Start` disabled.

### Dummy transitions

Task 02's status panel gets `SetState(state string)` accepting
`starting | running | stopped | crashed`, mapping to dot + label:
starting=amber pulsing, running=green, stopped=red, crashed=dark-red +
`CRASHED exit N`. This previews the real state machine spec'd in the README
phase-2 section.

- Start clicked:
  1. Status → `starting`; Log `starting llama-server: <assembled command line>`; both buttons disabled
  2. After 2s (tick timer): Log `ready in 2.0s (dummy)`; status → `running` (uptime 00:00:00); `Stop` enabled
- Stop clicked:
  1. Log `stopping llama-server (pid 749941)`, both buttons disabled
  2. After ~1s: Log `stopped (dummy)`; status → `stopped`; `Start` enabled
- (Design-review hook: a hidden test hook or intentional flat in review to
  force `crashed exit 1` — verifies the CRASHED presentation works.)

### Command line assembly (string only)

Assembled from the ONE shared `*dummy.Config` (both forms Collect into it:
task 04 then task 05) in llama-server flag order:
`--model --alias --host --port -ngl --fit -c -np -fa -ctk -ctv --jinja
--reasoning-format --spec-type --spec-draft-n-max --spec-draft-ngl
--spec-draft-type-k --spec-draft-type-v --mmproj --no-mmproj-offload --perf
--sleep-idle-seconds --cache-ram -mlock -no-mmap -t`.
Rules: skip empty values; skip `--spec-*` when spec type is `none`;
`--jinja`/`--perf`/`-mlock`/`-no-mmap`/`--no-mmproj-offload` only when checked;
numeric-zero fields (sleep, cache-ram, threads) skipped when 0.
The result is ONLY used in the log line — never executed.

## Acceptance

- Full start->stop->start cycle works; status dot, uptime, and button
  enabled-state all consistent.
- Log shows a full, plausible command line matching the current form values
  (with `vision+mtp` selected it must contain `--mmproj`,
  `--spec-draft-ngl all`, `--cache-ram 4096`).
