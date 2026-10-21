#!/usr/bin/env bash
# LlamaHerd installer — Linux, macOS, and WSL (terminal only).
#
# Installs the llamaherd binary either by building a local source checkout with
# Go, or by downloading a prebuilt asset from GitHub Releases. Optionally
# installs system dependencies, seeds the shipped presets, writes a desktop
# entry / .app bundle, sets up llama.cpp (detect an existing llama-server, take
# a path, or clone + build it), detects models and writes the user config,
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
DEPS="yes"                      # yes | no
WITH_DESKTOP=1
WITH_CONFIG=1
DRY_RUN=0
ASSUME_YES=0
UNINSTALL=0
PURGE=0
ICON_SRC=""

# llama.cpp (optional): point at an existing build or clone + build it.
LLAMA_MODE=""                   # "" | path | source | skip
LLAMA_SERVER=""                 # serverBin chosen by --llama-server / detection / build
LLAMA_CONFIG_DIR=""             # checkout to record as llamaDir when we built it
LLAMA_DIR="${LLAMA_DIR:-${LLAMAHERD_LLAMA_DIR:-$HOME/llama.cpp}}"
LLAMA_BUILD_DIR="${LLAMA_BUILD_DIR:-build-latest}"
LLAMA_BACKEND="${LLAMAHERD_LLAMA_BACKEND:-auto}"   # auto | cuda | vulkan | metal | cpu
LLAMA_GIT_URL="${LLAMAHERD_LLAMA_GIT_URL:-https://github.com/ggml-org/llama.cpp}"
CUDA_NVCC=""                    # nvcc chosen for the CUDA backend (set at build time)
CUDA_BIN_DIR=""
CUDA_ROOT=""
BACKEND=""                      # resolved backend (auto -> cuda|cpu)
WIN_SHORTCUT=0                  # Windows Desktop/Start shortcut created (WSL)

# Set this to your "OWNER/REPO" so the piped one-liner needs no --repo flag:
#   curl -fsSL .../install.sh | bash -s -- --from-release
# (Left empty in the source repo; --repo / LLAMAHERD_REPO still override it.)
BUILTIN_REPO="halilcagriakkuzu/llamaherd"

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

# Expand a leading ~/ (or a bare ~) in a user-supplied path.
expand_home() {
  case "$1" in
    "~") printf '%s' "$HOME" ;;
    "~/"*) printf '%s' "$HOME/${1#\~/}" ;;
    *) printf '%s' "$1" ;;
  esac
}

# Interactive questions go through /dev/tty: `curl … | bash` leaves stdin as
# the script pipe, so reading stdin would eat the script itself.
TTY_OK=0
tty_ok() { { true </dev/tty; } 2>/dev/null; }

# say_tty LINE... — print to the terminal, silently doing nothing without one.
say_tty() {
  [ "$TTY_OK" = 1 ] && printf '%s\n' "$*" >/dev/tty 2>/dev/null
  return 0
}

# ask PROMPT — read one line from the terminal (empty when there is none).
ask() {
  local reply=""
  if [ "$TTY_OK" = 1 ]; then
    printf '%s' "$1" >/dev/tty 2>/dev/null || true
    IFS= read -r reply </dev/tty 2>/dev/null || reply=""
  fi
  printf '%s' "$reply"
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
  --with-deps            install missing system packages (the default)
  --no-deps              never touch system packages
  --no-desktop           skip the .desktop entry / .app bundle / Windows shortcut
  --no-config            do not write the user config file
  --icon FILE            icon to install (PNG for the desktop entry, ICO for the
                         Windows shortcut; default: bundled assets/)
  -y, --yes              assume yes (non-interactive)
  --dry-run              print what would happen; change nothing

llama.cpp (only needed when no llama-server is found):
  --llama-server PATH    use this llama-server build
  --install-llamacpp     clone llama.cpp and build it (installs git/cmake/toolchain)
  --llama-dir DIR        checkout to reuse or build (default: ~/llama.cpp)
  --llama-build-dir DIR  build dir inside the checkout (default: build-latest)
  --llama-backend B      auto|cuda|vulkan|metal|cpu (default: auto; CUDA installs
                         the toolkit when an NVIDIA GPU is present but nvcc is not)
  --no-llamacpp          never ask about llama.cpp; leave serverBin unset

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
    --llama-server)
      need_arg $# "$1"; LLAMA_SERVER="$2"; LLAMA_MODE="path"; shift ;;
    --install-llamacpp)
      LLAMA_MODE="source" ;;
    --llama-dir)
      need_arg $# "$1"; LLAMA_DIR="$2"; shift ;;
    --llama-build-dir)
      need_arg $# "$1"; LLAMA_BUILD_DIR="$2"; shift ;;
    --llama-backend)
      need_arg $# "$1"; LLAMA_BACKEND="$2"; shift ;;
    --no-llamacpp)
      LLAMA_MODE="skip" ;;
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

LLAMA_DIR="$(expand_home "$LLAMA_DIR")"
case "$LLAMA_BUILD_DIR" in
  /*) LLAMA_BUILD_PATH="$LLAMA_BUILD_DIR" ;;
  *) LLAMA_BUILD_PATH="$LLAMA_DIR/$LLAMA_BUILD_DIR" ;;
esac
case "$LLAMA_BACKEND" in
  auto | cuda | vulkan | metal | cpu) ;;
  *) die "unknown --llama-backend: $LLAMA_BACKEND (auto|cuda|vulkan|metal|cpu)" ;;
esac

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
  # Windows shortcuts created by a WSL install
  remove_windows_shortcut

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

# ldconfig_bin prints ldconfig even when /sbin is not on PATH.
ldconfig_bin() {
  if have ldconfig; then command -v ldconfig; return 0; fi
  local c
  for c in /sbin/ldconfig /usr/sbin/ldconfig; do
    [ -x "$c" ] && { printf '%s' "$c"; return 0; }
  done
  return 0
}

# gui_libs_present is true when the GUI libraries are already usable, so a
# re-run does not need sudo for packages that are there.
gui_libs_present() {
  if [ "${MODE:-}" = "source" ]; then
    have pkg-config || return 1
    pkg-config --exists gtk+-3.0 2>/dev/null || return 1
    pkg-config --exists webkit2gtk-4.1 2>/dev/null ||
      pkg-config --exists webkit2gtk-4.0 2>/dev/null
    return
  fi
  local ld libs
  ld="$(ldconfig_bin)"
  [ -n "$ld" ] || return 1
  libs="$("$ld" -p 2>/dev/null || true)"
  case "$libs" in
    *libgtk-3*) ;;
    *) return 1 ;;
  esac
  case "$libs" in
    *libwebkit2gtk-4*) ;;
    *) return 1 ;;
  esac
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

  if gui_libs_present; then
    info "GUI libraries already installed"
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

  log "Installing system dependencies: $packages"
  local SUDO=""
  [ "$(id -u)" -ne 0 ] && SUDO="sudo"
  if [ -n "$SUDO" ] && ! have sudo; then
    warn "sudo not found; install $packages as root, then re-run"
    return 0
  fi

  local ok=0
  if have apt-get; then
    run $SUDO apt-get update || warn "apt-get update failed; trying the install anyway"
    # shellcheck disable=SC2086
    if run $SUDO apt-get install -y $packages; then ok=1; fi
  elif have dnf; then
    if run $SUDO dnf install -y $packages; then ok=1; fi
  elif have pacman; then
    if run $SUDO pacman -Sy --noconfirm $packages; then ok=1; fi
  elif have zypper; then
    if run $SUDO zypper --non-interactive install $packages; then ok=1; fi
  fi
  if [ "$ok" = 1 ]; then
    info "system dependencies installed"
  else
    warn "system packages could not be installed; the GUI may not start"
    info "install them manually or re-run this installer with --with-deps"
  fi
}

# ---------------------------------------------------------------------------
# llama.cpp (optional): reuse an existing build or clone + build one
# ---------------------------------------------------------------------------
detect_cuda_arch() {
  command -v nvidia-smi >/dev/null 2>&1 || return 1
  local caps
  caps="$(nvidia-smi --query-gpu=compute_cap --format=csv,noheader 2>/dev/null |
    tr -d ' \r' | grep -E '^[0-9]+\.[0-9]+$' | tr -d '.' | sort -un | paste -sd';' -)"
  [ -n "$caps" ] && printf '%s' "$caps"
}

# cuda_nvcc prints the path of an installed CUDA compiler, if any. An NVIDIA
# driver (nvidia-smi) alone is not enough to build with CUDA.
cuda_nvcc() {
  if have nvcc; then command -v nvcc; return 0; fi
  local d
  for d in "${CUDA_HOME:-}" "${CUDA_PATH:-}" "${CUDAToolkit_ROOT:-}" \
    /usr/local/cuda /opt/cuda /usr/local/cuda-*; do
    [ -n "$d" ] && [ -x "$d/bin/nvcc" ] && { printf '%s' "$d/bin/nvcc"; return 0; }
  done
  return 0
}

# nvidia_keyring_url prints the CUDA repository keyring for this distro.
nvidia_keyring_url() {
  local id="" ver="" seg="" archdir=""
  [ -r /etc/os-release ] || return 1
  id="$(. /etc/os-release && printf '%s' "${ID:-}")"
  ver="$(. /etc/os-release && printf '%s' "${VERSION_ID:-}")"
  case "$ARCH" in
    amd64) archdir="x86_64" ;;
    arm64) archdir="sbsa" ;;
    *) return 1 ;;
  esac
  if [ "$IS_WSL" = 1 ] && [ "$id" = "ubuntu" ]; then
    seg="wsl-ubuntu"
  else
    case "$id" in
      ubuntu) seg="ubuntu${ver//./}" ;;
      debian) seg="debian${ver%%.*}" ;;
      *) return 1 ;;
    esac
  fi
  printf 'https://developer.download.nvidia.com/compute/cuda/repos/%s/%s/cuda-keyring_1.1-1_all.deb' \
    "$seg" "$archdir"
}

# install_cuda_toolkit installs nvcc when an NVIDIA GPU is present but no
# toolkit is. Returns non-zero when it cannot (the caller then falls back).
install_cuda_toolkit() {
  local nvcc
  nvcc="$(cuda_nvcc)"
  [ -n "$nvcc" ] && return 0
  if [ "$DEPS" = "no" ]; then
    warn "the CUDA toolkit (nvcc) is missing and --no-deps was given"
    return 1
  fi

  warn "NVIDIA GPU detected, but the CUDA toolkit (nvcc) is not installed"
  info "installing it; this is a large download (several GB)"
  local SUDO=""
  [ "$(id -u)" -ne 0 ] && SUDO="sudo"
  if [ -n "$SUDO" ] && ! have sudo; then
    warn "sudo not found; install the CUDA toolkit as root, then re-run"
    return 1
  fi

  if have apt-get; then
    local url deb
    url="$(nvidia_keyring_url || true)"
    if [ -n "$url" ]; then
      log "Adding the NVIDIA CUDA repository"
      deb="$(mktemp -d)/cuda-keyring.deb"
      if download_file "$url" "$deb" &&
        run $SUDO dpkg -i "$deb" &&
        run $SUDO apt-get update; then
        if run $SUDO apt-get install -y cuda-toolkit && [ -n "$(cuda_nvcc)" ]; then
          return 0
        fi
        warn "cuda-toolkit could not be installed from the NVIDIA repository"
      else
        warn "could not set up the NVIDIA CUDA repository"
      fi
    fi
    # Older distro toolkit as a fallback (no extra repository needed).
    if apt-cache show nvidia-cuda-toolkit >/dev/null 2>&1; then
      log "Installing nvidia-cuda-toolkit from the distribution"
      run $SUDO apt-get update || true
      run $SUDO apt-get install -y nvidia-cuda-toolkit && [ -n "$(cuda_nvcc)" ] && return 0
    fi
  elif have dnf; then
    if dnf info -q cuda-toolkit >/dev/null 2>&1; then
      run $SUDO dnf install -y cuda-toolkit && [ -n "$(cuda_nvcc)" ] && return 0
    else
      info "enable the NVIDIA CUDA repository (cuda-fedora*), then install cuda-toolkit"
    fi
  elif have pacman; then
    run $SUDO pacman -Sy --noconfirm cuda && [ -n "$(cuda_nvcc)" ] && return 0
  elif have zypper; then
    run $SUDO zypper --non-interactive install cuda-toolkit && [ -n "$(cuda_nvcc)" ] && return 0
  fi
  return 1
}

# resolve_llama_backend turns --llama-backend auto into cuda|cpu and, for cuda,
# makes sure nvcc exists (installing the toolkit when it is missing).
resolve_llama_backend() {
  BACKEND="$LLAMA_BACKEND"
  if [ "$BACKEND" = "auto" ]; then
    if [ "$OS" = "linux" ] && have nvidia-smi; then
      BACKEND="cuda"
    else
      BACKEND="cpu"
    fi
  fi
  [ "$BACKEND" = "cuda" ] || return 0

  CUDA_NVCC="$(cuda_nvcc)"
  if [ -z "$CUDA_NVCC" ] && [ "$DRY_RUN" = 1 ]; then
    info "CUDA toolkit (nvcc) missing; it would be installed before configuring"
    return 0
  fi
  if [ -z "$CUDA_NVCC" ]; then
    install_cuda_toolkit || true
    CUDA_NVCC="$(cuda_nvcc)"
  fi
  if [ -z "$CUDA_NVCC" ]; then
    if [ "$LLAMA_BACKEND" = "cuda" ]; then
      die "CUDA backend requested, but the CUDA toolkit (nvcc) is still missing"
    fi
    warn "building without CUDA: no usable CUDA toolkit (the CPU build still works)"
    warn "install the toolkit and re-run with --llama-backend cuda for GPU support"
    BACKEND="cpu"
    return 0
  fi
  CUDA_BIN_DIR="$(dirname "$CUDA_NVCC")"
  CUDA_ROOT="$(dirname "$CUDA_BIN_DIR")"
  info "CUDA toolkit: $CUDA_ROOT (nvcc $CUDA_NVCC)"
}

# install_llama_tools installs what building llama.cpp needs. Choosing the
# source build is the consent, so it does not ask again (--no-deps still wins).
install_llama_tools() {
  local missing=()
  have git || missing+=("git")
  have cmake || missing+=("cmake")
  if ! have cc && ! have gcc && ! have clang; then missing+=("cc"); fi
  [ "${#missing[@]}" -eq 0 ] && return 0

  if [ "$DEPS" = "no" ]; then
    die "building llama.cpp needs ${missing[*]} (install them, or drop --no-deps)"
  fi

  if [ "$OS" = "darwin" ]; then
    xcode-select -p >/dev/null 2>&1 ||
      die "install the Xcode Command Line Tools first: xcode-select --install"
    if ! have cmake; then
      have brew || die "cmake is required; install it from https://cmake.org/download/ (or 'brew install cmake')"
      run brew install cmake
    fi
    return 0
  fi

  local packages=""
  if have apt-get; then
    packages="git cmake build-essential"
  elif have dnf; then
    packages="git cmake gcc gcc-c++ make"
  elif have pacman; then
    packages="git cmake base-devel"
  elif have zypper; then
    packages="git cmake gcc gcc-c++ make"
  else
    die "cannot install ${missing[*]}: unknown package manager (install them manually)"
  fi

  log "Installing llama.cpp build tools: $packages"
  local SUDO=""
  [ "$(id -u)" -ne 0 ] && SUDO="sudo"
  if [ -n "$SUDO" ] && ! have sudo; then
    die "sudo not found; install $packages as root, then re-run"
  fi
  if have apt-get; then
    run $SUDO apt-get update
    # shellcheck disable=SC2086
    run $SUDO apt-get install -y $packages
  elif have dnf; then
    run $SUDO dnf install -y $packages
  elif have pacman; then
    run $SUDO pacman -Sy --noconfirm $packages
  else
    run $SUDO zypper --non-interactive install $packages
  fi
}

# build_llamacpp clones (unless a checkout is already there) and builds
# llama-server like scripts/update-llamacpp.sh does.
build_llamacpp() {
  log "Building llama.cpp from source"
  info "checkout:  $LLAMA_DIR"
  info "build dir: $LLAMA_BUILD_PATH"
  install_llama_tools

  if [ -d "$LLAMA_DIR/.git" ]; then
    info "reusing the existing checkout"
  elif [ -d "$LLAMA_DIR" ] && [ -n "$(ls -A "$LLAMA_DIR" 2>/dev/null)" ]; then
    die "$LLAMA_DIR is not empty and is not a git checkout (pick one with --llama-dir)"
  else
    log "Cloning $LLAMA_GIT_URL"
    # A shallow clone breaks llama.cpp's build number (it counts commits), so
    # keep the full history: blobless is cheap and still counts correctly.
    if ! run git clone --filter=blob:none "$LLAMA_GIT_URL" "$LLAMA_DIR"; then
      warn "partial clone failed; retrying with the full history"
      run git clone "$LLAMA_GIT_URL" "$LLAMA_DIR"
    fi
  fi

  # Older installs used --depth 1; repair those so builds report a real number.
  if [ "$DRY_RUN" != 1 ] && [ -d "$LLAMA_DIR/.git" ] &&
    [ "$(git -C "$LLAMA_DIR" rev-parse --is-shallow-repository 2>/dev/null)" = "true" ]; then
    log "Fetching the full history (the llama.cpp build number counts commits)"
    run git -C "$LLAMA_DIR" fetch --unshallow ||
      warn "could not unshallow $LLAMA_DIR; the build number will stay low"
  fi

  resolve_llama_backend
  local flags=("-DCMAKE_BUILD_TYPE=Release")
  local arch=""
  case "$BACKEND" in
    cuda)
      flags+=("-DGGML_CUDA=ON")
      [ -n "$CUDA_ROOT" ] && flags+=("-DCUDAToolkit_ROOT=$CUDA_ROOT")
      [ -n "$CUDA_NVCC" ] && flags+=("-DCMAKE_CUDA_COMPILER=$CUDA_NVCC")
      arch="$(detect_cuda_arch || true)"
      [ -n "$arch" ] && flags+=("-DCMAKE_CUDA_ARCHITECTURES=$arch")
      ;;
    vulkan) flags+=("-DGGML_VULKAN=ON") ;;
    metal) flags+=("-DGGML_METAL=ON") ;;
  esac
  log "Backend: $BACKEND"

  run env "PATH=$PATH${CUDA_BIN_DIR:+:$CUDA_BIN_DIR}" \
    cmake -S "$LLAMA_DIR" -B "$LLAMA_BUILD_PATH" "${flags[@]}"
  local jobs
  jobs="$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)"
  run cmake --build "$LLAMA_BUILD_PATH" --config Release -j"$jobs"

  LLAMA_SERVER="$LLAMA_BUILD_PATH/bin/llama-server"
  LLAMA_CONFIG_DIR="$LLAMA_DIR"
  if [ "$DRY_RUN" != 1 ] && [ ! -x "$LLAMA_SERVER" ]; then
    die "build finished but $LLAMA_SERVER is missing"
  fi
  info "built $LLAMA_SERVER"
}

# ensure_llama_server decides what serverBin goes into the config: an explicit
# flag, an existing installation, or an interactive choice (path or build).
ensure_llama_server() {
  [ "$WITH_CONFIG" = 1 ] || { info "skipping llama.cpp setup (--no-config)"; return 0; }

  case "$LLAMA_MODE" in
    path)
      LLAMA_SERVER="$(expand_home "$LLAMA_SERVER")"
      [ -x "$LLAMA_SERVER" ] || die "not an executable llama-server: $LLAMA_SERVER"
      info "using llama-server $LLAMA_SERVER"
      return 0 ;;
    source)
      build_llamacpp
      return 0 ;;
    skip)
      info "skipping llama.cpp setup (--no-llamacpp)"
      return 0 ;;
  esac

  # Automatic: keep a user's existing config, else look around, else ask.
  if [ -e "$CONFIG_DIR/config.json" ]; then
    info "llama.cpp already configured in $CONFIG_DIR/config.json"
    return 0
  fi
  LLAMA_SERVER="$(detect_server_bin)"
  if [ -n "$LLAMA_SERVER" ]; then
    info "found llama-server at $LLAMA_SERVER"
    return 0
  fi

  if [ "$DRY_RUN" = 1 ] || [ "$ASSUME_YES" = 1 ] || ! tty_ok; then
    info "no llama-server found; the first run will ask for it"
    info "  (re-run with --install-llamacpp to clone and build it here)"
    return 0
  fi

  TTY_OK=1
  say_tty ""
  say_tty "No llama-server found. How should the installer provide it?"
  say_tty "  1) I already have a build — ask me for the path  [default]"
  say_tty "  2) Clone llama.cpp and build it from source (installs git, cmake and a C/C++ toolchain)"
  say_tty "  3) Skip for now — configure it on first launch"
  local choice="" path=""
  choice="$(ask "Choice [1/2/3]: ")"
  case "$choice" in
    2) build_llamacpp ;;
    3) info "skipped; set the server path on first launch" ;;
    *)
      path="$(ask "Path to llama-server: ")"
      path="$(expand_home "$path")"
      if [ -z "$path" ]; then
        info "skipped; set the server path on first launch"
      elif [ -x "$path" ]; then
        LLAMA_SERVER="$path"
        info "using llama-server $path"
      else
        warn "not an executable file: $path; leaving serverBin unset"
      fi
      ;;
  esac
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

install_update_script() {
  local src="" c
  for c in \
    "$ASSET_DIR/update-llamacpp.sh" \
    "$SOURCE_DIR/scripts/update-llamacpp.sh" \
    "$REPO_ROOT/scripts/update-llamacpp.sh"; do
    [ -f "$c" ] && { src="$c"; break; }
  done
  if [ -z "$src" ]; then
    info "no update-llamacpp.sh found; the in-app llama.cpp update will be unavailable"
    return 0
  fi
  log "Installing llama.cpp updater $BIN_DIR/$APP-update-llamacpp.sh"
  install_file "$src" "$BIN_DIR/$APP-update-llamacpp.sh" 0755
  manifest_add "$BIN_DIR/$APP-update-llamacpp.sh"
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
  if [ -n "${LLAMAHERD_SERVER_BIN:-}" ]; then printf '%s' "$LLAMAHERD_SERVER_BIN"; return 0; fi
  if have llama-server; then command -v llama-server; return 0; fi
  local c
  for c in \
    "$LLAMA_BUILD_PATH/bin/llama-server" \
    "$LLAMA_DIR/build/bin/llama-server" \
    "$HOME/llama.cpp/build-latest/bin/llama-server" \
    "$HOME/llama.cpp/build/bin/llama-server" \
    "$HOME/llama.cpp/bin/llama-server"; do
    [ -x "$c" ] && { printf '%s' "$c"; return 0; }
  done
  return 0
}

detect_models_dir() {
  if [ -n "${LLAMAHERD_MODELS_DIR:-}" ]; then printf '%s' "$LLAMAHERD_MODELS_DIR"; return 0; fi
  [ -d "$HOME/llm/models" ] && { printf '%s' "$HOME/llm/models"; return 0; }
  [ -d "$HOME/models" ] && { printf '%s' "$HOME/models"; return 0; }
  return 0
}

write_config() {
  [ "$WITH_CONFIG" = 1 ] || { info "skipping user config (--no-config)"; return 0; }
  local cfg="$CONFIG_DIR/config.json"
  if [ -e "$cfg" ]; then
    info "kept existing config $cfg"
    return 0
  fi
  local sb md
  sb="${LLAMA_SERVER:-$(detect_server_bin)}"
  md="$(detect_models_dir)"
  if [ -z "$sb" ] && [ -z "$md" ]; then
    info "no llama-server / models dir detected; first run will ask for them"
    return 0
  fi
  local parts=() out=""
  [ -n "$sb" ] && parts+=("  \"serverBin\": \"$(json_escape "$sb")\"")
  [ -n "$md" ] && parts+=("  \"modelsDir\": \"$(json_escape "$md")\"")
  [ -n "$LLAMA_CONFIG_DIR" ] && parts+=("  \"llamaDir\": \"$(json_escape "$LLAMA_CONFIG_DIR")\"")
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
  if [ -n "$LLAMA_CONFIG_DIR" ]; then info "llamaDir=$LLAMA_CONFIG_DIR"; fi
}

resolve_icon() {
  if [ -n "$ICON_SRC" ]; then
    case "$ICON_SRC" in
      *.ico) ;; # the .desktop entry needs a PNG; use the bundled one
      *) printf '%s' "$ICON_SRC"; return ;;
    esac
  fi
  local c
  for c in "$ASSET_DIR/llamaherd.png" "$REPO_ROOT/assets/llamaherd.png"; do
    [ -f "$c" ] && { printf '%s' "$c"; return; }
  done
}

# resolve_icon_ico finds the Windows .ico (the desktop shortcut needs one).
resolve_icon_ico() {
  case "$ICON_SRC" in
    *.ico) printf '%s' "$ICON_SRC"; return ;;
  esac
  local c
  for c in \
    "$ASSET_DIR/llamaherd.ico" \
    "$SOURCE_DIR/assets/llamaherd.ico" \
    "$REPO_ROOT/assets/llamaherd.ico" \
    "$REPO_ROOT/build/windows/icon.ico"; do
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

# powershell_exe prints the Windows PowerShell path usable from WSL.
powershell_exe() {
  if have powershell.exe; then command -v powershell.exe; return 0; fi
  local c
  for c in \
    "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe" \
    "/mnt/c/Program Files/PowerShell/7/pwsh.exe"; do
    [ -x "$c" ] && { printf '%s' "$c"; return 0; }
  done
  return 0
}

# install_windows_shortcut puts LlamaHerd on the Windows desktop and Start menu
# when installing inside WSL: the Linux .desktop file is invisible to Windows.
install_windows_shortcut() {
  [ "$IS_WSL" = 1 ] || return 0
  [ "$WITH_DESKTOP" = 1 ] || return 0
  local ps
  ps="$(powershell_exe)"
  if [ -z "$ps" ] || ! have wslpath; then
    info "no Windows shortcut created; launch the GUI from this terminal with: $APP-gui"
    return 0
  fi
  if [ -z "${WSL_DISTRO_NAME:-}" ]; then
    info "WSL_DISTRO_NAME is unset; skipping the Windows shortcut"
    return 0
  fi

  log "Creating Windows Desktop / Start menu shortcuts (LlamaHerd.lnk)"
  local tmp ico ico_win=""
  tmp="$(mktemp -d)"
  ico="$(resolve_icon_ico)"
  if [ -n "$ico" ]; then
    ico_win="$(wslpath -w "$ico" 2>/dev/null || true)"
  else
    warn "no .ico found; the shortcut will use the default icon"
  fi
  cat >"$tmp/shortcut.ps1" <<'PS'
param([string]$Distro, [string]$Target, [string]$IconIco)
$ErrorActionPreference = 'Stop'
$wslg = 'C:\Program Files\WSL\wslg.exe'
if (-not (Test-Path $wslg)) { $wslg = Join-Path $env:SystemRoot 'System32\wsl.exe' }
$icon = $null
if ($IconIco -and (Test-Path $IconIco)) {
  # Copy under a content-hashed name so Explorer does not keep a stale icon.
  $iconDir = Join-Path $env:USERPROFILE 'LlamaHerd'
  New-Item -ItemType Directory -Force -Path $iconDir | Out-Null
  $hash = (Get-FileHash -Algorithm SHA1 $IconIco).Hash.Substring(0, 8).ToLower()
  $icon = Join-Path $iconDir "icon-$hash.ico"
  Copy-Item -Force $IconIco $icon
  Get-ChildItem $iconDir -Filter 'icon*.ico' -ErrorAction SilentlyContinue |
    Where-Object { $_.FullName -ne $icon } | Remove-Item -Force -ErrorAction SilentlyContinue
} else {
  Write-Warning "LlamaHerd icon not found ($IconIco); using the default icon"
}
$shell = New-Object -ComObject WScript.Shell
foreach ($dir in @([Environment]::GetFolderPath('Desktop'), [Environment]::GetFolderPath('Programs'))) {
  $lnk = $shell.CreateShortcut((Join-Path $dir 'LlamaHerd.lnk'))
  $lnk.TargetPath = $wslg
  $lnk.Arguments = "-d $Distro -- $Target"
  $lnk.WorkingDirectory = 'C:\'
  $lnk.Description = 'LlamaHerd - llama.cpp control panel (WSL)'
  if ($icon) { $lnk.IconLocation = "$icon,0" }
  $lnk.Save()
}
PS
  run "$ps" -NoProfile -NonInteractive -ExecutionPolicy Bypass \
    -File "$(wslpath -w "$tmp/shortcut.ps1")" \
    -Distro "$WSL_DISTRO_NAME" -Target "$BIN_DIR/$APP-gui" \
    -IconIco "$ico_win" && WIN_SHORTCUT=1
}

# remove_windows_shortcut deletes the shortcuts install_windows_shortcut made.
remove_windows_shortcut() {
  [ "$IS_WSL" = 1 ] || return 0
  local ps
  ps="$(powershell_exe)"
  [ -n "$ps" ] && have wslpath || return 0
  local tmp
  tmp="$(mktemp -d)"
  cat >"$tmp/remove.ps1" <<'PS'
$ErrorActionPreference = 'SilentlyContinue'
$shell = New-Object -ComObject WScript.Shell
foreach ($dir in @([Environment]::GetFolderPath('Desktop'), [Environment]::GetFolderPath('Programs'))) {
  $p = Join-Path $dir 'LlamaHerd.lnk'
  if (Test-Path $p) { Remove-Item -Force $p }
}
PS
  run "$ps" -NoProfile -NonInteractive -ExecutionPolicy Bypass \
    -File "$(wslpath -w "$tmp/remove.ps1")"
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
  install_update_script
  seed_data
  ensure_llama_server
  write_config
  install_desktop_linux
  install_app_bundle_macos
  install_windows_shortcut
  write_manifest

  log "Done."
  echo
  info "Binary:   $BIN_DIR/$APP   (also the CLI: '$APP list', '$APP status', …)"
  [ "$WITH_DESKTOP" = 1 ] && info "Launcher: $BIN_DIR/$APP-gui  (desktop/Start entry: LlamaHerd)"
  info "Config:   $CONFIG_DIR/config.json"
  info "Data:     $DATA_DIR  (configs/, library/)"
  [ -n "$LLAMA_SERVER" ] && info "Server:   $LLAMA_SERVER"
  if [ "$IS_WSL" = 1 ]; then
    echo
    if [ "$WITH_DESKTOP" = 1 ]; then
      info "WSL: run '$APP-gui' (or '$APP') in this terminal to open the window."
    else
      info "WSL: run '$APP' in this terminal to open the window."
    fi
    [ "${WIN_SHORTCUT:-0}" = 1 ] &&
      info "     A 'LlamaHerd' shortcut is also on the Windows desktop / Start menu."
    if [ "$OS" = "linux" ]; then
      local ld libs
      ld="$(ldconfig_bin)"
      if [ -n "$ld" ]; then
        libs="$("$ld" -p 2>/dev/null || true)"
        case "$libs" in
          *libwebkit2gtk-4*) ;;
          *)
            warn "the GTK/WebKit GUI libraries look missing; install them with:"
            info "  sudo apt-get install -y libgtk-3-0 libwebkit2gtk-4.1-0"
            info "  (or re-run this installer with --with-deps)"
            ;;
        esac
      fi
    fi
  fi
  echo
  info "Uninstall any time with: $PROG --uninstall"
}

main "$@"
