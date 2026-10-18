# 01 — Main window shell

Status: DONE

## Goal

Window + layout skeleton so every later task has a fixed place to put its panel.

## Spec

- `QMainWindow`, title `LlamaHerd — llama.cpp control panel`, size 1100x720
  (`core.NewQSize2(1100, 720)`).
- Central widget, root vertical layout:
  1. Main horizontal layout:
     - Left column: fixed width **340**, vertical layout:
       - STATUS placeholder (fixed height ~230)
       - MODELS placeholder (stretch 1)
     - Right column: stretch 1, vertical layout:
       - ACTION ROW placeholder (fixed height ~56)
       - CONFIG TABS placeholder (stretch 1) — will hold QTabWidget
       - LOGS placeholder (fixed height ~160)
  2. QMainWindow status bar.
- Placeholders are `QGroupBox` with the final title (STATUS, MODELS, LOGS) and
  a `QLabel` inside saying `TODO <task#>`.
- Margins/spacing: 8px window margins, 8px between panels.

## Acceptance

- `CGO_ENABLED=1 go build -o llamaherd .` succeeds.
- `DISPLAY=:0 ./llamaherd` shows the exact layout above with empty placeholder
  groups in the right proportions (left fixed, right stretch).
