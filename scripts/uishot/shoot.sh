#!/usr/bin/env bash
# Render frontend/dist with a mocked backend and write PNG screenshots.
#   scripts/uishot/shoot.sh OUTDIR [theme: light|dark]
# Needs a Chromium (Playwright's cache is auto-detected, or set CHROME=...).
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
out="${1:?usage: shoot.sh OUTDIR [light|dark]}"; theme="${2:-light}"
CHROME="${CHROME:-$(ls -d "$HOME"/.cache/ms-playwright/chromium-*/chrome-linux*/chrome 2>/dev/null | tail -1)}"
[ -x "$CHROME" ] || { echo "no chromium found (set CHROME=)"; exit 1; }
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
cp -r "$root/frontend/dist/." "$tmp/"
cp "$here/mock.js" "$tmp/mock.js"
sed -i 's#<script src="main.js"></script>#<script src="mock.js"></script><script src="main.js"></script>#' "$tmp/index.html"
mkdir -p "$out"
shot() { # name query [WxH]
  local size="${3:-1240,820}"
  "$CHROME" --headless=new --no-sandbox --disable-gpu --hide-scrollbars \
    --force-color-profile=srgb --force-prefers-color-scheme="$theme" \
    --window-size="$size" --virtual-time-budget=2500 \
    --screenshot="$out/$1-$theme.png" "file://$tmp/index.html?theme=$theme&$2" >/dev/null 2>&1
}
shot main "status=running&tab=basic"
shot advanced "status=running&tab=advanced&vision=1"
shot llama "status=running&tab=llama"
shot stopped "status=stopped&tab=basic&dirty=1&warn=1"
shot validation "status=running&tab=basic&issues=1"
shot download "status=stopped&modal=download"
shot download-catalog "status=stopped&modal=catalog" 1240,900
shot download-bg "status=running&dl=1"
shot download-queue "status=running&dl=1&modal=download"
shot more-menu "status=running&menu=more"
shot notifications "status=running&notifications=1&notif=1"
shot download-repo "status=stopped&modal=repo" 1240,980
shot menu-model "status=running&menu=model"
shot menu-folder "status=running&view=all&menu=folder"
shot dialog-rename "status=running&dialog=rename"
shot dialog-move "status=running&dialog=move"
shot dialog-details "status=running&dialog=details"
shot firstrun "status=stopped&firstrun=1"
shot failed "status=failed&tab=basic"
shot import "status=stopped&modal=import"
ls "$out"
