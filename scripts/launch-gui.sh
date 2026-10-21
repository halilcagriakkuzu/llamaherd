#!/usr/bin/env bash
# Launch the LlamaHerd GUI on the WSLg display from the Windows 11 host.
#
# This is what the Windows shortcuts call (via "C:\Program Files\WSL\wslg.exe").
# It always starts the GUI; the CLI/daemon subcommands are used directly.
#
# Pass --rebuild to force a build before the window opens (any other argument
# goes to llamaherd). Without it the binary is rebuilt when a source file is
# newer, so a shortcut never silently launches a stale window frame.
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

FORCE_BUILD=0
if [ "${1:-}" = "--rebuild" ]; then
    FORCE_BUILD=1
    shift
fi

# Already running? Just stop (the control API on 127.0.0.1:8517 is a singleton).
if pgrep -x llamaherd >/dev/null 2>&1; then
    echo "llamaherd is already running (pid $(pgrep -x llamaherd | head -1)); not starting a second window."
    exit 0
fi

# Sources that end up in the binary: Go code, the embedded frontend and the
# module files. Data directories and build output are pruned.
stale() {
    [ -x "$BIN" ] || return 0
    local newer
    newer="$(find "$DIR" \
        \( -path "$DIR/build" -o -path "$DIR/.git" -o -path "$DIR/configs" \
           -o -path "$DIR/library" -o -path "$DIR/models" \) -prune -o \
        -type f \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' \
           -o -path "$DIR/frontend/dist/*" \) \
        -newer "$BIN" -print -quit 2>/dev/null)"
    [ -n "$newer" ]
}

if [ "$FORCE_BUILD" = 1 ] || stale; then
    echo "llamaherd: building $BIN ..." | tee -a "$LOG"
    (
        cd "$DIR"
        CGO_ENABLED=1 go build -tags webkit2_41,desktop,production -o "$BIN" .
    ) >>"$LOG" 2>&1 || {
        echo "llamaherd: build failed, see $LOG"
        exit 1
    }
fi

exec "./$BIN" "$@"
