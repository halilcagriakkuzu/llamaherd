# LlamaHerd — Quickstart

The impatient path. Full details are in [`README.md`](README.md).

## 1. Get the app

**Linux / WSL (prebuilt):**

```bash
curl -fsSL https://raw.githubusercontent.com/<you>/LlamaHerd/main/scripts/install.sh \
  | bash -s -- --from-release --repo <you>/LlamaHerd
```

**Linux / WSL (from a checkout):**

```bash
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev
scripts/install.sh
```

**Windows 11:** `powershell -ExecutionPolicy Bypass -File scripts\install.ps1`
(default runs the GUI inside WSL and creates shortcuts).
**macOS:** `git clone … && scripts/install.sh` (needs `xcode-select --install`).

## 2. Point it at llama.cpp and your models

```bash
export LLAMAHERD_SERVER_BIN=~/llama.cpp/build/bin/llama-server
export LLAMAHERD_MODELS_DIR=~/llm/models
llamaherd doctor          # verifies binary, GPU, libs, models dir, port
```

`doctor` prints a fix for anything that fails. You can also set these in
`~/.config/llamaherd/config.json` (see the README).

## 3. Launch

```bash
DISPLAY=:0 GSK_RENDERER=cairo llamaherd   # GUI (WSLg)
```

No display? Use the headless API + CLI instead:

```bash
llamaherd daemon &        # serves the loopback control API on 127.0.0.1:8517
llamaherd list
llamaherd open <alias>
llamaherd status
llamaherd stop
```

## 4. No model yet? Download one

In the GUI: **Models → ＋ Download a model**. From the terminal:

```bash
llamaherd download list                          # curated, scored for this machine
llamaherd download Qwen/Qwen2.5-1.5B-Instruct-GGUF
```

Downloads resume automatically and are verified (size + sha256) before a
~50k-context preset is written.

## 5. Start and use

Pick a model + preset, press **Start**, then watch the **Status** panel and
**Logs**. Point your client (or `llamaherd status`) at `http://127.0.0.1:8090`.

Hit the VRAM guard? Lower `basic.context`, set the KV cache to `q8_0`/`q4_0`,
or reduce `gpuLayers`. See the README's Troubleshooting section.
