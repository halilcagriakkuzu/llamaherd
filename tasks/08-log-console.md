# 08 — Log console

Status: TODO
Depends on: 01

## Goal

One place to see what the app is doing (in the dummy phase: every interaction).

## Spec

`QGroupBox "LOGS"` at the bottom of the right column (fixed height ~160).
Read-only `QTextEdit`, monospace font (`QFontDatabase::systemFont` Fixed or
`QFont("Monospace", 9)`), no cursor interaction needed.

- API: `NewLogConsole(parent)` returning the group + `Append(line string)`
  method (or a small struct) used by all other panels.
- Lines are plain, oldest scrolled out of view automatically (Qt default).
- Pre-populated on launch with dummy boot lines for command 1's config:

```
LlamaHerd (dummy mode) — no real server control
server: running (pid 749941)
listen: 127.0.0.1:8080
model:  Qwen3.8-27B-UD-IQ4_XS.gguf
preset: qwen3.8-27b-local/default
```

- All other panels append here on every interaction (selection, preset change,
  save, load, start, stop, browse).

## Acceptance

- Launch shows the 5 boot lines.
- Every interaction in tasks 03/06/07 appends a line; scrolling works when
  many lines accumulate (spam start/stop a few times).
