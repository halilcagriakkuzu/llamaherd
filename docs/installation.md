# Installation

Two supported paths: a **prebuilt binary** (no toolchain) or **building from source**.
The GUI uses the OS webview (WebKit2GTK on Linux, WebView2 on Windows 11, WebKit on
macOS), so nothing else ships with the app. You also need a `llama-server` build and at
least one GGUF model (see [quick start](../QUICKSTART.md)).

## Prebuilt binary

Release archives are attached to every [GitHub release](https://github.com/halilcagriakkuzu/llamaherd/releases)
with stable names: `llamaherd_{linux,darwin}_{amd64,arm64}.tar.gz` and
`llamaherd_windows_amd64.zip`.

```bash
# Linux / macOS / WSL
curl -fsSL https://raw.githubusercontent.com/halilcagriakkuzu/llamaherd/main/scripts/install.sh \
  | bash -s -- --from-release
```

```powershell
# Windows 11 — default mode runs the GUI inside WSL (WSLg); -Mode native installs llamaherd.exe
powershell -ExecutionPolicy Bypass -File scripts\install.ps1 -FromRelease
powershell -ExecutionPolicy Bypass -File scripts\install.ps1 -Mode native -FromRelease
```

Use `bash`, not `sh` (the installer uses bash features).

### `install.sh` options

| Option | Meaning |
| --- | --- |
| `--from-release` / `--from-source` | force the release download or a local source build |
| `--release-tag TAG`, `--repo OWNER/REPO` | which release / repository to use |
| `--prefix DIR`, `--bin-dir DIR`, `--data-dir DIR` | install locations (default prefix `~/.local`) |
| `--with-deps` / `--no-deps` | install missing GUI packages (the default) / never touch system packages |
| `--no-desktop` / `--no-config` | skip the desktop entry / user config (on WSL also the Windows shortcuts) |
| `--llama-server PATH` | use that `llama-server` build (no detection, no prompt) |
| `--install-llamacpp` | clone llama.cpp and build it (installs git, cmake, a C/C++ toolchain) |
| `--llama-dir DIR`, `--llama-build-dir DIR` | checkout / build dir (default `~/llama.cpp`, `build-latest`) |
| `--llama-backend auto\|cuda\|vulkan\|metal\|cpu` | GPU backend for that build (default: CUDA when an NVIDIA GPU and the CUDA toolkit are available, else CPU) |
| `--no-llamacpp` | never ask about llama.cpp; leave `serverBin` unset |
| `--dry-run`, `-y` | preview / assume yes (also skips the llama.cpp prompt) |
| `--uninstall [--purge]` | remove the app (`--purge` also removes data + config) |

It installs `llamaherd` to `~/.local/bin`, adds a desktop entry + `llamaherd-gui` launcher
(macOS: an `.app` bundle), and detects `llama-server` and a models folder for
`~/.config/llamaherd/config.json`. Re-running upgrades in place and keeps your data.

Missing GTK3 + WebKit2GTK (the GUI's runtime) are installed automatically through the distro
package manager — no question, `--no-deps` skips it. Inside WSL it also creates a **LlamaHerd**
shortcut on the Windows Desktop and Start menu, with the app icon.

### llama.cpp setup

llama.cpp is not bundled. When no `llama-server` is found and no config exists yet, the
installer asks (on the controlling terminal, so it also works with the `curl … | bash`
one-liner):

```
No llama-server found. How should the installer provide it?
  1) I already have a build — ask me for the path  [default]
  2) Clone llama.cpp and build it from source (installs git, cmake and a C/C++ toolchain)
  3) Skip for now — configure it on first launch
```

Option 2 clones `ggml-org/llama.cpp` into `~/llama.cpp` (or `--llama-dir`), installs the
missing build tools through your package manager (apt/dnf/pacman/zypper; on macOS the Xcode
Command Line Tools + Homebrew `cmake`), then runs the same `cmake` configure/build as
`scripts/update-llamacpp.sh`. The resulting path is written as `serverBin`, and the checkout
as `llamaDir` so the app's update button uses it. Non-interactive runs (CI, `-y`,
`--dry-run`, no terminal) never prompt: they report the missing binary and leave `serverBin`
unset; pass `--llama-server PATH` or `--install-llamacpp` to decide up front.

With `--llama-backend auto` the build uses CUDA when an NVIDIA GPU is present **and a CUDA
toolkit is usable**; a driver alone (`nvidia-smi`) is not enough. If `nvcc` is missing the
installer adds NVIDIA's CUDA repository (Ubuntu/Debian/WSL with apt) and installs
`cuda-toolkit`, falling back to the distro's `nvidia-cuda-toolkit`, `dnf`'s `cuda-toolkit`
(needs the repo enabled), `pacman`'s `cuda` or zypper's `cuda-toolkit`. If that still fails it
warns and builds CPU-only instead, so the install never dies on CUDA; `--llama-backend cuda`
makes the toolkit mandatory, and `--no-deps` disables the automatic install.

`install.ps1` switches: `-Mode wsl|native`, `-Distro`, `-FromRelease`, `-Tag`, `-Repo`,
`-WithDeps`, `-NoShortcuts`, `-Uninstall`, `-Purge`. Windows setups are explained in
[windows-and-wsl.md](windows-and-wsl.md).

## Build from source

Needs Go (version in `go.mod`), a C toolchain (cgo) and the platform GUI libraries.

```bash
# Linux / WSL (Ubuntu)
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev
go build -tags webkit2_41,desktop,production -o build/bin/llamaherd .
# distros that only ship WebKit2GTK 4.0: use the webkit2_40 tag

# macOS (xcode-select --install)  /  Windows 11 (WebView2 runtime + a C toolchain)
go build -tags desktop,production -o build/bin/llamaherd .
```

The Wails CLI (`wails build`) also works but is not required. The frontend is plain files
in `frontend/dist` — there is no npm step. On WSLg run the GUI with
`DISPLAY=:0 GSK_RENDERER=cairo ./build/bin/llamaherd`.

The CLI/daemon part builds anywhere without GUI libraries:
`CGO_ENABLED=0 go build ./...`.

### Version stamping

`llamaherd version` prints `main.version`, `dev` by default. Release builds stamp it:

```bash
go build -tags webkit2_41,desktop,production \
  -ldflags "-X main.version=$(git describe --tags --always)" -o build/bin/llamaherd .
```

## Releases (maintainers)

Pushing a `vX.Y.Z` tag runs `.github/workflows/release.yml`, which builds all five
archives and publishes the release. macOS and Windows jobs are best-effort and never
block the Linux assets. Update `CHANGELOG.md` first; regenerate the icon/Windows
resources with `scripts/make-icon.py` and `scripts/make-winres.sh` if they changed.
