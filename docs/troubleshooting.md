# Troubleshooting

## Good to know

- **Changing your password:** `passwd` doesn't survive a reboot, because users are defined by the config. Write a new hash instead; it applies at the next login:

  ```bash
  mkpasswd -m yescrypt | sudo tee /persist/passwords/$USER
  sudo chmod 600 /persist/passwords/$USER
  ```

- Hibernation is off. Suspend works.
- `/boot` isn't encrypted. The boot menu editor is disabled.
- SSH is off. `services.openssh.enable = true` turns it on with port 22 still firewalled; the host key is kept in `/persist/etc/ssh`.
- `flake.lock` pins the config release and, through it, nixpkgs and everything else. Installs are reproducible: the same installer release always installs the same system.

## Common problems

| problem | fix |
|---------|-----|
| black screen at the LUKS prompt | add `boot.kernelParams = [ "plymouth.use-simpledrm" ];`, or load your GPU driver early in `boot.initrd.kernelModules` |
| settings are ignored | rebuild with `path:.#nixos`, not `.#nixos` |
| `settings.json is not in this flake` | rebuild from `/persist/etc/nixos` with `path:.#nixos`; the file is there, but only `path:` sees it |
| can't log in | `/persist/passwords/<user>` is missing; write it as above from a TTY or the ISO |
| no home directory | `sudo install -d -m 0700 -o $USER -g users /persist/home/$USER`, then log in again |
| VM sits at "Booting from Hard Disk" | use UEFI with Secure Boot off, and detach the ISO |
| the menus crash or show no disks | use `bootstrap.sh` instead |
| `getcwd` errors during install | `cd / && sudo nixos-install --no-root-passwd --flake path:/mnt/persist/etc/nixos#nixos` |
