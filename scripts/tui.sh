#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"

[[ $EUID -eq 0 ]] || die "run as root (sudo ./scripts/tui.sh)"
require_uefi

FLAKE_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
cd "$FLAKE_ROOT"

[[ -f $SCRIPT_DIR/install.sh ]] || die "install.sh missing"
chmod +x "$SCRIPT_DIR/install.sh"

export NIX_CONFIG="${NIX_CONFIG-}
experimental-features = nix-command flakes
http2 = false
download-attempts = 5
connect-timeout = 10
"

case ${TERM:-} in
  linux | dumb | "" ) export TERM=xterm-256color ;;
esac

fallback_plain() {
  info "${1:-gum unavailable} — using plain install.sh"
  exec "$SCRIPT_DIR/install.sh"
}

[[ -t 0 && -t 1 ]] || fallback_plain "no TTY"

GUM_BIN=$(type -P gum 2>/dev/null || true)
if [[ -z ${GUM_BIN:-} ]]; then
  command -v nix >/dev/null || fallback_plain "need gum or nix"
  info "fetching gum via nix…"
  GUM_BIN=$(nix build --no-link --print-out-paths --inputs-from "path:${FLAKE_ROOT}" nixpkgs#gum)/bin/gum
fi
[[ -x ${GUM_BIN:-} ]] || fallback_plain "gum binary missing"

run_gum() { "$GUM_BIN" "$@"; }

if ! run_gum style "ok" >/dev/null 2>&1; then
  fallback_plain "gum failed on this terminal"
fi

run_gum style --border rounded --padding "0 2" --border-foreground 212 \
  "EphemeralOS install" "LUKS + btrfs + impermanence"

if [[ -z ${DISK:-} ]]; then
  mapfile -t disk_paths < <(lsblk -dpn -o NAME,TYPE | awk '$2 == "disk" { print $1 }')
  [[ ${#disk_paths[@]} -gt 0 ]] || die "no disks found (lsblk -dpn -o NAME,TYPE)"

  choices=()
  for path in "${disk_paths[@]}"; do
    meta=$(lsblk -dn -o SIZE,TRAN,MODEL "$path" 2>/dev/null | sed 's/[[:space:]]\+/ /g')
    choices+=("$path  $meta")
  done

  picked=$(run_gum choose --header "Whole disk to erase:" "${choices[@]}") || die "aborted"
  DISK=$(awk '{print $1}' <<<"$picked")
fi
validate_disk "$DISK"

if [[ -z ${INSTALL_DESKTOP:-} ]]; then
  desk=$(run_gum choose --header "Desktop:" \
    "plasma  — KDE Plasma 6" \
    "gnome   — GNOME" \
    "niri    — niri + Noctalia shell") || die "aborted"
  INSTALL_DESKTOP=$(awk '{print $1}' <<<"$desk")
fi
validate_desktop "$INSTALL_DESKTOP"

if [[ -z ${NVIDIA_DRIVER:-} ]]; then
  if gpu=$(nvidia_gpu); then
    nv=$(run_gum choose --header "Found $gpu. Install NVIDIA drivers?" \
      "open    — RTX 20 / GTX 16 series and newer" \
      "legacy  — GTX 900 / GTX 10 series" \
      "none    — keep the open-source nouveau driver") || die "aborted"
    NVIDIA_DRIVER=$(awk '{print $1}' <<<"$nv")
  else
    NVIDIA_DRIVER=none
  fi
fi
validate_nvidia "$NVIDIA_DRIVER"

if [[ -z ${KEYBOARD_LAYOUT:-} ]]; then
  KEYBOARD_LAYOUT=$(run_gum input --header "Keyboard layout" --value "us" --placeholder "us, de, fr, us,de")
  KEYBOARD_VARIANT=$(run_gum input --header "Keyboard variant (empty for the default)" --value "${KEYBOARD_VARIANT:-}" --placeholder "nodeadkeys")
fi
KEYBOARD_VARIANT=${KEYBOARD_VARIANT:-}
validate_keyboard "$KEYBOARD_LAYOUT" "$KEYBOARD_VARIANT"
warn_luks_layout "$KEYBOARD_LAYOUT"
if [[ -z ${LOCALE:-} ]]; then
  LOCALE=$(run_gum input --header "Language" --value "en_US.UTF-8" --placeholder "de_DE.UTF-8")
fi
validate_locale "$LOCALE"

if [[ -z ${EXTRAS:-} ]]; then
  EXTRAS=false
  run_gum confirm --default=false "Install extra apps and tools? Steam, VS Code, Brave, VLC, atuin, comma and more." && EXTRAS=true
fi

if [[ -z ${FINGERPRINT:-} ]]; then
  FINGERPRINT=false
  if reader=$(fingerprint_reader); then
    run_gum confirm "Fingerprint reader found: $reader. Use it for login and sudo?" && FINGERPRINT=true
  fi
fi

INSTALL_HOSTNAME=$(run_gum input --header "Hostname" --value "${INSTALL_HOSTNAME:-nixos}" --placeholder "nixos")
INSTALL_USERNAME=$(run_gum input --header "Username" --value "${INSTALL_USERNAME:-nixuser}" --placeholder "nixuser")
SWAP_SIZE=$(run_gum input --header "Swap size" --value "${SWAP_SIZE:-4G}" --placeholder "4G")
TIMEZONE=$(run_gum input --header "Timezone" --value "${TIMEZONE:-UTC}" --placeholder "UTC")

validate_hostname "$INSTALL_HOSTNAME"
validate_username "$INSTALL_USERNAME"
validate_swap "$SWAP_SIZE"
validate_timezone "$TIMEZONE"

run_gum style --border rounded --padding "0 2" --border-foreground 196 \
  "WILL ERASE: $DISK" \
  "desktop=$INSTALL_DESKTOP  nvidia=$NVIDIA_DRIVER  host=$INSTALL_HOSTNAME  user=$INSTALL_USERNAME" \
  "swap=$SWAP_SIZE  tz=$TIMEZONE" \
  "keyboard=$KEYBOARD_LAYOUT${KEYBOARD_VARIANT:+ ($KEYBOARD_VARIANT)}  locale=$LOCALE  extras=$EXTRAS  fingerprint=$FINGERPRINT"

run_gum confirm "Erase $DISK and install?" || die "aborted"

export DISK
export INSTALL_HOSTNAME
export INSTALL_USERNAME
export INSTALL_DESKTOP
export NVIDIA_DRIVER
export KEYBOARD_LAYOUT
export KEYBOARD_VARIANT
export LOCALE
export FINGERPRINT
export EXTRAS
export SWAP_SIZE
export TIMEZONE
export FLAKE_ATTR="${FLAKE_ATTR:-nixos}"
export CONFIRM=YES

exec "$SCRIPT_DIR/install.sh"
