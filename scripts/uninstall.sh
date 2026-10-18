#!/usr/bin/env bash
# Convenience wrapper: uninstall a LlamaHerd install created by install.sh.
#
#   scripts/uninstall.sh            # remove the app, keep data + config
#   scripts/uninstall.sh --purge    # also remove data + config
#
# All arguments are passed through to install.sh --uninstall.
set -euo pipefail
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "$DIR/install.sh" --uninstall "$@"
