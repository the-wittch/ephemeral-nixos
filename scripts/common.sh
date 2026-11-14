# shellcheck shell=bash

die() { echo "error: $*" >&2; exit 1; }
info() { echo "==> $*"; }

validate_hostname() {
  [[ $1 =~ ^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$ ]] \
    || die "the hostname can only have letters, digits and hyphens, and can't start or end with a hyphen, so not $1. Try something like my-laptop"
}

validate_username() {
  [[ $1 =~ ^[a-z_][a-z0-9_-]*$ && ${#1} -le 32 ]] \
    || die "the username has to be lowercase letters, digits, - and _, start with a letter, and be at most 32 long, so not $1. Try something like alice"
}

validate_swap() {
  [[ $1 =~ ^[0-9]+[KMGT]?$ ]] || die "swap is a size like 4G or 8192M, not $1"
}

validate_timezone() {
  [[ $1 =~ ^[A-Za-z0-9_/+-]+$ ]] || die "$1 isn't a time zone; they look like Europe/Berlin or America/New_York"
}

validate_desktop() {
  case $1 in
    plasma | gnome | niri | umbriel) ;;
    *) die "desktop must be plasma, gnome, niri or umbriel (got: $1)" ;;
  esac
}

validate_profile() {
  case $1 in
    full | minimal) ;;
    *) die "PROFILE must be full or minimal (got: $1)" ;;
  esac
}

validate_bool() {
  [[ $2 == true || $2 == false ]] || die "$1 must be true or false (got: $2)"
}

validate_vault() {
  case $1 in
    "" | *[[:cntrl:]]* | *'"'* | *'$'* | *'`'* | *\\*)
      die "a Proton Pass vault name can't have quotes, \$, backticks or backslashes, since it ends up in the login script: $1"
      ;;
  esac
}

validate_controld() {
  [[ -z $1 || $1 =~ ^[A-Za-z0-9_-]+$ ]] || die "a Control D resolver id is letters, digits, - and _, as in your Control D dashboard, so not $1"
}

validate_dns() {
  case $1 in
    none | cloudflare | quad9 | adguard | custom) ;;
    *) die "ENCRYPTED_DNS is none, cloudflare, quad9, adguard or custom, not $1" ;;
  esac
}

# address#name, the name being what the server's certificate is checked against
validate_custom_dns() {
  local s list
  read -ra list <<<"$1"
  ((${#list[@]})) || die "ENCRYPTED_DNS=custom needs servers in CUSTOM_DNS, as address#name, e.g. 194.242.2.2#dns.mullvad.net"
  for s in "${list[@]}"; do
    [[ $s =~ ^[0-9A-Fa-f.:]+#[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?$ ]] ||
      die "$s in CUSTOM_DNS isn't address#name, such as 194.242.2.2#dns.mullvad.net"
  done
}

validate_apps() {
  [[ $1 =~ ^[A-Za-z0-9._+\ -]*$ ]] || die "APPS has to be app names separated by commas or spaces (got: $1)"
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
  [[ $1 =~ ^[a-z]{2,3}(,[a-z]{2,3})*$ ]] || die "a keyboard layout is a short code like us or de, or two like us,de, so not $1"
  [[ $2 =~ ^[a-z0-9_-]*(,[a-z0-9_-]*)*$ ]] || die "a keyboard variant is a name like nodeadkeys, or empty for the usual one, so not $2"
}

validate_locale() {
  [[ $1 =~ ^[A-Za-z_]+\.UTF-8$ ]] || die "a language is a name like en_US.UTF-8 or de_DE.UTF-8, so not $1"
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

validate_disk() {
  local d=$1 type
  [[ -b $d ]] || die "$d isn't a disk on this machine. The disks are:
$(lsblk -dpn -o NAME,SIZE,MODEL -e 7,11 2>/dev/null | grep -v '^/dev/zram' | sed 's/^/  /')"
  type=$(lsblk -ndo TYPE "$d" 2>/dev/null) || die "lsblk can't read $d"
  case $type in
    disk | loop) ;;
    part) die "$d is a partition; use the whole disk, e.g. /dev/nvme0n1" ;;
    *) die "$d is a $type device, not a whole disk" ;;
  esac
  # booted with copytoram, the stick isn't mounted any more
  if holds_installer_image "$d" && [[ ${FORCE_DISK:-} != 1 ]]; then
    die "$d holds an installer image, likely the USB stick this runs from. Pick another disk, or set FORCE_DISK=1"
  fi
  if disk_in_use "$d" && [[ ${FORCE_DISK:-} != 1 ]]; then
    die "$d is in use (mounted, swap, or an open LUKS, LVM or RAID device), so it may be the disk you booted from. If it's the right disk, close what's open on it: sudo swapoff -a; sudo umount -R /mnt; sudo cryptsetup close <name>. Or set FORCE_DISK=1"
  fi
  local group
  if group=$(storage_member "$d") && [[ ${FORCE_DISK:-} != 1 ]]; then
    die "$d is part of $group. Erasing it removes all of that, on its other disks too, not only on $d. Pick another disk, or set FORCE_DISK=1 if that's what you want"
  fi
}

holds_installer_image() {
  lsblk -nro FSTYPE -- "$1" 2>/dev/null | grep -qx iso9660
}

# disko runs vgremove -f or zpool destroy -f on these, which also wipes the rest of the
# group or pool on other disks
storage_member() {
  local fstypes
  fstypes=$(lsblk -nro FSTYPE -- "$1" 2>/dev/null) || return 1
  if grep -qx LVM2_member <<<"$fstypes"; then
    echo "an LVM volume group"
  elif grep -qx zfs_member <<<"$fstypes"; then
    echo "a ZFS pool"
  elif grep -qE '^[a-z]+_raid_member$' <<<"$fstypes"; then
    echo "a RAID array"
  else
    return 1
  fi
}

disk_in_use() {
  lsblk -nro MOUNTPOINTS "$1" 2>/dev/null | grep -q . && return 0
  lsblk -nro TYPE "$1" 2>/dev/null | grep -qE '^(crypt|lvm|raid)'
}

# A name like /dev/sdb goes to whichever disk shows up second, so a disk plugged in or out
# between choosing and erasing can change what it points at. These don't change.
disk_identity() {
  lsblk -dbP -o SERIAL,WWN,SIZE,MODEL -- "$1"
}

# The new system finds its partitions by these labels, so another disk that has them, from an
# earlier install, can be the one it unlocks and boots.
validate_partlabels() {
  local disk others
  disk=$(realpath "$1")
  others=$(lsblk -nrpo PKNAME,PARTLABEL | awk -v disk="$disk" \
    '($2 == "disk-main-ESP" || $2 == "disk-main-luks") && $1 != disk { print $1 }' | sort -u | paste -sd ' ')
  [[ -z $others ]] || [[ ${ALLOW_SAME_PARTLABELS:-} == 1 ]] ||
    die "$others has partitions from an earlier Ephemeral Nix install, with the labels the new system boots by, so it could start the old one. Detach it while installing, or set ALLOW_SAME_PARTLABELS=1"
}

# room for the 1G ESP and at least 8G for the system besides the swap file
validate_swap_fits() {
  local disk=$1 swap size swap_bytes
  swap=$2
  size=$(lsblk -dnbo SIZE -- "$disk") || die "lsblk can't read the size of $disk"
  swap_bytes=$(numfmt --from=iec -- "$swap") || die "can't read the swap size $swap"
  ((size - swap_bytes >= 9 * 1024 ** 3)) ||
    die "a $swap swap file leaves too little of $disk ($(numfmt --to=iec -- "$size")) for the system; make it smaller"
}

validate_flake_attr() {
  [[ $1 =~ ^[a-zA-Z][a-zA-Z0-9_-]*$ ]] \
    || die "invalid FLAKE_ATTR: $1"
}

require_uefi() {
  [[ -d /sys/firmware/efi ]] \
    || die "this PC started in legacy BIOS mode, and Ephemeral Nix needs UEFI. Restart, open the firmware settings, and boot the USB stick in UEFI mode. In a VM, turn on UEFI (OVMF)"
}

# Undoes what an earlier attempt left mounted or open, as recorded in $STATE, and nothing else.
# Without it the disk check refuses the same disk, and another disk can't open its cryptroot.
teardown() {
  [[ -f $STATE ]] || return 0
  local disk dev
  disk=$(<"$STATE")
  info "closing what the last attempt left open on $disk"
  while read -r dev; do
    if [[ $dev == /mnt/* ]]; then
      swapoff "$dev"
    fi
  done < <(swapon --show=NAME --noheadings)
  if findmnt /mnt >/dev/null; then
    [[ $(findmnt -no SOURCE /mnt) == /dev/mapper/cryptroot* ]] ||
      die "/mnt has something on it that the installer didn't mount; unmount it first"
    umount -R /mnt
  fi
  if cryptsetup status cryptroot >/dev/null 2>&1; then
    dev=$(cryptsetup status cryptroot | awk '$1 == "device:" { print $2 }')
    lsblk -nlpo NAME "$disk" | grep -qxF -- "$dev" ||
      die "cryptroot is open on $dev, which isn't on $disk; close it yourself if that's safe: sudo cryptsetup close cryptroot"
    cryptsetup close cryptroot
  fi
  rm -f "$STATE"
}

# ::stage lines tell the screens which step is running, and errors say what state the disk is in
stage() {
  STAGE_NAME=$1
  echo "::stage $1"
}
on_error() {
  local status=$?
  case ${STAGE_NAME:-prepare} in
    prepare) echo "error: stopped before changing anything on ${DISK:-the disk}" >&2 ;;
    partition) echo "error: stopped while partitioning ${DISK:-the disk}, so it may be half formatted; trying again starts it over" >&2 ;;
    *) echo "error: stopped with ${DISK:-the disk} formatted and mounted at /mnt; trying again closes it and starts over" >&2 ;;
  esac
  echo "  the full output is in ${LOG:-/tmp/ephemeral-install.log}" >&2
  exit "$status"
}
