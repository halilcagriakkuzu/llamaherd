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

### Launching the GUI (scenario B)

The installer writes a Linux `.desktop` entry, which only appears in a Linux desktop menu —
WSL has none, so nothing shows up on the Windows desktop by itself. On Windows 11 the
installer also creates **LlamaHerd** shortcuts on the Windows Desktop and in the Start menu
(they run `wslg.exe -d <distro> -- ~/.local/bin/llamaherd-gui`). Skip them with `--no-desktop`.

Ways to start it:

```bash
llamaherd-gui        # installed launcher: sets DISPLAY/GSK_RENDERER, opens the window
llamaherd            # same binary with no arguments; CLI subcommands still work
```

- Requires **WSLg** (Windows 11, or Windows 10 with the Microsoft Store WSL). Check with
  `ls /mnt/wslg` and `echo $DISPLAY` (usually `:0`); if `/mnt/wslg` is missing you need a
  Windows X server and `export DISPLAY=<host>:0`.
- No shortcut appeared? Run the installer again without `--no-desktop`, make sure
  `powershell.exe` is reachable from WSL (`powershell.exe -Command echo ok`), or create the
  shortcut manually: target `C:\Program Files\WSL\wslg.exe`, arguments
  `-d <distro> -- /home/<user>/.local/bin/llamaherd-gui`.
- Window does not open at all? Run `llamaherd-gui` in the terminal to see the error and
  `llamaherd doctor`. The installer installs the GTK/WebKit packages when they are missing
  (skip that with `--no-deps`); if `apt` was unavailable they are:
  `sudo apt-get install -y libgtk-3-0 libwebkit2gtk-4.1-0`.
- No icon on the shortcut? It uses the shipped `llamaherd.ico` copied to
  `%USERPROFILE%\LlamaHerd\`. Re-run the installer (or let Explorer refresh) if the cache
  still shows the old one.
- Uninstalling removes the Windows shortcuts again.

### Window frame (WSLg)

In WSL the window frame is drawn by WSLg's Windows-side compositor, not by the app. Since
0.4.0 LlamaHerd drops that frame by default and draws its own title bar in the app header
(minimise / maximise / close, drag anywhere on the header, double-click to maximise,
resize from all eight edges).

- `LLAMAHERD_FRAMELESS=0` brings the native frame back (also `"frameless": false` in the
  config file); `LLAMAHERD_FRAMELESS=1` forces the app-drawn title bar on other platforms.
- The window size and position are stored in `<data dir>/window.json`. A position that no
  longer matches a connected screen is discarded on the next launch. WSLg places windows
  itself and ignores move requests, so inside WSL only the size is restored.
- If dragging or edge-resizing does not work under your WSLg build, use the switch above;
  scenario **C** (Windows app + WSL server) also gives a fully native window.
- The Windows/desktop shortcut runs `scripts/launch-gui.sh`, which rebuilds
  `build/bin/llamaherd` when a source file is newer (or when called with `--rebuild`), so
  the shortcut always shows the current build. Startup logs the resolved frame:
  `window: frameless (app-drawn title bar)` or `window: native frame` (Config → Logs).

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
