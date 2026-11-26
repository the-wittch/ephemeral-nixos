# shellcheck shell=bash

die() { echo "error: $*" >&2; exit 1; }
info() { echo "==> $*"; }

# scripts/nix.conf, which the screens read too; bootstrap.sh and bootstrap-tui.sh keep a copy, since
# they run before there's a clone to read it from
set_nix_config() {
  local conf
  conf=$(<"$SCRIPT_DIR/nix.conf")
  export NIX_CONFIG="${NIX_CONFIG-}
$conf"
}

validate_bool() {
  [[ $2 == true || $2 == false ]] || die "$1 must be true or false (got: $2)"
}

validate_apps() {
  [[ $1 =~ ^[A-Za-z0-9._+\ -]*$ ]] || die "APPS has to be app names separated by commas or spaces (got: $1)"
}

# device names go into prompts; drop escape sequences
printable() {
  tr -cd '[:print:]\n'
}

# vendor 10de, pci class 03 (display); the screens call this too. lspci's name, without the
# revision, and only the model in brackets when it gives one
nvidia_gpu() {
  local d name
  for d in /sys/bus/pci/devices/*; do
    [[ -r $d/vendor && -r $d/class ]] || continue
    if [[ $(<"$d/vendor") == 0x10de && $(<"$d/class") == 0x03* ]]; then
      name=$(lspci -s "${d##*/}" 2>/dev/null | sed -E 's/^[0-9a-f:.]+ [^:]*: //; s/ *\(rev [0-9a-f]+\)$//; s/.*\[([^]]*)\]$/\1/' || true)
      printf '%s\n' "${name:-an NVIDIA card}" | printable
      return 0
    fi
  done
  return 1
}

warn_luks_layout() {
  [[ $1 == us ]] && return 0
  echo
  echo "  Heads up: you type the LUKS passphrase in a moment with this live system's"
  echo "  keyboard layout, but at boot with '$1'. Switch the live session to '$1' first,"
  echo "  or use a passphrase that types the same on both layouts."
  echo
}

# Whether the passphrase on stdin is 12 or more of the characters in $1, the ones the controller's
# buttons type at boot. It comes on stdin so it's never in an argument.
handheld_passphrase_ok() {
  local chars=$1 p=
  [[ $chars =~ ^[a-z0-9]+$ ]] || return 1
  IFS= read -r p || [[ -n $p ]] || return 1
  [[ ${#p} -ge 12 && $p =~ ^[$chars]+$ ]]
}

dmi_vendor() { cat /sys/class/dmi/id/sys_vendor 2>/dev/null || true; }

has_battery() {
  local b
  for b in /sys/class/power_supply/BAT*; do
    [[ -e $b ]] && return 0
  done
  return 1
}

# the makers in scripts/fingerprint-vendors; the screens call this too
fingerprint_reader() {
  local d v vendors name
  vendors=$(awk '$1 !~ /^#/ { print $1 }' "$SCRIPT_DIR/fingerprint-vendors")
  for d in /sys/bus/usb/devices/*; do
    [[ -r $d/idVendor ]] || continue
    v=$(<"$d/idVendor")
    if grep -qxF -- "$v" <<<"$vendors"; then
      name=$(printf '%s %s' "$(cat "$d/manufacturer" 2>/dev/null)" "$(cat "$d/product" 2>/dev/null)" | sed 's/^ *//; s/ *$//')
      printf '%s (%s)\n' "${name:-a fingerprint reader}" "$v" | printable
      return 0
    fi
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

# These read lsblk into a variable before grep: piped into grep -q, lsblk can be cut off when
# grep stops at the first match, and under pipefail a match then reads as none.
holds_installer_image() {
  local fstypes
  fstypes=$(lsblk -nro FSTYPE -- "$1" 2>/dev/null) || return 1
  grep -qx iso9660 <<<"$fstypes"
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
  local mounts types
  mounts=$(lsblk -nro MOUNTPOINTS "$1" 2>/dev/null) || true
  grep -q . <<<"$mounts" && return 0
  types=$(lsblk -nro TYPE "$1" 2>/dev/null) || true
  grep -qE '^(crypt|lvm|raid)' <<<"$types"
}

# A name like /dev/sdb goes to whichever disk shows up second, so a disk plugged in or out
# between choosing and erasing can change what it points at. These don't change; the physical
# path tells apart two of the same model that report no serial. The screens run this to show
# the disk, and install.sh runs it again right before erasing and compares the two as text.
disk_identity() {
  printf '%s ID_PATH="%s"\n' "$(lsblk -dbP -o SERIAL,WWN,SIZE,MODEL -- "$1")" \
    "$(udevadm info --query=property --property=ID_PATH --value --name="$1" 2>/dev/null || true)"
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

# the firmware only takes new Secure Boot keys in Setup Mode
setup_mode() {
  [[ $(od -An -tu1 -j4 -N1 /sys/firmware/efi/efivars/SetupMode-8be4df61-93ca-11d2-aa0d-00e098032b8c 2>/dev/null | tr -d ' ') == 1 ]]
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
    grep -qxF -- "$dev" <<<"$(lsblk -nlpo NAME "$disk")" ||
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
