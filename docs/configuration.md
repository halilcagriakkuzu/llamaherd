# Configuration

Every setting resolves in this order: **environment variable → user config file →
auto-discovery → built-in default**.

## User config file

Written by the first-run dialog; editable by hand.

| OS | Path |
| --- | --- |
| Linux/BSD | `$XDG_CONFIG_HOME/llamaherd/config.json` or `~/.config/llamaherd/config.json` |
| Windows | `%APPDATA%\llamaherd\config.json` |
| macOS | `~/Library/Application Support/llamaherd/config.json` |

```json
{
  "serverBin": "/opt/llama.cpp/build/bin/llama-server",
  "llamaDir": "/opt/llama.cpp",
  "llamaBuildDir": "build",
  "modelsDir": "/data/models",
  "apiPort": "8517",
  "apiToken": ""
}
```

The file is written with `0600` permissions (it can hold the API token). On Windows,
`serverBin` may be `wsl:<distro>:/path/to/llama-server`
([windows-and-wsl.md](windows-and-wsl.md)).

## Environment variables

| Variable | Purpose |
| --- | --- |
| `LLAMAHERD_SERVER_BIN` | full path to `llama-server` |
| `LLAMAHERD_LLAMA_DIR`, `LLAMAHERD_LLAMA_BUILD_DIR` | llama.cpp checkout and its build subdirectory |
| `LLAMAHERD_MODELS_DIR` | scanned recursively for `*.gguf` (default `~/LlamaHerd/models`, created on first run) |
| `LLAMAHERD_API_PORT`, `LLAMAHERD_API_TOKEN` | loopback control API port (default `8517`) and optional bearer token |
| `LLAMAHERD_CONFIG` | config-file path |
| `LLAMAHERD_DATA_DIR`, `LLAMAHERD_CONFIGS_DIR`, `LLAMAHERD_LIBRARY_DIR` | where `configs/` and `library/` live |
| `LLAMAHERD_MIN_LLAMA_BUILD` | minimum llama.cpp build that `doctor` accepts |
| `LLAMAHERD_CATALOG_URL` | model-catalog source (default: `hf/catalog.json` on the repo's `main` branch; `off` disables the network refresh) |
| `HF_TOKEN`, `HUGGING_FACE_HUB_TOKEN`, `HF_ENDPOINT` | Hugging Face token / mirror |

`configs/` and `library/` default to a per-user data dir (`~/.local/share/llamaherd`,
`%LOCALAPPDATA%\llamaherd`, `~/Library/Application Support/llamaherd`). A `./configs` or
`./library` next to the working directory (a source checkout) wins, so development and
tests use the repo's files.

## Presets: `configs/<alias>.json`

Each model has an alias (the file name, lower-cased, dots → dashes) and one preset file
holding named presets (full Basic + Advanced field sets). A model without a file gets a
synthesized `default` preset that fits your GPU. Model/mmproj/draft paths inside are stored
relative to the models directory, so files stay portable.

## Library: `library/<id>.json`

Per-model metadata, created automatically:

| Field | Meaning |
| --- | --- |
| `category` | path in the model tree (e.g. `["Qwen","3.8"]`); empty = Uncategorized |
| `name`, `alias` | optional display-name / alias overrides (set from the UI, CLI or API) |
| `mmproj`, `mtp`, `mtpDraft` | vision sidecar, embedded MTP head, separate MTP draft |
| `source` | where the model came from (e.g. a Hugging Face repo) |

`library/_categories.json` holds category folders that exist without models. Renaming,
moving and categorising never touches the `.gguf` files. The `id` is always derived from
the file name, so renaming the alias keeps the same library entry.

## Hugging Face token

For private or gated repos, in order: `HF_TOKEN` / `HUGGING_FACE_HUB_TOKEN`, a `0600`
`hf_token` file next to `config.json` (set it with `llamaherd download token <token>` or the
download dialog), then `~/.cache/huggingface/token`. It is never written into configs.

## Model catalog

The download dialog's recommendations come from `hf/catalog.json`. A copy is embedded in the
binary; on start and when the dialog opens the app fetches the file from the repository's
`main` branch (at most every 10 minutes), validates it and caches it as `catalog.json` in the
data dir, so the model list can be updated **without a new release**. Offline, or when the
fetch fails, the cached or embedded copy is used. An older or invalid file never replaces the
active catalog.

Which entries are shown depends on your hardware: a model is hidden unless it fits the GPU,
fits partly (at least half of its weights in VRAM, the rest in RAM) or is small enough for
CPU-only use. The dialog highlights up to five (best quality, vision, longest context, MTP,
lightweight); **Browse all models…** lists the whole catalog. Maintainers: see
[`AGENTS.md`](../AGENTS.md#updating-the-model-catalog).
