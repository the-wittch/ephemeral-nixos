# shellcheck shell=bash
# shellcheck disable=SC2016 # the fakes are written out as they are, expanded when they run
# Runs common.sh's checks and teardown against fake lsblk, findmnt, swapon and cryptsetup, so
# what they refuse and what they undo is tested without root or a disk. Called by
# modules/tests/scripts.nix with COMMON pointing at scripts/common.sh.
set -euo pipefail

work=$(mktemp -d)
fakes=$work/bin
mkdir -p "$fakes"
export CALLS=$work/calls
export PATH=$fakes:$PATH

fake() {
  printf '#!%s\n%s\n' "$(command -v bash)" "$2" >"$fakes/$1"
  chmod +x "$fakes/$1"
}
fake lsblk '
case "$*" in
  *PKNAME,PARTLABEL*) printf "%b" "${FAKE_LABELS:-}" ;;
  *-dnbo\ SIZE*) echo "$FAKE_SIZE" ;;
  *FSTYPE*) printf "%b" "${FAKE_FSTYPES:-}" ;;
  *MOUNTPOINTS*) printf "%b" "${FAKE_MOUNTS:-}" ;;
  *-nro\ TYPE*) printf "%b" "${FAKE_TYPES:-disk\n}" ;;
  *-nlpo\ NAME*) printf "/dev/vda\n/dev/vda1\n/dev/vda2\n" ;;
  *-dbP*) echo "SERIAL=\"\" WWN=\"\" SIZE=\"68719476736\" MODEL=\"Same Disk\"" ;;
esac'
fake udevadm 'echo "${FAKE_ID_PATH:-}"'
fake realpath 'echo "$1"'
fake swapon 'printf "%b" "${FAKE_SWAPS:-}"'
fake swapoff 'echo "swapoff $*" >>"$CALLS"'
fake findmnt '[[ -n ${FAKE_MNT:-} ]] || exit 1; [[ $1 == -no ]] && echo "$FAKE_MNT"; exit 0'
fake umount 'echo "umount $*" >>"$CALLS"'
fake cryptsetup '
case $1 in
  status) [[ -n ${FAKE_CRYPT:-} ]] || exit 4; printf "  type: LUKS2\n  device:  %s\n" "$FAKE_CRYPT" ;;
  close) echo "cryptsetup close $2" >>"$CALLS" ;;
esac'

failures=0
# a name, then the shell code to run with common.sh loaded, then whether it has to succeed
expect() {
  local name=$1 code=$2 want=$3 got=0
  : >"$CALLS"
  bash -c "set -euo pipefail; source '$COMMON'; $code" >"$work/out" 2>&1 || got=1
  if [[ $want == succeeds && $got == 0 || $want == fails && $got == 1 ]]; then
    echo "ok: $name"
  else
    echo "FAIL: $name ($want, but it didn't)" >&2
    sed 's/^/  /' "$work/out" >&2
    failures=$((failures + 1))
  fi
}
called() { grep -qxF -- "$1" "$CALLS"; }

expect "disk identity has the physical path" \
  "[[ \$(FAKE_ID_PATH=pci-0000:00:14.0-usb-0:2:1.0-scsi-0:0:0:0 disk_identity /dev/sdb) == 'SERIAL=\"\" WWN=\"\" SIZE=\"68719476736\" MODEL=\"Same Disk\" ID_PATH=\"pci-0000:00:14.0-usb-0:2:1.0-scsi-0:0:0:0\"' ]]" succeeds
expect "two disks of the same model with no serial tell apart by where they're plugged in" \
  "[[ \$(FAKE_ID_PATH=pci-a disk_identity /dev/sdb) != \$(FAKE_ID_PATH=pci-b disk_identity /dev/sdb) ]]" succeeds

expect "apps with a quote refused" "validate_apps 'glow \"x'" fails

expect "a mounted disk is in use" "FAKE_MOUNTS='/iso\n' disk_in_use /dev/sda" succeeds
expect "an open LUKS disk is in use" "FAKE_TYPES='disk\npart\ncrypt\n' disk_in_use /dev/sdb" succeeds
expect "an idle disk isn't in use" "FAKE_MOUNTS='\n\n' disk_in_use /dev/vda" fails
expect "an installer image is found unmounted" "FAKE_FSTYPES='iso9660\niso9660\n' holds_installer_image /dev/sda" succeeds
expect "a normal disk holds no installer image" "FAKE_FSTYPES='\nvfat\ncrypto_LUKS\n' holds_installer_image /dev/vda" fails
# more than a pipe holds after the match, so grep -q always stops before lsblk is done
long=$(printf 'vfat\\n%.0s' {1..20000})
expect "an installer image is found among many partitions" "FAKE_FSTYPES='iso9660\n$long' holds_installer_image /dev/sda" succeeds
expect "a disk with many mounts is in use" "FAKE_MOUNTS='/iso\n${long//vfat//m}' disk_in_use /dev/sda" succeeds
expect "an LVM member is found" "[[ \$(FAKE_FSTYPES='\nLVM2_member\n' storage_member /dev/vda) == 'an LVM volume group' ]]" succeeds
expect "a ZFS member is found" "[[ \$(FAKE_FSTYPES='\nvfat\nzfs_member\n' storage_member /dev/vda) == 'a ZFS pool' ]]" succeeds
expect "an md RAID member is found" "[[ \$(FAKE_FSTYPES='\nlinux_raid_member\n' storage_member /dev/vda) == 'a RAID array' ]]" succeeds
expect "a firmware RAID member is found" "FAKE_FSTYPES='isw_raid_member\n' storage_member /dev/vda" succeeds
expect "a plain disk isn't a member" "FAKE_FSTYPES='\nvfat\ncrypto_LUKS\nntfs\n' storage_member /dev/vda" fails

labels='/dev/vda disk-main-ESP\n/dev/vda disk-main-luks\n'
expect "the disk's own partition labels are fine" "FAKE_LABELS='$labels' validate_partlabels /dev/vda" succeeds
expect "another disk with the same labels is refused" "FAKE_LABELS='$labels/dev/sdb disk-main-luks\n' validate_partlabels /dev/vda" fails
expect "ALLOW_SAME_PARTLABELS=1 lets it through" "ALLOW_SAME_PARTLABELS=1 FAKE_LABELS='$labels/dev/sdb disk-main-luks\n' validate_partlabels /dev/vda" succeeds

gib=$((1024 ** 3))
expect "4G swap fits on 20G" "FAKE_SIZE=$((20 * gib)) validate_swap_fits /dev/vda 4G" succeeds
expect "16G swap on 20G is refused" "FAKE_SIZE=$((20 * gib)) validate_swap_fits /dev/vda 16G" fails
expect "8G swap on exactly 17G fits" "FAKE_SIZE=$((17 * gib)) validate_swap_fits /dev/vda 8G" succeeds

state=$work/state
ours="STATE=$state; printf '/dev/vda\n' >$state"
expect "teardown with nothing recorded does nothing" "STATE=$work/none FAKE_MNT=/dev/sdb1 teardown; [[ ! -s \$CALLS ]]" succeeds
expect "teardown closes what the last attempt opened" \
  "$ours; FAKE_SWAPS='/mnt/.swapvol/swapfile\n/dev/zram0\n' FAKE_MNT='/dev/mapper/cryptroot[/root]' FAKE_CRYPT=/dev/vda2 teardown; [[ ! -e $state ]]" succeeds
if called "swapoff /mnt/.swapvol/swapfile" && called "umount -R /mnt" && called "cryptsetup close cryptroot" && ! grep -q zram "$CALLS"; then
  echo "ok: teardown turned off only the swap under /mnt, unmounted and closed"
else
  echo "FAIL: teardown didn't undo the right things" >&2
  failures=$((failures + 1))
fi
expect "teardown refuses /mnt mounted from somewhere else" "$ours; FAKE_MNT=/dev/sdb1 teardown" fails
expect "teardown refuses cryptroot on another disk" "$ours; FAKE_CRYPT=/dev/sdb2 teardown" fails
expect "teardown with everything already closed forgets the record" "$ours; teardown; [[ ! -e $state ]]" succeeds

for s in prepare:'before changing anything' partition:'half formatted' install:'formatted and mounted at /mnt'; do
  expect "an error in ${s%%:*} says '${s#*:}'" \
    "DISK=/dev/vda; set -E; trap on_error ERR; stage ${s%%:*} >/dev/null; false" fails
  grep -qF "${s#*:}" "$work/out" || {
    echo "FAIL: the $s error said: $(cat "$work/out")" >&2
    failures=$((failures + 1))
  }
done

buttons=abxyudlrqezcst
expect "a handheld passphrase of button characters is taken" "printf '%s\n' ayuuqdcbxyrl | handheld_passphrase_ok $buttons" succeeds
expect "a handheld passphrase with a character no button types is refused" "printf '%s\n' ayuuqdcbxyrB | handheld_passphrase_ok $buttons" fails
expect "a handheld passphrase under 12 presses is refused" "printf '%s\n' ayuuqdcbxyr | handheld_passphrase_ok $buttons" fails
expect "a Deck back button is refused where the table hasn't got it" "printf '%s\n' ayuuqdcbxyr1 | handheld_passphrase_ok $buttons" fails
expect "a button table that isn't plain characters is refused" "printf '%s\n' ayuuqdcbxyrl | handheld_passphrase_ok 'a-z]|.*'" fails

if ((failures)); then
  echo "$failures failed" >&2
  exit 1
fi
