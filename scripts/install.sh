#!/usr/bin/env bash
# LlamaHerd installer — Linux, macOS, and WSL (terminal only).
#
# Installs the llamaherd binary either by building a local source checkout with
# Go, or by downloading a prebuilt asset from GitHub Releases. Optionally
# installs system dependencies, seeds the shipped presets, writes a desktop
# entry / .app bundle, detects llama-server + models and writes the user config,
# and records an install manifest for a clean uninstall.
#
#   scripts/install.sh                 # build from this checkout (if any)
#   scripts/install.sh --from-release  # download a prebuilt binary
#   scripts/install.sh --uninstall     # remove a previous install
#   scripts/install.sh --help
set -euo pipefail

APP="llamaherd"
# Works both as a file (`./install.sh`) and piped (`curl … | bash -s -- …`).
# When piped there is no script directory / checkout, so source-building is
# unavailable and the installer falls back to the release asset.
_self="${BASH_SOURCE[0]:-}"
if [ -n "$_self" ] && [ "$_self" != "-" ]; then
  PROG="$(basename "$_self")"
  SCRIPT_DIR="$(cd "$(dirname "$_self")" && pwd)"
  REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
else
  PROG="install.sh"
  SCRIPT_DIR=""
  REPO_ROOT="$(pwd)"
fi

# ---------------------------------------------------------------------------
# defaults
# ---------------------------------------------------------------------------
MODE=""                         # source | release (auto-detected)
SOURCE_DIR="${REPO_ROOT}"
SOURCE_URL=""
RELEASE_TAG="latest"
REPO_SLUG="${LLAMAHERD_REPO:-}"
PREFIX="${PREFIX:-$HOME/.local}"
BIN_DIR=""
DATA_DIR="${LLAMAHERD_DATA_DIR:-}"
CONFIG_DIR="${LLAMAHERD_CONFIG_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/llamaherd}"
DEPS="ask"                      # ask | yes | no
WITH_DESKTOP=1
WITH_CONFIG=1
DRY_RUN=0
ASSUME_YES=0
UNINSTALL=0
PURGE=0
ICON_SRC=""

# Set this to your "OWNER/REPO" so the piped one-liner needs no --repo flag:
#   curl -fsSL .../install.sh | bash -s -- --from-release
# (Left empty in the source repo; --repo / LLAMAHERD_REPO still override it.)
BUILTIN_REPO=""

# ---------------------------------------------------------------------------
# output helpers
# ---------------------------------------------------------------------------
if [ -t 1 ]; then
  C_RESET=$'\033[0m'; C_DIM=$'\033[2m'; C_BOLD=$'\033[1m'
  C_RED=$'\033[31m'; C_GREEN=$'\033[32m'; C_YELLOW=$'\033[33m'; C_BLUE=$'\033[34m'
else
  C_RESET=""; C_DIM=""; C_BOLD=""; C_RED=""; C_GREEN=""; C_YELLOW=""; C_BLUE=""
fi

log()  { printf '%s==>%s %s\n' "$C_BLUE$C_BOLD" "$C_RESET" "$*"; }
info() { printf '    %s\n' "$*"; }
warn() { printf '%swarn:%s %s\n' "$C_YELLOW" "$C_RESET" "$*" >&2; }
err()  { printf '%serror:%s %s\n' "$C_RED" "$C_RESET" "$*" >&2; }
die()  { err "$*"; exit 1; }

have() { command -v "$1" >/dev/null 2>&1; }

confirm() {
  [ "$ASSUME_YES" = 1 ] && return 0
  [ -t 0 ] || return 1
  local reply=""
  read -r -p "$1 [y/N] " reply || true
  case "$reply" in [yY] | [yY][eE][sS]) return 0 ;; *) return 1 ;; esac
}

# run CMD... honors --dry-run for simple (non-redirect) commands.
run() {
  if [ "$DRY_RUN" = 1 ]; then
    printf '  %s[dry-run]%s %s\n' "$C_DIM" "$C_RESET" "$*"
    return 0
  fi
  "$@"
}

usage() {
  cat <<EOF
${C_BOLD}LlamaHerd installer${C_RESET} (Linux / macOS / WSL)

Usage: $PROG [options]

Source selection:
  --from-source          build from a local checkout (default when run from one)
  --from-release         download a prebuilt binary from GitHub Releases
  --source-dir DIR       checkout to build from (default: this repo)
  --source-url URL       git URL to clone and build from
  --from-git URL         alias for --source-url
  --release-tag TAG      release tag to fetch (default: latest)
  --repo OWNER/REPO      GitHub repo used for releases (default: \$LLAMAHERD_REPO,
                         else the local git remote)

Install layout:
  --prefix DIR           install prefix (default: \$PREFIX or ~/.local)
  --bin-dir DIR          binary directory (default: PREFIX/bin)
  --data-dir DIR         configs/ + library/ data dir (default: XDG data dir)

Behavior:
  --with-deps            install system dependencies without asking
  --no-deps              never touch system packages
  --no-desktop           skip the .desktop entry / .app bundle + launcher
  --no-config            do not write the user config file
  --icon FILE            PNG icon to install (default: bundled assets/)
  -y, --yes              assume yes (non-interactive)
  --dry-run              print what would happen; change nothing

Uninstall:
  --uninstall            remove a previous installation (uses the manifest)
  --purge                with --uninstall, also remove data + config

Other:
  -h, --help             show this help
EOF
}

# ---------------------------------------------------------------------------
# argument parsing
# ---------------------------------------------------------------------------
need_arg() { [ "$1" -ge 2 ] || die "option $2 requires an argument"; }

while [ $# -gt 0 ]; do
  case "$1" in
    --from-source)  MODE="source" ;;
    --from-release) MODE="release" ;;
    --from-git | --source-url)
      need_arg $# "$1"; SOURCE_URL="$2"; MODE="source"; shift ;;
    --source-dir)
      need_arg $# "$1"; SOURCE_DIR="$2"; MODE="source"; shift ;;
    --release-tag)
      need_arg $# "$1"; RELEASE_TAG="$2"; shift ;;
    --repo)
      need_arg $# "$1"; REPO_SLUG="$2"; shift ;;
    --prefix)
      need_arg $# "$1"; PREFIX="$2"; shift ;;
    --bin-dir)
      need_arg $# "$1"; BIN_DIR="$2"; shift ;;
    --data-dir)
      need_arg $# "$1"; DATA_DIR="$2"; shift ;;
    --icon)
      need_arg $# "$1"; ICON_SRC="$2"; shift ;;
    --with-deps) DEPS="yes" ;;
    --no-deps)   DEPS="no" ;;
    --no-desktop) WITH_DESKTOP=0 ;;
    --no-config)  WITH_CONFIG=0 ;;
    -y | --yes)   ASSUME_YES=1 ;;
    --dry-run)    DRY_RUN=1 ;;
    --uninstall)  UNINSTALL=1 ;;
    --purge)      PURGE=1 ;;
    -h | --help)  usage; exit 0 ;;
    *) die "unknown option: $1 (try --help)" ;;
  esac
  shift
done

[ -n "$BIN_DIR" ] || BIN_DIR="$PREFIX/bin"
[ -n "$DATA_DIR" ] || DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/$APP"
MANIFEST="$PREFIX/share/$APP/install.manifest"

# ---------------------------------------------------------------------------
# platform detection
# ---------------------------------------------------------------------------
OS=""
ARCH=""
case "$(uname -s)" in
  Linux)  OS="linux" ;;
  Darwin) OS="darwin" ;;
  *) die "unsupported operating system: $(uname -s) (use install.ps1 on Windows)" ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  armv7l) ARCH="armv7" ;;
  *) die "unsupported architecture: $(uname -m)" ;;
esac

IS_WSL=0
if [ "$OS" = "linux" ] && grep -qi microsoft /proc/version 2>/dev/null; then
  IS_WSL=1
fi

# GUI launch env differs on WSLg.
if [ "$IS_WSL" = 1 ]; then
  PLATFORM="WSL (Linux)"
else
  PLATFORM="$OS/$ARCH"
fi

# ---------------------------------------------------------------------------
# install manifest helpers
# ---------------------------------------------------------------------------
manifest_add() {
  [ "$DRY_RUN" = 1 ] && return 0
  mkdir -p "$(dirname "$MANIFEST")" 2>/dev/null || true
  printf '%s\n' "$1" >>"$MANIFEST.tmp" 2>/dev/null || true
}

read_manifest() {
  [ -f "$MANIFEST" ] || return 1
  grep -v '^[[:space:]]*$' "$MANIFEST"
}

# ---------------------------------------------------------------------------
# uninstall
# ---------------------------------------------------------------------------
do_uninstall() {
  log "Uninstalling LlamaHerd from $PREFIX"
  if read_manifest >/dev/null; then
    while IFS= read -r path; do
      [ -n "$path" ] || continue
      if [ -e "$path" ] || [ -L "$path" ]; then
        run rm -f "$path"
        info "removed $path"
      fi
    done < <(read_manifest)
  else
    warn "no manifest at $MANIFEST — removing known paths instead"
    for p in \
      "$BIN_DIR/$APP" \
      "$BIN_DIR/$APP-gui" \
      "$PREFIX/share/applications/$APP.desktop" \
      "$PREFIX/share/icons/hicolor/256x256/apps/$APP.png"; do
      run rm -f "$p"
    done
  fi

  # macOS app bundle
  if [ "$OS" = "darwin" ]; then
    run rm -rf "$HOME/Applications/LlamaHerd.app"
  fi

  run rm -f "$MANIFEST"
  # Tidy now-empty manifest dir.
  run rmdir "$PREFIX/share/$APP" 2>/dev/null || true

  if [ "$PURGE" = 1 ]; then
    log "Purging user data and configuration (--purge)"
    run rm -rf "$DATA_DIR" "$CONFIG_DIR"
  else
    info "kept user data ($DATA_DIR) and config ($CONFIG_DIR); use --purge to remove"
  fi
  log "Uninstall complete."
}

# ---------------------------------------------------------------------------
# system dependencies
# ---------------------------------------------------------------------------
detect_webkit_pkgs() {
  # Sets WEBKIT_DEV / WEBKIT_RUNTIME for Debian/Ubuntu-style systems.
  if have apt-cache && apt-cache show libwebkit2gtk-4.1-dev >/dev/null 2>&1; then
    WEBKIT_DEV="libwebkit2gtk-4.1-dev"
    WEBKIT_RUNTIME="libwebkit2gtk-4.1-0"
  else
    WEBKIT_DEV="libwebkit2gtk-4.0-dev"
    WEBKIT_RUNTIME="libwebkit2gtk-4.0-37"
  fi
}

install_deps() {
  [ "$DEPS" = "no" ] && { info "skipping system dependencies (--no-deps)"; return 0; }

  if [ "$OS" = "darwin" ]; then
    if ! xcode-select -p >/dev/null 2>&1; then
      warn "Xcode Command Line Tools are required to build."
      info "Run: xcode-select --install"
      [ "$MODE" = "release" ] || die "cannot build without the Command Line Tools"
    fi
    return 0
  fi

  # Linux: figure out the package set.
  local packages=""
  if have apt-get; then
    detect_webkit_pkgs
    if [ "$MODE" = "source" ]; then
      packages="gcc g++ make pkg-config libgtk-3-dev $WEBKIT_DEV"
    else
      packages="libgtk-3-0 $WEBKIT_RUNTIME"
    fi
  elif have dnf; then
    packages="gtk3-devel webkit2gtk4.1-devel"
  elif have pacman; then
    packages="gtk3 webkit2gtk-4.1"
  elif have zypper; then
    packages="gtk3-devel webkit2gtk3-devel"
  else
    warn "unknown package manager — install GTK3 + WebKit2GTK manually"
    return 0
  fi

  log "System dependencies needed: $packages"
  if [ "$DEPS" = "ask" ] && ! confirm "Install these system packages now?"; then
    warn "skipped; install them manually if the GUI fails to load"
    return 0
  fi

  local SUDO=""
  [ "$(id -u)" -ne 0 ] && SUDO="sudo"
  if [ -n "$SUDO" ] && ! have sudo; then
    warn "sudo not found; run as root or install dependencies manually"
    return 0
  fi

  if have apt-get; then
    run $SUDO apt-get update
    # shellcheck disable=SC2086
    run $SUDO apt-get install -y $packages
  elif have dnf; then
    run $SUDO dnf install -y $packages
  elif have pacman; then
    run $SUDO pacman -Sy --noconfirm $packages
  elif have zypper; then
    run $SUDO zypper --non-interactive install $packages
  fi
}

# ---------------------------------------------------------------------------
# source build
# ---------------------------------------------------------------------------
check_go() {
  have go || die "Go toolchain not found. Install Go (see go.mod for the required
version), then re-run.  https://go.dev/dl/"
  local want got
  want="$(sed -n 's/^go[[:space:]]\+\([0-9][0-9.]*\).*/\1/p' "$SOURCE_DIR/go.mod" | head -1)"
  got="$(go env GOVERSION 2>/dev/null | sed 's/^go//')"
  if [ -n "$want" ] && [ -n "$got" ]; then
    # Compare major.minor only; tolerate a newer toolchain.
    local wmaj wmin gmaj gmin
    wmaj="${want%%.*}"; wmin="$(printf '%s' "$want" | cut -d. -f2)"
    gmaj="${got%%.*}"; gmin="$(printf '%s' "$got" | cut -d. -f2)"
    if [ "$gmaj" -lt "$wmaj" ] 2>/dev/null ||
      { [ "$gmaj" -eq "$wmaj" ] && [ "${gmin:-0}" -lt "${wmin:-0}" ]; } 2>/dev/null; then
      die "Go $want+ required, found $got"
    fi
  fi
}

webkit_build_tag() {
  if have pkg-config; then
    if pkg-config --exists webkit2gtk-4.1 2>/dev/null; then echo "webkit2_41"; return; fi
    if pkg-config --exists webkit2gtk-4.0 2>/dev/null; then echo "webkit2_40"; return; fi
  fi
  echo "webkit2_41" # Ubuntu 24.04 default
}

build_from_source() {
  log "Building from source ($SOURCE_DIR)"
  if [ -n "$SOURCE_URL" ]; then
    local clone_dir
    clone_dir="$(mktemp -d)/llamaherd"
    have git || die "git is required to clone $SOURCE_URL"
    log "Cloning $SOURCE_URL"
    run git clone --depth 1 "$SOURCE_URL" "$clone_dir"
    SOURCE_DIR="$clone_dir"
  fi
  [ -f "$SOURCE_DIR/go.mod" ] || die "not a LlamaHerd checkout: $SOURCE_DIR (no go.mod)"

  check_go
  local tags="desktop,production"
  if [ "$OS" = "linux" ]; then
    tags="$(webkit_build_tag),$tags"
  fi
  log "Build tags: $tags"
  BUILD_OUT="$(mktemp -d)/$APP"
  ( cd "$SOURCE_DIR" && run env CGO_ENABLED=1 go mod download )
  ( cd "$SOURCE_DIR" && run env CGO_ENABLED=1 go build -tags "$tags" -o "$BUILD_OUT" . )
  [ "$DRY_RUN" = 1 ] || [ -x "$BUILD_OUT" ] || die "build produced no binary"
  ASSET_DIR="$SOURCE_DIR"
  BIN_SRC="$BUILD_OUT"
}

# ---------------------------------------------------------------------------
# release download
# ---------------------------------------------------------------------------
resolve_repo_slug() {
  [ -n "$REPO_SLUG" ] && return 0
  REPO_SLUG="$BUILTIN_REPO"
  [ -n "$REPO_SLUG" ] && return 0
  if [ -d "$SOURCE_DIR/.git" ] || git -C "$SOURCE_DIR" rev-parse >/dev/null 2>&1; then
    local url
    url="$(git -C "$SOURCE_DIR" config --get remote.origin.url 2>/dev/null || true)"
    case "$url" in
      *github.com[:/]*) REPO_SLUG="$(printf '%s' "$url" | sed -E 's#.*github\.com[:/]##; s#\.git$##')" ;;
    esac
  fi
  [ -n "$REPO_SLUG" ] || die "cannot determine the GitHub repo. Pass --repo OWNER/REPO
or set LLAMAHERD_REPO."
}

releases_base() {
  echo "${LLAMAHERD_RELEASE_BASE_URL:-https://github.com/$REPO_SLUG/releases}"
}

download_file() {
  local url="$1" dest="$2"
  log "Downloading $url"
  if [ "$DRY_RUN" = 1 ]; then
    printf '  %s[dry-run]%s download -> %s\n' "$C_DIM" "$C_RESET" "$dest"
    return 0
  fi
  if have curl; then
    curl -fL --retry 3 --connect-timeout 15 -o "$dest" "$url"
  elif have wget; then
    wget -O "$dest" "$url"
  else
    die "need curl or wget to download releases"
  fi
}

download_release() {
  resolve_repo_slug
  local ext="tar.gz"
  [ "$OS" = "darwin" ] && ext="tar.gz"
  local asset="${APP}_${OS}_${ARCH}.${ext}"
  local url
  if [ "$RELEASE_TAG" = "latest" ]; then
    url="$(releases_base)/latest/download/$asset"
  else
    url="$(releases_base)/download/$RELEASE_TAG/$asset"
  fi
  log "Fetching prebuilt binary ($OS/$ARCH, tag $RELEASE_TAG)"

  local tmp
  tmp="$(mktemp -d)"
  if [ "$DRY_RUN" = 1 ]; then
    printf '  %s[dry-run]%s %s -> %s\n' "$C_DIM" "$C_RESET" "$url" "$tmp"
    BIN_SRC="$tmp/$APP"
    ASSET_DIR="$tmp"
    return 0
  fi
  download_file "$url" "$tmp/$asset"
  have tar || die "tar is required"
  tar -xzf "$tmp/$asset" -C "$tmp"
  [ -x "$tmp/$APP" ] || chmod +x "$tmp/$APP" 2>/dev/null || true
  BIN_SRC="$tmp/$APP"
  ASSET_DIR="$tmp"
}

# ---------------------------------------------------------------------------
# install steps
# ---------------------------------------------------------------------------
install_file() { # src dest [mode]
  local src="$1" dest="$2" mode="${3:-0755}" dir
  dir="$(dirname "$dest")"
  if [ "$DRY_RUN" = 1 ]; then
    printf '  %s[dry-run]%s install -m %s %s %s\n' "$C_DIM" "$C_RESET" "$mode" "$src" "$dest"
    return 0
  fi
  mkdir -p "$dir" 2>/dev/null || { local SUDO=""; [ "$(id -u)" -ne 0 ] && SUDO="sudo"; $SUDO mkdir -p "$dir"; }
  if [ -w "$dir" ] || { [ -e "$dest" ] && [ -w "$dest" ]; }; then
    run install -m "$mode" "$src" "$dest"
  else
    local SUDO=""; [ "$(id -u)" -ne 0 ] && SUDO="sudo"
    run $SUDO install -m "$mode" "$src" "$dest"
  fi
}

install_binary() {
  [ -n "${BIN_SRC:-}" ] || die "internal: no binary to install"
  log "Installing binary to $BIN_DIR/$APP"
  install_file "$BIN_SRC" "$BIN_DIR/$APP" 0755
  manifest_add "$BIN_DIR/$APP"
  command -v "$APP" >/dev/null 2>&1 || \
    info "note: $BIN_DIR is not on your PATH (add it to use the '$APP' command anywhere)"
}

install_launcher() {
  [ "$WITH_DESKTOP" = 1 ] || return 0
  local launcher="$BIN_DIR/$APP-gui"
  log "Installing GUI launcher $launcher"
  if [ "$DRY_RUN" = 1 ]; then
    printf '  %s[dry-run]%s write %s\n' "$C_DIM" "$C_RESET" "$launcher"
  else
    cat >"$launcher" <<EOF
#!/usr/bin/env bash
# LlamaHerd GUI launcher (installed by scripts/install.sh).
# Sets the WSLg-friendly GUI environment, then opens the window (no args).
export GSK_RENDERER="\${GSK_RENDERER:-cairo}"
if [ -z "\${DISPLAY:-}" ] && [ -S /tmp/.X11-unix/X0 ]; then
  export DISPLAY=:0
fi
exec "$BIN_DIR/$APP" "\$@"
EOF
    chmod 0755 "$launcher"
  fi
  manifest_add "$launcher"
}

seed_data() {
  log "Seeding presets into $DATA_DIR"
  mkdir -p "$DATA_DIR/configs" "$DATA_DIR/library" 2>/dev/null || true
  local src_cfg="$ASSET_DIR/configs"
  [ -d "$src_cfg" ] || { info "no shipped configs to seed"; return 0; }
  local f base
  for f in "$src_cfg"/*.json; do
    [ -e "$f" ] || continue
    base="$(basename "$f")"
    if [ -e "$DATA_DIR/configs/$base" ]; then
      info "kept existing configs/$base"
    else
      run cp "$f" "$DATA_DIR/configs/$base"
      info "installed configs/$base"
    fi
  done
}

json_escape() { printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'; }

detect_server_bin() {
  if [ -n "${LLAMAHERD_SERVER_BIN:-}" ]; then printf '%s' "$LLAMAHERD_SERVER_BIN"; return; fi
  if have llama-server; then command -v llama-server; return; fi
  local c
  for c in \
    "$HOME/llama.cpp/build-latest/bin/llama-server" \
    "$HOME/llama.cpp/build/bin/llama-server" \
    "$HOME/llama.cpp/bin/llama-server"; do
    [ -x "$c" ] && { printf '%s' "$c"; return; }
  done
}

detect_models_dir() {
  if [ -n "${LLAMAHERD_MODELS_DIR:-}" ]; then printf '%s' "$LLAMAHERD_MODELS_DIR"; return; fi
  [ -d "$HOME/llm/models" ] && { printf '%s' "$HOME/llm/models"; return; }
  [ -d "$HOME/models" ] && { printf '%s' "$HOME/models"; return; }
}

write_config() {
  [ "$WITH_CONFIG" = 1 ] || { info "skipping user config (--no-config)"; return 0; }
  local cfg="$CONFIG_DIR/config.json"
  if [ -e "$cfg" ]; then
    info "kept existing config $cfg"
    return 0
  fi
  local sb md
  sb="$(detect_server_bin)"
  md="$(detect_models_dir)"
  if [ -z "$sb" ] && [ -z "$md" ]; then
    info "no llama-server / models dir detected; first run will ask for them"
    return 0
  fi
  local parts=() out=""
  [ -n "$sb" ] && parts+=("  \"serverBin\": \"$(json_escape "$sb")\"")
  [ -n "$md" ] && parts+=("  \"modelsDir\": \"$(json_escape "$md")\"")
  for p in "${parts[@]}"; do
    [ -n "$out" ] && out="$out,\\n"
    out="$out$p"
  done
  log "Writing user config $cfg"
  if [ "$DRY_RUN" = 1 ]; then
    printf '  %s[dry-run]%s %s\n' "$C_DIM" "$C_RESET" "$cfg"; return 0
  fi
  mkdir -p "$CONFIG_DIR"
  printf '{\n%b\n}\n' "$out" >"$cfg"
  info "serverBin=$( [ -n "$sb" ] && echo "$sb" || echo '(unset)')"
  info "modelsDir=$( [ -n "$md" ] && echo "$md" || echo '(unset)')"
}

resolve_icon() {
  if [ -n "$ICON_SRC" ]; then printf '%s' "$ICON_SRC"; return; fi
  local c
  for c in "$ASSET_DIR/llamaherd.png" "$REPO_ROOT/assets/llamaherd.png"; do
    [ -f "$c" ] && { printf '%s' "$c"; return; }
  done
}

install_desktop_linux() {
  [ "$WITH_DESKTOP" = 1 ] || return 0
  [ "$OS" = "linux" ] || return 0
  local apps="$PREFIX/share/applications"
  local icons="$PREFIX/share/icons/hicolor/256x256/apps"
  local desktop="$apps/$APP.desktop"
  log "Installing desktop entry $desktop"
  local icon
  icon="$(resolve_icon)"
  if [ "$DRY_RUN" = 1 ]; then
    printf '  %s[dry-run]%s write %s\n' "$C_DIM" "$C_RESET" "$desktop"
    return 0
  fi
  if [ -n "$icon" ]; then
    install_file "$icon" "$icons/$APP.png" 0644
    manifest_add "$icons/$APP.png"
  else
    warn "no PNG icon found; desktop entry will use a generic icon"
  fi
  mkdir -p "$apps" 2>/dev/null || { local SUDO=""; [ "$(id -u)" -ne 0 ] && SUDO="sudo"; $SUDO mkdir -p "$apps"; }
  cat >"$desktop" <<EOF
[Desktop Entry]
Type=Application
Name=LlamaHerd
GenericName=llama.cpp control panel
Comment=Manage and run local GGUF models with llama.cpp
Exec=$BIN_DIR/$APP-gui
Icon=$APP
Terminal=false
Categories=Development;Utility;
Keywords=llama;llama.cpp;gguf;llm;ai;
EOF
  chmod 0644 "$desktop"
  manifest_add "$desktop"
  if have update-desktop-database; then
    run update-desktop-database "$apps" 2>/dev/null || true
  fi
}

install_app_bundle_macos() {
  [ "$WITH_DESKTOP" = 1 ] || return 0
  [ "$OS" = "darwin" ] || return 0
  local bundle="$HOME/Applications/LlamaHerd.app"
  log "Creating macOS app bundle $bundle"
  if [ "$DRY_RUN" = 1 ]; then
    printf '  %s[dry-run]%s create %s\n' "$C_DIM" "$C_RESET" "$bundle"; return 0
  fi
  mkdir -p "$bundle/Contents/MacOS" "$bundle/Contents/Resources"
  cp "$BIN_SRC" "$bundle/Contents/MacOS/LlamaHerd"
  chmod 0755 "$bundle/Contents/MacOS/LlamaHerd"
  local icon
  icon="$(resolve_icon)"
  if [ -n "$icon" ]; then
    cp "$icon" "$bundle/Contents/Resources/llamaherd.png"
    if have sips && have iconutil; then
      local iconset
      iconset="$(mktemp -d)/icon.iconset"
      mkdir -p "$iconset"
      for s in 16 32 64 128 256 512; do
        sips -z "$s" "$s" "$icon" --out "$iconset/icon_${s}x${s}.png" >/dev/null 2>&1 || true
      done
      iconutil -c icns "$iconset" -o "$bundle/Contents/Resources/llamaherd.icns" >/dev/null 2>&1 || true
    fi
  fi
  cat >"$bundle/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>LlamaHerd</string>
  <key>CFBundleDisplayName</key><string>LlamaHerd</string>
  <key>CFBundleIdentifier</key><string>io.llamaherd.app</string>
  <key>CFBundleVersion</key><string>1.0</string>
  <key>CFBundleShortVersionString</key><string>1.0</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleExecutable</key><string>LlamaHerd</string>
  <key>CFBundleIconFile</key><string>llamaherd</string>
  <key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
EOF
  manifest_add "$bundle/Contents/MacOS/LlamaHerd"
  manifest_add "$bundle/Contents/Info.plist"
}

write_manifest() {
  [ "$DRY_RUN" = 1 ] && return 0
  mkdir -p "$(dirname "$MANIFEST")" 2>/dev/null || true
  if [ -f "$MANIFEST.tmp" ]; then
    sort -u "$MANIFEST.tmp" >"$MANIFEST"
    rm -f "$MANIFEST.tmp"
  else
    : >"$MANIFEST"
  fi
}

# ---------------------------------------------------------------------------
# main
# ---------------------------------------------------------------------------
main() {
  if [ "$UNINSTALL" = 1 ]; then
    do_uninstall
    return 0
  fi

  # Auto-pick the mode: explicit wins, else source if we're in a checkout.
  if [ -z "$MODE" ]; then
    if [ -f "$SOURCE_DIR/go.mod" ]; then MODE="source"; else MODE="release"; fi
  fi
  if [ "$MODE" = "source" ] && [ -z "$SOURCE_URL" ] && [ ! -f "$SOURCE_DIR/go.mod" ]; then
    MODE="release"
  fi

  log "LlamaHerd installer — $PLATFORM"
  info "mode: $MODE   prefix: $PREFIX   data: $DATA_DIR"
  [ "$DRY_RUN" = 1 ] && warn "dry run: nothing will be changed"

  install_deps

  if [ "$MODE" = "source" ]; then
    build_from_source
  else
    download_release
  fi

  install_binary
  install_launcher
  seed_data
  write_config
  install_desktop_linux
  install_app_bundle_macos
  write_manifest

  log "Done."
  echo
  info "Binary:   $BIN_DIR/$APP   (also the CLI: '$APP list', '$APP status', …)"
  [ "$WITH_DESKTOP" = 1 ] && info "Launcher: $BIN_DIR/$APP-gui  (desktop/Start entry: LlamaHerd)"
  info "Config:   $CONFIG_DIR/config.json"
  info "Data:     $DATA_DIR  (configs/, library/)"
  echo
  info "Uninstall any time with: $PROG --uninstall"
}

main "$@"
