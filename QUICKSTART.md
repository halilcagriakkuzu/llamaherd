# Quickstart

Install, point LlamaHerd at llama.cpp, start a model.

**1. Install** (details: [docs/installation.md](docs/installation.md))

```bash
curl -fsSL https://raw.githubusercontent.com/halilcagriakkuzu/llamaherd/main/scripts/install.sh \
  | bash -s -- --from-release
```

Windows 11: `powershell -ExecutionPolicy Bypass -File scripts\install.ps1 -FromRelease`
(see [Windows and WSL](docs/windows-and-wsl.md) for the three setups).

**2. Point it at `llama-server` and your models**

In the GUI, the first-run dialog has a **Detect** button: it searches `PATH`, the usual
llama.cpp build directories (and WSL distros on Windows) and validates the binary with
`--version` before filling the field.

```bash
export LLAMAHERD_SERVER_BIN=~/llama.cpp/build/bin/llama-server
export LLAMAHERD_MODELS_DIR=~/LlamaHerd/models      # default; scanned recursively
llamaherd doctor                                      # checks everything, prints fixes
```

No models yet? `llamaherd download list` shows a few picks for your GPU/RAM (`--all` lists the
whole catalog, `download files <owner/name>` any Hugging Face repo):

```bash
llamaherd download Qwen/Qwen2.5-1.5B-Instruct-GGUF
```

**3. Run**

```bash
llamaherd                       # GUI  (WSLg: DISPLAY=:0 GSK_RENDERER=cairo llamaherd)
```

Pick a model, press **Start**, watch the status box. Headless:

```bash
llamaherd list && llamaherd open <alias> && llamaherd status
```

The server then listens on **`http://127.0.0.1:8090`** — LlamaHerd's default port (change it under
Basic → Port; `llama-server` itself would default to 8080). Start refused? That's the
[VRAM guard](docs/vram-guard.md); other issues: [troubleshooting](docs/troubleshooting.md).
