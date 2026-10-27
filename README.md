# Ephemeral Nix

[![CI](https://github.com/the-wittch/ephemeral-nixos/actions/workflows/ci.yml/badge.svg)](https://github.com/the-wittch/ephemeral-nixos/actions/workflows/ci.yml)

<img width="3840" height="2160" alt="wallpaper" src="https://github.com/user-attachments/assets/b4e08080-876a-4ab3-911f-4055a4f2de0d" />

An installer for NixOS on an encrypted btrfs disk where `/` is wiped on every boot. Only `/home` and a short list of system paths survive, on `/persist`. You pick Plasma, GNOME or niri (with Noctalia) during the install.

```
disk ─ ESP (/boot, 1G)
     └ LUKS2 ─ btrfs: root, root-blank, nix, persist, log, swap
```

Machine details (hostname, user, disk, desktop) live in `settings.json`, which is gitignored. The login password hash lives in `/persist/passwords/`, never in the flake.

## Install

Boot the NixOS graphical ISO in UEFI mode and connect to the network.

### From a release (recommended)

Take the tag and commit from the latest [release](https://github.com/the-wittch/ephemeral-nixos/releases), or from [`RELEASE`](./RELEASE) on that tag, and check the clone against them:

```bash
TAG=v2.0.0
REV=d422ad35cb81287c35450b1ee1e1f2551b276690
git clone --branch "$TAG" --depth 1 https://github.com/the-wittch/ephemeral-nixos.git ~/nix
cd ~/nix
[[ $(git rev-parse HEAD) == "$REV" ]] || echo "hash mismatch, stop here"
sudo ./scripts/tui.sh
```

Or let the bootstrap script clone and check it:

```bash
curl -fsSL https://raw.githubusercontent.com/the-wittch/ephemeral-nixos/v2.0.0/scripts/bootstrap-tui.sh -o /tmp/bootstrap-tui.sh
sudo env TAG=v2.0.0 EXPECTED_REV=d422ad35cb81287c35450b1ee1e1f2551b276690 bash /tmp/bootstrap-tui.sh
```

### From main

```bash
git clone https://github.com/the-wittch/ephemeral-nixos.git ~/nix
cd ~/nix
sudo ./scripts/tui.sh
```

Without the menus:

```bash
curl -fsSL https://raw.githubusercontent.com/the-wittch/ephemeral-nixos/main/scripts/bootstrap.sh | sudo env SWAP_SIZE=4G bash -s -- /dev/vda
```

Use `sudo env VAR=...`. Plain `sudo VAR=...` usually drops the variables.

The installer asks for the disk, desktop, hostname, user and swap size, then a LUKS passphrase and a login password. It detects your hardware once the disk is mounted. When it's done, remove the ISO and reboot.

**VMs:** UEFI with Secure Boot off and a VirtIO disk (`/dev/vda`). 64 GB of disk and 8 GB of RAM is plenty. The boot splash can look odd on virtual GPUs; unlocking still works.

## After the install

The config is at `/persist/etc/nixos`. Rebuild with `path:`, so Nix can see the gitignored files:

```bash
cd /persist/etc/nixos
sudo nixos-rebuild switch --flake path:.#nixos
```

To see the wipe working: `sudo touch /ephemeral-test`, put a file in `~`, and reboot. The first is gone, the second is still there.

### Next step

This installer sets up a small first-boot system. [ephemeral-nixos-config](https://github.com/the-wittch/ephemeral-nixos-config) is the full config to switch to afterwards: Home Manager dotfiles, a Catppuccin theme, snapshots of `/persist`, Control D, SMB mounts and more. It reads the same `settings.json`, and its README has the steps for keeping your own changes in a private repo.

### settings.json

```json
{
  "hostname": "nixos",
  "username": "nixuser",
  "diskDevice": "/dev/nvme0n1",
  "swapSize": "16G",
  "timeZone": "UTC",
  "desktop": "plasma"
}
```

`diskDevice` is the whole disk, not a partition. `desktop` is `plasma`, `gnome` or `niri`. Installs older than v1.3 have a `settings.nix` instead; it still loads, and the rebuild prints the command that converts it.

### chezmoi

Set `CHEZMOI_REPO` and `CHEZMOI_REV` (the full 40-character commit) and the installer writes `<repo> <commit>` to `/persist/etc/chezmoi-repo` and `"useChezmoi": true` to `settings.json`. On first boot your dotfiles are cloned and applied. Without them, the config template manages dotfiles with Home Manager.

### Unfree packages

Nothing unfree builds unless it's listed by name, next to the package that needs it:

```nix
ephemeral.unfreePackages = [ "steam" "steam-unwrapped" ];
```

A package that isn't listed fails the build and tells you the name to add.

## Layout

The config follows the [dendritic pattern](https://github.com/mightyiam/dendritic). Every `.nix` file under `modules/` is a [flake-parts](https://flake.parts) module for one feature, loaded automatically by [import-tree](https://github.com/vic/import-tree). Files don't import each other: each one adds to a few shared modules (`flake.modules.nixos.base`, `.pc`, and one per desktop), and the machine is built by combining those. Adding a feature means adding a file.

Most of these files are shared byte for byte with [ephemeral-nixos-config](https://github.com/the-wittch/ephemeral-nixos-config), and its CI fails if they drift apart.

| path | what |
|------|------|
| `modules/host-builder.nix`, `modules/hosts/nixos.nix` | build the `nixos` machine from `settings.json` and `hardware-configuration.nix` |
| `modules/settings/` | the settings options and the `settings.json` loader |
| `modules/disko.nix` | partitions, LUKS, btrfs |
| `modules/boot.nix` | bootloader, boot splash, the root wipe |
| `modules/persist.nix` | what survives a reboot |
| `modules/users.nix` | the user and sudo |
| `modules/dotfiles/` | zsh, and chezmoi when it's on |
| `modules/desktop/` | Plasma, GNOME, niri |
| `modules/apps.nix` | first-boot packages |
| `modules/hardening.nix` | kernel sysctls and blocked modules |
| `modules/unfree.nix` | the unfree allowlist |
| `modules/wallpaper.nix`, `assets/wallpaper.svg` | default wallpaper and login background |
| `scripts/tui.sh`, `scripts/bootstrap-tui.sh` | installer with menus |
| `scripts/bootstrap.sh` | installer without menus, for piping from curl |
| `scripts/install.sh` | the part that partitions and installs |

## Good to know

- **Changing your password:** `passwd` doesn't survive a reboot, because users are defined by the config. Write a new hash instead; it applies at the next login:

  ```bash
  mkpasswd -m yescrypt | sudo tee /persist/passwords/$USER
  sudo chmod 600 /persist/passwords/$USER
  ```

- Hibernation is off. Suspend works.
- `/boot` isn't encrypted. The boot menu editor is disabled.
- SSH is off. `services.openssh.enable = true` turns it on with port 22 still firewalled; the host key is kept in `/persist/etc/ssh`.
- `flake.lock` pins nixpkgs, disko and impermanence. The menu installer asks whether to keep the pins or update them; the plain one keeps them unless `FLAKE_INPUTS=update` is set.

## Troubleshooting

| problem | fix |
|---------|-----|
| black screen at the LUKS prompt | add `boot.kernelParams = [ "plymouth.use-simpledrm" ];`, or load your GPU driver early in `boot.initrd.kernelModules` |
| settings are ignored | rebuild with `path:.#nixos`, not `.#nixos` |
| `settings.json is not in this flake` | `cp settings.example.json settings.json`, then rebuild with `path:.#nixos` |
| can't log in | `/persist/passwords/<user>` is missing; write it as above from a TTY or the ISO |
| no home directory | `sudo install -d -m 0700 -o $USER -g users /persist/home/$USER`, then log in again |
| VM sits at "Booting from Hard Disk" | use UEFI with Secure Boot off, and detach the ISO |
| the menus crash or show no disks | use `bootstrap.sh` instead |
| `getcwd` errors during install | `cd / && sudo nixos-install --no-root-passwd --flake path:/mnt/persist/etc/nixos#nixos` |

## Manual install

```bash
cp settings.example.json settings.json
nano settings.json
sudo nix run --inputs-from path:. disko -- --flake path:.#nixos --mode destroy,format,mount
sudo btrfs subvolume list /mnt
sudo nixos-generate-config --no-filesystems --root /mnt --show-hardware-config > hardware-configuration.nix
sudo mkdir -p /mnt/persist/passwords /mnt/persist/etc/ssh /mnt/persist/etc/nixos /mnt/persist/home/nixuser
sudo chmod 700 /mnt/persist/passwords /mnt/persist/home/nixuser
mkpasswd -m yescrypt | sudo tee /mnt/persist/passwords/nixuser
sudo chmod 600 /mnt/persist/passwords/nixuser
sudo cp -a . /mnt/persist/etc/nixos/
sudo nixos-install --no-root-passwd --flake path:/mnt/persist/etc/nixos#nixos
```

The subvolume list should include `root-blank`. Use your own username in place of `nixuser`.

## Wallpaper

`assets/wallpaper.svg` is rendered to a 4K PNG at build time and installed at `/run/current-system/sw/share/backgrounds/ephemeral/wallpaper.png`. It's the default on GNOME (desktop, lock screen and login screen) and on the Plasma login screen. On the Plasma desktop or niri, pick that file in the wallpaper settings. A wallpaper you choose always wins.

The snowflake is the "NixOS Logo" by [Simon Frankau](https://github.com/simon-frankau), [Tim Cuthbertson](https://github.com/timbertson) and [Daniel Baker](https://github.com/djacu) (maintained by the [NixOS Marketing Team](https://nixos.org/community/teams/marketing/)), from [NixOS/branding](https://github.com/NixOS/branding), licensed under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/). It has been recoloured.
