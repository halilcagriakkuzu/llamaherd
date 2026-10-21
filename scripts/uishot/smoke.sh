#!/usr/bin/env bash
# Headless UI smoke test: boot the real frontend with the mock backend and
# assert a few visible states. No npm runtime needed; only a Chromium binary.
#
#   scripts/uishot/smoke.sh            # default: running + failed scenarios
#   CHROME=/usr/bin/google-chrome scripts/uishot/smoke.sh
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"

find_chrome() {
  if [ -n "${CHROME:-}" ] && [ -x "$CHROME" ]; then echo "$CHROME"; return; fi
  for c in google-chrome google-chrome-stable chromium chromium-browser; do
    if command -v "$c" >/dev/null 2>&1; then command -v "$c"; return; fi
  done
  local p
  p="$(ls -d "$HOME"/.cache/ms-playwright/chromium-*/chrome-linux*/chrome 2>/dev/null | tail -1)"
  if [ -n "$p" ] && [ -x "$p" ]; then echo "$p"; return; fi
  return 1
}

CHROME_BIN="$(find_chrome || true)"
if [ -z "$CHROME_BIN" ]; then
  echo "no Chromium found; set CHROME=/path/to/chrome (or run 'npx playwright install chromium')" >&2
  exit 1
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
cp -r "$root/frontend/dist/." "$tmp/"
cp "$here/mock.js" "$tmp/mock.js"
# Load the mock before the app; it defines window.go from the URL query.
python3 - "$tmp/index.html" <<'PY'
import sys
p = sys.argv[1]
s = open(p).read()
s = s.replace('<script src="main.js"></script>',
              '<script src="mock.js"></script><script src="main.js"></script>')
open(p, "w").write(s)
PY

run_case() {
  local name="$1" query="$2" want="$3"
  local dom
  dom="$("$CHROME_BIN" --headless=new --no-sandbox --disable-gpu --hide-scrollbars \
    --force-color-profile=srgb --virtual-time-budget=6000 --dump-dom \
    "file://$tmp/index.html?$query" 2>/dev/null || true)"
  if printf '%s' "$dom" | grep -q "$want"; then
    echo "ok   - $name"
    return 0
  fi
  echo "FAIL - $name" >&2
  printf '%s\n' "$dom" | grep -o 'SELFTEST FAIL[^<]*' >&2 || true
  printf '%s\n' "$dom" | grep -o 'SELFTEST PASS[^<]*' >&2 || true
  return 1
}

fail=0
run_case "running: models, benchmark table, import dialog, download queue" \
  "selftest=main&status=running&dl=1" "SELFTEST PASS" || fail=1
run_case "failed: not-responding banner and actions" \
  "selftest=failed&status=failed" "SELFTEST PASS" || fail=1

if [ "$fail" -ne 0 ]; then
  echo "UI smoke test failed" >&2
  exit 1
fi
echo "UI smoke test passed ($CHROME_BIN)"
