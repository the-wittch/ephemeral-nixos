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
TAG=v2.2.0
REV=ba84045377cbe756eaa43b9555adc79e31f0bba8
git clone --branch "$TAG" --depth 1 https://github.com/the-wittch/ephemeral-nixos.git ~/nix
cd ~/nix
[[ $(git rev-parse HEAD) == "$REV" ]] || echo "hash mismatch, stop here"
sudo ./scripts/tui.sh
```

Or let the bootstrap script clone and check it:

```bash
curl -fsSL https://raw.githubusercontent.com/the-wittch/ephemeral-nixos/v2.2.0/scripts/bootstrap-tui.sh -o /tmp/bootstrap-tui.sh
sudo env TAG=v2.2.0 EXPECTED_REV=ba84045377cbe756eaa43b9555adc79e31f0bba8 bash /tmp/bootstrap-tui.sh
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

The installer asks for the disk, desktop, hostname, user, swap size, keyboard layout and language, then a LUKS passphrase and a login password. With a non-US layout, switch the live session to that layout before typing the passphrase, since that's the layout you'll type it in at boot. It asks whether to add the extras (Steam, VS Code, Brave and more shell tools; off keeps the system lean), turns on the laptop extras when it finds a battery, and asks about fingerprint login when it finds a reader. If it finds an NVIDIA card, it shows its name and asks whether to install the drivers: `open` for RTX 20 / GTX 16 series and newer, `legacy` for GTX 900 / 10 series, or `none`. The driver is compiled during the install, which adds a few minutes. It detects the rest of your hardware once the disk is mounted. When it's done, remove the ISO and reboot.

**VMs:** UEFI with Secure Boot off and a VirtIO disk (`/dev/vda`). 64 GB of disk and 8 GB of RAM is plenty. The boot splash can look odd on virtual GPUs; unlocking still works.

## After the install

The config is at `/persist/etc/nixos`. Rebuild with `path:`, so Nix can see the gitignored files:

```bash
cd /persist/etc/nixos
sudo nixos-rebuild switch --flake path:.#nixos
```

To see the wipe working: `sudo touch /ephemeral-test`, put a file in `~`, and reboot. The first is gone, the second is still there.

A read-only copy of the config as installed is kept in `/persist/etc/nixos-install`. However far `/persist/etc/nixos` moves on, this gets the fresh-install system back (your files stay):

```bash
sudo nixos-rebuild switch --flake path:/persist/etc/nixos-install#nixos
```

### Next step: the full config

The installer sets up the `minimal` profile of [ephemeral-nixos-config](https://github.com/the-wittch/ephemeral-nixos-config): a working desktop and nothing more. Its `full` profile adds Home Manager dotfiles, a Catppuccin theme, snapshots of `/persist`, OpenSnitch, Control D, SMB mounts, `ephemeral-apps` for picking apps, and more. Opting in is one setting; its README's [Setup](https://github.com/the-wittch/ephemeral-nixos-config#setup) section has the steps, including trying it in a VM first.

### settings.json

The installer writes your answers, for example:

```json
{
  "hostname": "nixos",
  "username": "nixuser",
  "diskDevice": "/dev/nvme0n1",
  "swapSize": "16G",
  "timeZone": "Europe/Berlin",
  "desktop": "gnome",
  "nvidia": "none",
  "keyboardLayout": "de",
  "keyboardVariant": "",
  "locale": "de_DE.UTF-8",
  "extras": false,
  "laptop": true,
  "fingerprint": false,
  "profile": "minimal"
}
```

The config's [`settings.example.json`](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/settings.example.json) lists every setting with its default; the [config template's README](https://github.com/the-wittch/ephemeral-nixos-config#settingsjson) explains each one. `diskDevice` is the whole disk, not a partition. `desktop` is `plasma`, `gnome` or `niri`. `nvidia` is `none`, `open` or `legacy`. Installs older than v1.3 have a `settings.nix` instead; it still loads, and the rebuild prints the command that converts it.

### Unfree packages

Nothing unfree builds unless it's listed by name, next to the package that needs it:

```nix
ephemeral.unfreePackages = [ "steam" "steam-unwrapped" ];
```

A package that isn't listed fails the build and tells you the name to add.

## Layout

This repo is the installer and nothing else. Its `flake.nix` is the machine flake that gets installed to `/persist/etc/nixos`:

```nix
{
  inputs = {
    ephemeral.url = "github:the-wittch/ephemeral-nixos-config/v3.0.0";
    # the installer runs these by name
    nixpkgs.follows = "ephemeral/nixpkgs";
    disko.follows = "ephemeral/disko";
  };
  outputs = inputs: inputs.ephemeral.lib.mkFlake { inherit inputs; root = ./.; };
}
```

Every module, the partition layout included, comes from that pinned release of the config repo; `flake.lock` pins the rest. Each installer release pins a tested config release.

| path | what |
|------|------|
| `flake.nix`, `flake.lock` | the machine flake, copied to `/persist/etc/nixos` and `/persist/etc/nixos-install` |
| `hardware-configuration.example.nix` | used until the installer has detected your hardware |
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
- `flake.lock` pins the config release and, through it, nixpkgs and everything else. Installs are reproducible: the same installer release always installs the same system.

## Troubleshooting

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

## Manual install

```bash
printf '{\n  "username": "nixuser",\n  "diskDevice": "/dev/vda",\n  "desktop": "gnome",\n  "profile": "minimal"\n}\n' > settings.json
nano settings.json
sudo nix run --inputs-from path:. disko -- --flake path:.#nixos --mode destroy,format,mount
sudo btrfs subvolume list /mnt
sudo nixos-generate-config --no-filesystems --root /mnt --show-hardware-config > hardware-configuration.nix
sudo mkdir -p /mnt/persist/passwords /mnt/persist/etc/ssh /mnt/persist/etc/nixos /mnt/persist/home/nixuser
sudo chmod 700 /mnt/persist/passwords /mnt/persist/home/nixuser
mkpasswd -m yescrypt | sudo tee /mnt/persist/passwords/nixuser
sudo chmod 600 /mnt/persist/passwords/nixuser
sudo cp flake.nix flake.lock .gitignore settings.json hardware-configuration.nix /mnt/persist/etc/nixos/
sudo nixos-install --no-root-passwd --flake path:/mnt/persist/etc/nixos#nixos
```

The subvolume list should include `root-blank`. Use your own username in place of `nixuser`.

## Wallpaper

The wallpaper and default user picture come from the [config repo](https://github.com/the-wittch/ephemeral-nixos-config#wallpaper), which also lists the credits for the snowflake.
