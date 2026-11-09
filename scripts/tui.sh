#!/usr/bin/env bash
# Starts the installer's screens (installer/) from a clone of this repo, as bootstrap-tui.sh does.
# Without a terminal, or if they can't be built, it falls back to the plain prompts in install.sh.

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"

[[ $EUID -eq 0 ]] || die "run as root (sudo ./scripts/tui.sh)"
require_uefi

FLAKE_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
[[ -x $SCRIPT_DIR/install.sh ]] || die "install.sh missing or not executable"

export NIX_CONFIG="${NIX_CONFIG-}
experimental-features = nix-command flakes
http2 = false
download-attempts = 5
connect-timeout = 10
"

fallback_plain() {
  info "$1, so using the plain prompts"
  exec "$SCRIPT_DIR/install.sh"
}

[[ -t 0 && -t 1 ]] || fallback_plain "no terminal"

info "building the installer (the first time takes a few minutes)"
installer=$(nix build --no-link --print-out-paths "path:$FLAKE_ROOT#installer") \
  || fallback_plain "couldn't build the installer"

exec "$installer/bin/ephemeral-install" --scripts "$SCRIPT_DIR" "$@"
