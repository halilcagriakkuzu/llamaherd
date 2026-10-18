# Windows, WSL and other platforms

| Platform | GUI | CLI / daemon | Notes |
| --- | --- | --- | --- |
| Linux (incl. WSLg) | yes | yes | primary target; GTK3 + WebKit2GTK |
| Windows 11 | yes | yes | WebView2; or run the GUI inside WSL |
| macOS | yes | yes | builds and cross-compiles; not yet run on real hardware |

## Windows setups

| Scenario | LlamaHerd runs in | `llama-server` | Models | Setup |
| --- | --- | --- | --- | --- |
| **A. No WSL** | Windows | `llama-server.exe` | Windows folder (`%USERPROFILE%\LlamaHerd\models`) | `install.ps1 -Mode native`, point `serverBin` at the `.exe` |
| **B. All inside WSL** | WSL (GUI on WSLg) | Linux build in WSL | WSL folder | `install.ps1` (default) — fastest model loads |
| **C. Windows app + WSL server** | Windows | Linux build in WSL | Windows or WSL folder | `serverBin` = `wsl:<distro>:/path/to/llama-server` |

### Scenario C details

- `LLAMAHERD_SERVER_BIN=wsl:Ubuntu:/home/me/llama.cpp/build/bin/llama-server`
  (`wsl:/path` uses the default distro). The server is launched through `wsl.exe`;
  Windows paths in its arguments are translated (`C:\x` → `/mnt/c/x`,
  `\\wsl.localhost\Ubuntu\home\me\m` → `/home/me/m`); Stop terminates the Linux process.
- **Models on a Windows drive load slowly** (WSL reads them over `/mnt/c`). For fast loads
  keep models inside WSL and set the models directory to the UNC path
  `\\wsl.localhost\Ubuntu\home\me\models`. `llamaherd doctor` warns about the slow case.
- `llama-server` must listen on `127.0.0.1`; WSL2 localhost forwarding exposes it to Windows.

## Known limitations

- **WSL IP** is shown only inside WSL; `llama-server` binds `127.0.0.1` by default, so
  other machines need mirrored networking or a non-loopback `--host`.
- **Process discovery:** `/proc` (Linux), `pgrep`/`ps` (macOS), `tasklist` (Windows; model /
  listen columns are only filled on Linux/macOS).
- **Stop:** `SIGTERM` then `SIGKILL` on Linux/macOS; Windows has no SIGTERM, so it is a hard
  kill (`TerminateProcess`).
- **llama.cpp updates and GPU metrics** use bash scripts and `nvidia-smi`: the updater is
  Unix-only; inference works everywhere and a CUDA GPU is optional.
- Open verification work is tracked in [`BACKLOG.md`](../BACKLOG.md).
