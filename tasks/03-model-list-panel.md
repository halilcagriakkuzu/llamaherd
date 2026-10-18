# 03 — Model list panel

Status: TODO
Depends on: 01

## Goal

List available models; selecting one drives the whole config side.

## Spec

`QGroupBox "MODELS"` in the left column (stretch), containing a `QListWidget`.

- Items (short display name, full path in tooltip), in `dummy/data.go`:

| Display | Path |
|---|---|
| Qwen3.8-27B-UD-IQ4_XS | /home/halilcagriakkuzu/llm/models/unsloth/Qwen3.8-27B-GGUF/Qwen3.8-27B-UD-IQ4_XS.gguf |
| Qwen3.8-27B-TurboFCFusion-...-IQ4_XS | /home/halilcagriakkuzu/llm/models/Qwen3.8_Uncensored/Heretic/Fable-Distilled/Qwen3.8-27B-TurboFCFusion-735-882-Here-Uncen-NEO-CODER-MAX-LOW-MTP-IQ4_XS.gguf |
| Qwen3.8-27B-Uncensored-HauhauCS-Q3_K_P | /home/halilcagriakkuzu/llm/models/Qwen3.8-27B-Uncensored-HauhauCS-Aggressive-Q3_K_P.gguf |
| mmproj-F16.gguf | /home/halilcagriakkuzu/llm/models/Qwen3.8_Uncensored/Heretic/Fable-Distilled/mmproj-F16.gguf |

- Item 1 selected by default.
- On selection of a model (items 1–3):
  1. Fill BASIC form: model path (read-only), alias, and the alias's first preset values (task 06).
  2. Fill preset combo with that alias's presets (task 06).
  3. Update config path label to `configs/<alias>.json` (task 06).
  4. Append log line `selected: <display name>`.
- On selection of the mmproj item (4): fill the `--mmproj` advanced field with
  its path, append log line `mmproj: mmproj-F16.gguf`. Model selection is NOT
  changed.

## Acceptance

- Selecting each of the 4 items produces the effects above; log console
  receives the lines.
- Default state on launch: item 1 selected, BASIC form populated with
  qwen3.8-27b-local preset.
