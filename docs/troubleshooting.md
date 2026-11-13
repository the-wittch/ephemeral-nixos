# Troubleshooting

`flake.lock` pins the config release and, through it, nixpkgs and everything else, so the same installer release always installs the same system. Once the machine boots, the [config's docs](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/docs/troubleshooting.md) take over: passwords, SSH, rollbacks and recovery.

## Common problems

| problem | fix |
|---------|-----|
| black screen at the LUKS prompt | add `boot.kernelParams = [ "plymouth.use-simpledrm" ];`, or load your GPU driver early in `boot.initrd.kernelModules` |
| settings are ignored, or `settings.json is not in this flake` | rebuild with `ephemeral-update`. Running `nix` or `nixos-rebuild` yourself, use `path:/persist/etc/nixos#nixos`: without `path:`, Nix only sees files tracked by git |
| can't log in | `/persist/passwords/<user>` is missing; write it from a TTY or the ISO, as in the [security notes](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/docs/security.md) |
| no home directory | `sudo install -d -m 0700 -o $USER -g users /persist/home/$USER`, then log in again |
| VM sits at "Booting from Hard Disk" | use UEFI with Secure Boot off, and detach the ISO |
| the menus crash or show no disks | use `bootstrap.sh` instead |
| "is in use (mounted, swap, or an open LUKS, LVM or RAID device)" | often the USB stick you booted from, so pick another disk. After a failed install, trying again closes what that attempt left open by itself. If something else has the disk open: `sudo swapoff -a && sudo umount -R /mnt && sudo cryptsetup close cryptroot` |
| "is part of an LVM volume group", "a ZFS pool" or "a RAID array", or "LVM member", "ZFS member" or "RAID member" in the disk list | that storage can span other disks, and erasing this one removes it from all of them. Pick another disk. If you mean to get rid of it everywhere, set `FORCE_DISK=1` |
| "holds an installer image" | that's a USB stick with an ISO on it, likely the one you booted from, even when it isn't mounted; pick another disk |
| "has partitions from an earlier Ephemeral Nix install" | another disk still has the old system on it, with the partition names the new one boots by. Unplug it while you install, or erase it first |
| "isn't the disk that was chosen any more" | a disk was plugged in or out after you chose, so the name now points somewhere else. Nothing was erased; choose the disk again |
| "leaves too little of … for the system" | the swap size is too big for the disk; pick a smaller one |
| "another install is running" | an install is still going, maybe from the screens you left; `ephemeral-install` shows it |
| `getcwd` errors during install | `cd / && sudo nixos-install --no-root-passwd --flake path:/mnt/persist/etc/nixos#nixos` |

## Reporting a problem

If the installer fails or the new install won't boot, open an [install problem](https://github.com/the-wittch/ephemeral-nixos/issues/new/choose) with the end of `/tmp/ephemeral-install.log`; after a second try, the first one is in `/tmp/ephemeral-install.log.1`. If the installer's screens themselves went wrong, or the app list didn't load, add `/tmp/ephemeral-install-ui.log`. A successful install keeps a copy in `/var/log/ephemeral-install.log`. Once the machine boots, problems go to the [config's tracker](https://github.com/the-wittch/ephemeral-nixos-config/issues/new/choose). Security problems go through [SECURITY.md](../SECURITY.md), not an issue.
