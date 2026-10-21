#!/usr/bin/env bash
# Update the local llama.cpp checkout and rebuild it (CUDA, Release).
#
# Steps (same as the manual routine):
#   1. git pull
#   2. cmake -S . -B build-latest -DGGML_CUDA=ON -DCMAKE_BUILD_TYPE=Release -DCMAKE_CUDA_ARCHITECTURES=<detected>
#   3. cmake --build build-latest --config Release -j$(nproc)
#
# The GUI runs this from the "llama.cpp" tab; it can also be run by hand.
# Override with env vars: LLAMA_DIR, BUILD_DIR, CUDA_ARCH (auto-detected via
# nvidia-smi when unset), JOBS, GIT_PULL_ARGS.
set -euo pipefail

LLAMA_DIR="${LLAMA_DIR:-$HOME/llama.cpp}"
BUILD_DIR="${BUILD_DIR:-build-latest}"
# CUDA architecture(s): CUDA_ARCH wins; otherwise detect the installed GPU(s)'
# compute capability via nvidia-smi (e.g. 8.9 -> 89, several GPUs -> "86;89");
# fall back to "native" (CMake >= 3.24 picks the visible GPU) when undetectable.
detect_cuda_arch() {
    command -v nvidia-smi >/dev/null 2>&1 || return 1
    local caps
    caps="$(nvidia-smi --query-gpu=compute_cap --format=csv,noheader 2>/dev/null \
        | tr -d ' \r' | grep -E '^[0-9]+\.[0-9]+$' | tr -d '.' | sort -un | paste -sd';' -)"
    [ -n "$caps" ] && printf '%s' "$caps"
}
CUDA_ARCH="${CUDA_ARCH:-$(detect_cuda_arch || true)}"
CUDA_ARCH="${CUDA_ARCH:-native}"
JOBS="${JOBS:-$(nproc)}"
# --ff-only: refuse to create a merge commit in the checkout; fail loudly instead.
GIT_PULL_ARGS="${GIT_PULL_ARGS:---ff-only}"

step() { printf '\n==> %s\n' "$*"; }

cd "$LLAMA_DIR"

if pgrep -x llama-server >/dev/null 2>&1; then
    echo "WARNING: llama-server is running (pid $(pgrep -x llama-server | tr '\n' ' ')); the new binary will only be used after it restarts."
fi

before="$(git rev-parse --short HEAD)"
step "1/3 git pull $GIT_PULL_ARGS   (at $before)"
# A shallow clone reports a build number of 1 (llama.cpp counts commits).
if [ "$(git rev-parse --is-shallow-repository 2>/dev/null)" = "true" ]; then
    echo "shallow clone: fetching the full history so the build number is real"
    git fetch --unshallow
fi
# shellcheck disable=SC2086
git pull $GIT_PULL_ARGS
after="$(git rev-parse --short HEAD)"

step "2/3 cmake configure -> $BUILD_DIR (CUDA arch $CUDA_ARCH)"
cmake -S . -B "$BUILD_DIR" \
    -DGGML_CUDA=ON \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_CUDA_ARCHITECTURES="$CUDA_ARCH"

step "3/3 cmake build -j$JOBS"
cmake --build "$BUILD_DIR" --config Release -j"$JOBS"

step "done: $before -> $after"
"$BUILD_DIR/bin/llama-server" --version 2>&1 | head -2
