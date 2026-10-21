# Troubleshooting

Start with `llamaherd doctor` — it checks the binary, GPU, GUI libraries, display, models
folder, disk space and API port, and prints a fix for each failure.

| Symptom | Fix |
| --- | --- |
| **`llama-server binary` fails / Start does nothing** | `LLAMAHERD_SERVER_BIN=/path/to/llama-server llamaherd doctor`, or set `serverBin`. On Windows with a WSL build use `wsl:<distro>:/path`. |
| **Blank window / WebKit or GTK error (Linux, WSL)** | `sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev`; on WSLg launch with `DISPLAY=:0 GSK_RENDERER=cairo`. |
| **"something is already listening on 127.0.0.1:8090"** | Another `llama-server` owns the port. Use **Force stop** in the llama.cpp tab or change `basic.port`. |
| **Start refused by the VRAM guard** | Lower context, use a `q8_0`/`q4_0` KV cache or fewer GPU layers — see [vram-guard.md](vram-guard.md). |
| **Download fails / verification fails** | Failed downloads keep a `.part` file and resume; a sha256 mismatch deletes it. Private/gated repos need `llamaherd download token <token>`. `llamaherd download status` shows the error. |
| **Can't reach the server from Windows** | The WSL IP row is only filled inside WSL and the server binds `127.0.0.1`; see [windows-and-wsl.md](windows-and-wsl.md). |
| **Windows shortcut shows a blank icon** | Re-run `scripts/make-windows-shortcut.ps1`; if Explorer still caches the old icon, restart Explorer once. |
| **The download list looks outdated** | The app refreshes it from GitHub (`LLAMAHERD_CATALOG_URL`); offline it uses the cached or embedded copy. |
| **Rename refused: "stop the running server"** | A model's alias can't change while its server runs — stop it first. |
| **AMD / Intel / Apple GPU shows "—" for VRAM** | LlamaHerd probes `nvidia-smi`, then `rocm-smi` (AMD), `xpu-smi` (Intel) and, on macOS, unified memory via `sysctl`. Install the vendor tool (or run `llamaherd doctor` to see which provider is active). Unknown memory means no estimate — the VRAM guard stands down rather than blocking. The AMD/Intel parsers are best-effort and have not been validated on real hardware. |
| **No host notifications** | Linux needs a notification daemon and `notify-send`; macOS uses `osascript`; Windows uses the toast API. Check the per-class toggles in the llama.cpp tab or `llamaherd notifications`, and `llamaherd logs` for a `notify:` line. Events always remain in the in-app bell. |
