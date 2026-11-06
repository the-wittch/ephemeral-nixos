#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"

[[ $EUID -eq 0 ]] || die "run as root (sudo ./scripts/tui.sh)"
require_uefi

FLAKE_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
cd "$FLAKE_ROOT"

[[ -x $SCRIPT_DIR/install.sh ]] || die "install.sh missing or not executable"

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

# type to narrow the list; text that matches nothing is taken as typed
pick() {
  local line
  line=$(run_gum filter --header "$1" --placeholder "type to search" --height 15 --no-strict) || die "aborted"
  printf '%s\n' "${line%% *}"
}

default_first() {
  local all
  all=$(cat)
  grep -E "^$1( |$)" <<<"$all" || true
  grep -vE "^$1( |$)" <<<"$all" || true
}

LISTS=${EPHEMERAL_LISTS:-/etc/ephemeral-nixos-lists}

xkb_rules() {
  local f
  for f in "$LISTS/base.lst" /etc/X11/xkb/rules/base.lst /run/current-system/sw/share/X11/xkb/rules/base.lst; do
    [[ -r $f ]] && { echo "$f"; return 0; }
  done
  return 1
}

# layouts as "code  description"; variants of one layout the same way
xkb_list() {
  awk -v section="! $2" -v layout="${3:-}:" '
    $0 == section { on = 1; next }
    /^!/ { on = 0 }
    !on || NF < 2 { next }
    section == "! variant" && $2 != layout { next }
    {
      code = $1
      $1 = ""
      if (section == "! variant") $2 = ""
      sub(/^ +/, "")
      printf "%-16s %s\n", code, $0
    }' "$1"
}

locales() {
  local f
  for f in "$LISTS/locales" /run/current-system/sw/share/i18n/SUPPORTED; do
    if [[ -r $f ]]; then
      sed -n 's#^\([A-Za-z_]*\.UTF-8\)/UTF-8.*#\1#p' "$f"
      return 0
    fi
  done
  return 1
}

# the config's app catalog and a jq to read it; the ISO has no jq
load_catalog() {
  local out
  out=$(run_gum spin --show-stdout --title "Loading the app list (the first time takes a few minutes)..." -- \
    nix build --no-link --print-out-paths --inputs-from "path:$FLAKE_ROOT" \
    "path:$FLAKE_ROOT#app-catalog" nixpkgs#jq) || return 1
  CATALOG=$(grep -m1 -- '-app-catalog.json$' <<<"$out") || return 1
  JQ=$(grep -m1 -- '-bin$' <<<"$out")/bin/jq
  [[ -r $CATALOG && -x $JQ ]]
}

pick_apps() {
  local id row out
  local -A ids=()
  local -a rows=()
  while IFS=$'\t' read -r id row; do
    ids[$row]=$id
    rows+=("$row")
  done < <("$JQ" -r 'to_entries | sort_by(.value.category, (.value.label | ascii_downcase)) | .[] |
    "\(.key)\t\(.value.category): \(.value.label) — \(.value.description)"
    + (if .value.unfree then " · unfree" else "" end)
    + (if .value.flatpak then " · flatpak" else "" end)' "$CATALOG")

  out=$(printf '%s\n' "${rows[@]}" |
    run_gum filter --no-limit --height 20 --header "Apps: tab to tick, enter when done" --placeholder "type to search") || return 0
  while IFS= read -r row; do
    [[ -z $row ]] || printf '%s ' "${ids[$row]}"
  done <<<"$out"
}

if ! run_gum style "ok" >/dev/null 2>&1; then
  fallback_plain "gum failed on this terminal"
fi

run_gum style --border rounded --padding "0 2" --border-foreground 212 \
  "Ephemeral Nix install" "LUKS + btrfs + impermanence"

if [[ -z ${DISK:-} ]]; then
  mapfile -t disk_paths < <(lsblk -dpn -o NAME,TYPE | awk '$2 == "disk" { print $1 }')
  [[ ${#disk_paths[@]} -gt 0 ]] || die "no disks found (lsblk -dpn -o NAME,TYPE)"

  choices=()
  hidden=0
  for path in "${disk_paths[@]}"; do
    if disk_in_use "$path" && [[ ${FORCE_DISK:-} != 1 ]]; then
      hidden=$((hidden + 1))
      continue
    fi
    meta=$(lsblk -dn -o SIZE,TRAN,MODEL "$path" 2>/dev/null | sed 's/[[:space:]]\+/ /g')
    choices+=("$path  $meta")
  done
  ((hidden == 0)) || info "$hidden disk(s) with mounted filesystems left out, such as the one you booted from (FORCE_DISK=1 shows them)"
  [[ ${#choices[@]} -gt 0 ]] || die "every disk is in use; unmount the one to install on first"

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
  if rules=$(xkb_rules); then
    KEYBOARD_LAYOUT=$(xkb_list "$rules" layout | default_first us | pick "Keyboard layout (us,de for more than one)")
    variants=$(xkb_list "$rules" variant "$KEYBOARD_LAYOUT")
    KEYBOARD_VARIANT=
    if [[ -n $variants ]]; then
      KEYBOARD_VARIANT=$(printf '%s\n%s\n' "default" "$variants" | pick "Variant of $KEYBOARD_LAYOUT")
      [[ $KEYBOARD_VARIANT != default ]] || KEYBOARD_VARIANT=
    fi
  else
    KEYBOARD_LAYOUT=$(run_gum input --header "Keyboard layout" --value "us" --placeholder "us, de, fr, us,de")
    KEYBOARD_VARIANT=$(run_gum input --header "Keyboard variant (empty for the default)" --value "${KEYBOARD_VARIANT:-}" --placeholder "nodeadkeys")
  fi
fi
KEYBOARD_VARIANT=${KEYBOARD_VARIANT:-}
validate_keyboard "$KEYBOARD_LAYOUT" "$KEYBOARD_VARIANT"
warn_luks_layout "$KEYBOARD_LAYOUT"
if [[ -z ${LOCALE:-} ]]; then
  if all_locales=$(locales); then
    LOCALE=$(default_first "en_US\.UTF-8" <<<"$all_locales" | pick "Language")
  else
    LOCALE=$(run_gum input --header "Language" --value "en_US.UTF-8" --placeholder "de_DE.UTF-8")
  fi
fi
validate_locale "$LOCALE"

if [[ -z ${PROFILE:-} ]]; then
  profile_pick=$(run_gum choose --header "What to install" \
    "full   — everything, with the options you pick next" \
    "small  — a plain desktop now; the rest later with ephemeral-setup") || die "aborted"
  case $profile_pick in
    full*) PROFILE=full ;;
    *) PROFILE=minimal ;;
  esac
fi
validate_profile "$PROFILE"

if [[ $PROFILE == full ]]; then
  if [[ -z ${EXTRAS:-} ]]; then
    EXTRAS=false
    run_gum confirm --default=false "Install extra apps and tools? Steam, VS Code, Brave, VLC, atuin, comma and more." && EXTRAS=true
  fi
  if [[ -z ${OPENSNITCH:-} ]]; then
    OPENSNITCH=false
    run_gum confirm "OpenSnitch: ask before programs connect out?" && OPENSNITCH=true
  fi
  if [[ -z ${PASS_SSH_AGENT:-} ]]; then
    PASS_SSH_AGENT=false
    run_gum confirm --default=false "Proton Pass SSH agent at login?" && PASS_SSH_AGENT=true
  fi
  if [[ $PASS_SSH_AGENT == true && -z ${PASS_SSH_VAULT:-} ]]; then
    PASS_SSH_VAULT=$(run_gum input --header "Proton Pass vault with your SSH keys" --value "Personal")
  fi
  if [[ -z ${CONTROLD_RESOLVER+x} ]]; then
    CONTROLD_RESOLVER=$(run_gum input --header "Control D resolver id (empty for none)" --placeholder "abcd1234")
  fi
  if [[ -z ${APPS+x} ]]; then
    APPS=
    if run_gum confirm --default=false "Pick more apps now? ephemeral-apps does the same after the install."; then
      if load_catalog; then
        APPS=$(pick_apps)
        for id in $APPS; do
          # shellcheck disable=SC2016 # $id is jq's
          warn=$("$JQ" -r --arg id "$id" '.[$id] | if .warn then "\(.label): \(.warn)" else empty end' "$CATALOG")
          [[ -z $warn ]] || run_gum style --foreground 214 "$warn"
        done
        [[ -z $APPS ]] || run_gum confirm "Add ${APPS% }?" || APPS=
      else
        info "couldn't load the app list; pick apps after the install with ephemeral-apps"
      fi
    fi
  fi
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
if [[ -z ${TIMEZONE:-} ]] && zones=$(timedatectl list-timezones 2>/dev/null) && [[ -n $zones ]]; then
  TIMEZONE=$(default_first UTC <<<"$zones" | pick "Timezone")
else
  TIMEZONE=$(run_gum input --header "Timezone" --value "${TIMEZONE:-UTC}" --placeholder "UTC")
fi

validate_hostname "$INSTALL_HOSTNAME"
validate_username "$INSTALL_USERNAME"
validate_swap "$SWAP_SIZE"
validate_timezone "$TIMEZONE"

summary=(
  "WILL ERASE: $DISK"
  "desktop=$INSTALL_DESKTOP  nvidia=$NVIDIA_DRIVER  host=$INSTALL_HOSTNAME  user=$INSTALL_USERNAME"
  "swap=$SWAP_SIZE  tz=$TIMEZONE"
  "keyboard=$KEYBOARD_LAYOUT${KEYBOARD_VARIANT:+ ($KEYBOARD_VARIANT)}  locale=$LOCALE  fingerprint=$FINGERPRINT"
  "profile=$PROFILE  extras=${EXTRAS:-false}  opensnitch=${OPENSNITCH:-true}  pass-ssh-agent=${PASS_SSH_AGENT:-false}${CONTROLD_RESOLVER:+  controld=on}"
)
[[ -z ${APPS// /} ]] || summary+=("apps=${APPS% }")
run_gum style --border rounded --padding "0 2" --border-foreground 196 "${summary[@]}"

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
export PROFILE
export OPENSNITCH
export PASS_SSH_AGENT
export PASS_SSH_VAULT
export CONTROLD_RESOLVER
export APPS
export SWAP_SIZE
export TIMEZONE
export FLAKE_ATTR="${FLAKE_ATTR:-nixos}"
export CONFIRM=YES

exec "$SCRIPT_DIR/install.sh"
