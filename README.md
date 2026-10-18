# LlamaHerd

A desktop control panel for [llama.cpp](https://github.com/ggml-org/llama.cpp)'s
`llama-server`: pick a GGUF model, tune it in a form, press **Start**. One Go binary gives
you the GUI, a CLI and a local control API for scripts and AI agents.

<p align="center">
  <img src="docs/screenshots/main-light.png" alt="LlamaHerd, light theme" width="49%">
  <img src="docs/screenshots/main-dark.png" alt="LlamaHerd, dark theme" width="49%">
</p>

> **Status: early.** Linux (incl. WSL) and Windows 11 are exercised; macOS builds but has
> not been run yet. Expect rough edges. Open work: [`BACKLOG.md`](BACKLOG.md).

## Features

- Model browser with nested categories (rename, move, create folders from the UI/CLI/API).
- Download from Hugging Face: hardware-scored catalog, resumable, sha256-verified; vision
  and MTP sidecars detected.
- Config form for the common `llama-server` flags, per-model presets as plain JSON.
- Start/stop, crash detection, live VRAM/RAM, average tokens/s.
- **VRAM pre-flight guard**: refuses a start predicted to exceed 90 % of GPU memory.
- One-click llama.cpp update/rebuild, environment check (`llamaherd doctor`).
- Light/dark UI; Linux, Windows 11 (native or WSL), macOS.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/halilcagriakkuzu/llamaherd/main/scripts/install.sh \
  | bash -s -- --from-release
```

Windows, building from source and all options: [docs/installation.md](docs/installation.md).
You also need a `llama-server` build and a GGUF model — see the [quickstart](QUICKSTART.md).

## Use

```bash
llamaherd                      # GUI
llamaherd doctor               # check binary, GPU, libs, models dir, port
llamaherd list                 # models and presets
llamaherd open <alias>         # start llama-server
llamaherd status | stop
llamaherd download list        # models that fit this machine
```

The same actions are available over HTTP on `127.0.0.1:8517`:

```bash
curl -s 127.0.0.1:8517/status | jq '.status, .server.tokPerSec'
```

## Documentation

| | |
| --- | --- |
| [Quickstart](QUICKSTART.md) | install → first model in five minutes |
| [Installation](docs/installation.md) | installer options, build from source, releases |
| [Configuration](docs/configuration.md) | config file, env vars, presets, library, HF token |
| [CLI and API](docs/cli-and-api.md) | every subcommand and endpoint |
| [Windows and WSL](docs/windows-and-wsl.md) | three Windows setups, platform limits |
| [VRAM guard](docs/vram-guard.md) | how the estimate works |
| [Managing llama.cpp](docs/llama-cpp.md) | locating, updating, force-stop |
| [Troubleshooting](docs/troubleshooting.md) | common problems |
| [Architecture](docs/architecture.md) | design, packages, Wails contract |
| [AGENTS.md](AGENTS.md) | contributor / coding-agent guide (build, test, conventions) |
| [Changelog](CHANGELOG.md) · [Backlog](BACKLOG.md) | what shipped · what's open |

## Contributing

Issues and pull requests are welcome. Start with [`AGENTS.md`](AGENTS.md): it lists the
build, test (`go test ./...` needs no GPU) and lint commands and the project conventions.

## License

[The Unlicense](LICENSE) — public domain.
Built on [llama.cpp](https://github.com/ggml-org/llama.cpp) and [Wails](https://wails.io/);
not affiliated with the llama.cpp project.
