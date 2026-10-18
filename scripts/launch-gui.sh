#!/usr/bin/env bash
# Launch the LlamaHerd GUI on the WSLg display from the Windows 11 host.
#
# This is what the Windows shortcuts call (via "C:\Program Files\WSL\wslg.exe").
# It always starts the GUI; the CLI/daemon subcommands are used directly.
set -euo pipefail

# Resolve the repo root no matter where the shortcut starts us.
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$DIR"

# GUI environment. WSLg normally provides DISPLAY; fall back to :0 and force
# the software renderer (see AGENTS.md).
export DISPLAY="${DISPLAY:-:0}"
export GSK_RENDERER="${GSK_RENDERER:-cairo}"

BIN="build/bin/llamaherd"
LOG="$DIR/build/launch.log"
mkdir -p "$DIR/build"

# Already running? Just stop (the control API on 127.0.0.1:8517 is a singleton).
if pgrep -x llamaherd >/dev/null 2>&1; then
    echo "llamaherd is already running (pid $(pgrep -x llamaherd | head -1)); not starting a second window."
    exit 0
fi

# Build on first run / after a clean checkout.
if [ ! -x "$BIN" ]; then
    echo "llamaherd: $BIN not found, building..." | tee -a "$LOG"
    (
        cd "$DIR"
        CGO_ENABLED=1 go build -tags webkit2_41,desktop,production -o "$BIN" .
    ) >>"$LOG" 2>&1 || {
        echo "llamaherd: build failed, see $LOG"
        exit 1
    }
fi

exec "./$BIN" "$@"
