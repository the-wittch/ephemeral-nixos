# Troubleshooting

`flake.lock` pins the config release and, through it, nixpkgs and everything else, so the same installer release always installs the same system. Once the machine boots, the [config's docs](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/docs/troubleshooting.md) take over: passwords, SSH, rollbacks and recovery.

## Common problems

| problem | fix |
|---------|-----|
| black screen at the LUKS prompt | add `boot.kernelParams = [ "plymouth.use-simpledrm" ];`, or load your GPU driver early in `boot.initrd.kernelModules` |
| settings are ignored | rebuild with `path:.#nixos`, not `.#nixos` |
| `settings.json is not in this flake` | rebuild from `/persist/etc/nixos` with `path:.#nixos`; the file is there, but only `path:` sees it |
| can't log in | `/persist/passwords/<user>` is missing; write it from a TTY or the ISO, as in the [security notes](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/docs/security.md) |
| no home directory | `sudo install -d -m 0700 -o $USER -g users /persist/home/$USER`, then log in again |
| VM sits at "Booting from Hard Disk" | use UEFI with Secure Boot off, and detach the ISO |
| the menus crash or show no disks | use `bootstrap.sh` instead |
| "has mounted filesystems or swap" | that disk is in use, often the USB stick you booted from; pick another, or unmount it after a failed try with `sudo umount -R /mnt && sudo swapoff -a` |
| `getcwd` errors during install | `cd / && sudo nixos-install --no-root-passwd --flake path:/mnt/persist/etc/nixos#nixos` |

## Reporting a problem

If the installer fails or the new install won't boot, open an [install problem](https://github.com/the-wittch/ephemeral-nixos/issues/new/choose) with the end of `/tmp/ephemeral-install.log`. A successful install keeps a copy in `/var/log/ephemeral-install.log`. Once the machine boots, problems go to the [config's tracker](https://github.com/the-wittch/ephemeral-nixos-config/issues/new/choose). Security problems go through [SECURITY.md](../SECURITY.md), not an issue.
