#!/usr/bin/env bash
# Regenerate rsrc_windows_amd64.syso: embeds the app icon + version-info strings
# into llamaherd.exe (shortcuts, taskbar and Explorer then show the llama icon).
# The .syso is committed, so normal builds need no extra tooling; re-run this
# after changing assets/llamaherd.png (python3 scripts/make-icon.py first) or
# to bump the file/product version below (the runtime version still comes from
# the -ldflags stamp).
set -euo pipefail
cd "$(dirname "$0")/.."
go run github.com/tc-hib/go-winres@v0.3.3 simply \
  --arch amd64 \
  --manifest none \
  --product-version "0.3.0" \
  --file-version "0.3.0" \
  --icon assets/llamaherd.png \
  --product-name "LlamaHerd" \
  --file-description "LlamaHerd - llama.cpp control panel" \
  --original-filename "llamaherd.exe" \
  --copyright "MIT License"
ls -la rsrc_windows_amd64.syso
