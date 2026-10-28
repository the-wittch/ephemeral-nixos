# shellcheck shell=bash

die() { echo "error: $*" >&2; exit 1; }
info() { echo "==> $*"; }

validate_hostname() {
  [[ $1 =~ ^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$ ]] \
    || die "invalid hostname: $1 (letters/digits/hyphen)"
}

validate_username() {
  [[ $1 =~ ^[a-z_][a-z0-9_-]*$ && ${#1} -le 32 ]] \
    || die "invalid username: $1 (use [a-z_][a-z0-9_-]*, max 32)"
}

validate_swap() {
  [[ $1 =~ ^[0-9]+[KMGT]?$ ]] || die "invalid swap size: $1 (e.g. 4G, 8192M)"
}

validate_timezone() {
  [[ $1 =~ ^[A-Za-z0-9_/+-]+$ ]] || die "invalid timezone: $1"
}

validate_desktop() {
  case $1 in
    plasma | gnome | niri) ;;
    *) die "desktop must be plasma, gnome, or niri (got: $1)" ;;
  esac
}

validate_nvidia() {
  case $1 in
    none | open | legacy) ;;
    *) die "NVIDIA_DRIVER must be none, open, or legacy (got: $1)" ;;
  esac
}

# device names go into prompts; drop escape sequences
printable() {
  tr -cd '[:print:]\n'
}

# vendor 10de, pci class 03 (display)
nvidia_gpu() {
  local d
  for d in /sys/bus/pci/devices/*; do
    [[ -r $d/vendor && -r $d/class ]] || continue
    if [[ $(<"$d/vendor") == 0x10de && $(<"$d/class") == 0x03* ]]; then
      if command -v lspci >/dev/null; then
        lspci -s "${d##*/}" | cut -d' ' -f2- | sed 's/^[^:]*: //' | printable
      else
        echo "NVIDIA GPU"
      fi
      return 0
    fi
  done
  return 1
}

validate_keyboard() {
  [[ $1 =~ ^[a-z]{2,3}(,[a-z]{2,3})*$ ]] || die "invalid keyboard layout: $1 (e.g. us, de, us,de)"
  [[ $2 =~ ^[a-z0-9_-]*(,[a-z0-9_-]*)*$ ]] || die "invalid keyboard variant: $2 (e.g. nodeadkeys)"
}

validate_locale() {
  [[ $1 =~ ^[A-Za-z_]+\.UTF-8$ ]] || die "invalid locale: $1 (e.g. en_US.UTF-8)"
}

warn_luks_layout() {
  [[ $1 == us ]] && return 0
  echo
  echo "  Heads up: you type the LUKS passphrase in a moment with this live system's"
  echo "  keyboard layout, but at boot with '$1'. Switch the live session to '$1' first,"
  echo "  or use a passphrase that types the same on both layouts."
  echo
}

has_battery() {
  local b
  for b in /sys/class/power_supply/BAT*; do
    [[ -e $b ]] && return 0
  done
  return 1
}

# usb vendors that make fingerprint readers: goodix, synaptics, validity, elan, egis, focaltech, authentec, upek, fpc
fingerprint_reader() {
  local d v
  for d in /sys/bus/usb/devices/*; do
    [[ -r $d/idVendor ]] || continue
    v=$(<"$d/idVendor")
    case $v in
      27c6 | 06cb | 138a | 04f3 | 1c7a | 2808 | 08ff | 147e | 10a5)
        printf '%s %s (%s)\n' "$(cat "$d/manufacturer" 2>/dev/null)" "$(cat "$d/product" 2>/dev/null)" "$v" | printable
        return 0
        ;;
    esac
  done
  return 1
}

validate_chezmoi() {
  local url=$1
  [[ -z $url ]] && return 0
  case $url in
    https://* | ssh://* | git@*) ;;
    *) die "chezmoi repo must start with https://, ssh://, or git@" ;;
  esac
  [[ $url != *$'\n'* && $url != *$'\r'* && $url != *" "* ]] \
    || die "chezmoi repo must be a single token"
}

validate_chezmoi_rev() {
  [[ $1 =~ ^[0-9a-fA-F]{40}$ ]] \
    || die "chezmoi commit must be the full 40-character hash"
}

validate_disk() {
  local d=$1
  [[ -b $d ]] || die "not a block device: $d"
  if [[ $d =~ p[0-9]+$ ]]; then
    die "looks like a partition: $d (use the whole disk, e.g. /dev/nvme0n1)"
  fi
  if [[ $d =~ [0-9]+$ && ! $d =~ nvme[0-9]+n[0-9]+$ && ! $d =~ mmcblk[0-9]+$ && ! $d =~ loop[0-9]+$ ]]; then
    die "looks like a partition: $d (use the whole disk, e.g. /dev/vda)"
  fi
}

validate_flake_inputs() {
  case $1 in
    lock | update) ;;
    *) die "FLAKE_INPUTS must be lock or update (got: $1)" ;;
  esac
}

validate_flake_attr() {
  [[ $1 =~ ^[a-zA-Z][a-zA-Z0-9_-]*$ ]] \
    || die "invalid FLAKE_ATTR: $1"
}

require_uefi() {
  [[ -d /sys/firmware/efi ]] \
    || die "UEFI required (this flake uses systemd-boot). Enable OVMF/UEFI in the VM/firmware."
}
