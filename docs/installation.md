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
| `--with-deps` / `--no-deps` | install (or never touch) system packages |
| `--no-desktop` / `--no-config` | skip the desktop entry / user config |
| `--dry-run`, `-y` | preview / assume yes |
| `--uninstall [--purge]` | remove the app (`--purge` also removes data + config) |

It installs `llamaherd` to `~/.local/bin`, adds a desktop entry + `llamaherd-gui` launcher
(macOS: an `.app` bundle), and detects `llama-server` and a models folder for
`~/.config/llamaherd/config.json`. Re-running upgrades in place and keeps your data.

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
