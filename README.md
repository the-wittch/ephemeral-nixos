# Ephemeral Nix

[![CI](https://github.com/the-wittch/ephemeral-nixos/actions/workflows/ci.yml/badge.svg)](https://github.com/the-wittch/ephemeral-nixos/actions/workflows/ci.yml)

<img width="3840" height="2160" alt="wallpaper" src="https://github.com/user-attachments/assets/b4e08080-876a-4ab3-911f-4055a4f2de0d" />

An installer for NixOS on an encrypted btrfs disk where `/` is wiped on every boot. Only `/home` and a short list of system paths survive, on `/persist`. You pick Plasma, GNOME, niri or Umbriel (both with Noctalia) during the install.

```
disk ─ ESP (/boot, 1G)
     └ LUKS2 ─ btrfs: root, root-blank, nix, persist, log, swap
```

Machine details (hostname, user, disk, desktop) live in `settings.json`, which is gitignored. The login password hash lives in `/persist/passwords/`, never in the flake.

## Limits

- x86_64 with UEFI, Secure Boot off. See [Hardware](docs/hardware.md) for what's been tested.
- Installing wipes the whole disk, and reinstalling wipes it again, `/persist` included. Make a [backup](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/docs/backup.md) first. No dual boot.
- One user per machine.
- Suspend works; hibernation doesn't.
- The disk unlocks with a passphrase only, no TPM.
- It's opinionated: Catppuccin, Ghostty and zsh by default. Changing them takes a module of your own ([appearance](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/docs/appearance.md)).

## Install

### From the ISO (easiest)

Download the `.iso` and `SHA256SUMS` from the latest [release](https://github.com/the-wittch/ephemeral-nixos/releases), check it, and write it to a USB stick. From v3.2.4 on, `gh attestation verify` also proves the ISO was built by this repo's CI:

```bash
sha256sum -c SHA256SUMS
gh attestation verify ephemeral-nix-*.iso -R the-wittch/ephemeral-nixos
sudo dd if=ephemeral-nix-*.iso of=/dev/sdX bs=4M status=progress oflag=sync
```

Boot it in UEFI mode. The installer starts on its own and offers `nmtui` if there's no network yet. Run `ephemeral-install` to start it again.

### From the NixOS ISO

Boot the NixOS graphical ISO in UEFI mode and connect to the network, then use one of the ways below.

#### From a release (recommended)

Take the tag and commit from the latest [release](https://github.com/the-wittch/ephemeral-nixos/releases), or from [`RELEASE`](./RELEASE) on that tag, and check the clone against them:

```bash
TAG=v3.5.0
REV=cd349d855a098043b8a508ffea445690cc12aa47
git clone --branch "$TAG" --depth 1 https://github.com/the-wittch/ephemeral-nixos.git ~/nix
cd ~/nix
[[ $(git rev-parse HEAD) == "$REV" ]] || echo "hash mismatch, stop here"
sudo ./scripts/tui.sh
```

Or let the bootstrap script clone and check it:

```bash
curl -fsSL https://raw.githubusercontent.com/the-wittch/ephemeral-nixos/v3.5.0/scripts/bootstrap-tui.sh -o /tmp/bootstrap-tui.sh
sudo env TAG=v3.5.0 EXPECTED_REV=cd349d855a098043b8a508ffea445690cc12aa47 bash /tmp/bootstrap-tui.sh
```

#### From main

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

Without the menus, it doesn't ask for the hostname, user, swap size or timezone, and it skips the apps. It uses `nixos`, `nixuser`, `4G` and `UTC` unless you set them, as in [Unattended installs](docs/unattended.md).

The installer asks for the disk, desktop, hostname, user, swap size, keyboard layout, language and timezone (type to search the list), then a LUKS passphrase and a login password. With a non-US layout, switch the live session to that layout before typing the passphrase, since that's the layout you'll type it in at boot. It asks what to install: the full system, or a small install (a plain desktop, finished later with `ephemeral-setup`). For the full system it asks about the extras (Steam, VS Code, Brave and more shell tools), OpenSnitch, the Proton Pass SSH agent and Control D, then offers the same apps as `ephemeral-apps`. It turns on the laptop extras when it finds a battery, and asks about fingerprint login when it finds a reader. If it finds an NVIDIA card, it shows its name and asks whether to install the drivers: `open` for RTX 20 / GTX 16 series and newer, `legacy` for GTX 900 / 10 series, or `none`. The driver is compiled during the install, which adds a few minutes. It detects the rest of your hardware once the disk is mounted. When it's done, remove the ISO and reboot.

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

### Changing it later

`ephemeral-setup` changes what's installed: the profile, OpenSnitch, the Proton Pass SSH agent and Control D. After a small install, run it to get the full system. The [config's README](https://github.com/the-wittch/ephemeral-nixos-config#setup) covers the rest, including `ephemeral-apps` for extra apps.

## How it works

This repo is the installer and nothing else. Its `flake.nix` is the machine flake that gets installed to `/persist/etc/nixos`:

```nix
{
  inputs = {
    ephemeral.url = "github:the-wittch/ephemeral-nixos-config/v3.8.0";
    # the installer runs nixpkgs tools by name
    nixpkgs.follows = "ephemeral/nixpkgs";
  };
  outputs = inputs: inputs.ephemeral.lib.mkFlake { inherit inputs; root = ./.; };
}
```

Every module, the partition layout included, comes from that pinned release of the config repo; `flake.lock` pins the rest. Each installer release pins a tested config release. This repo's own `modules/` only holds the ISO and the install test, which machines never see.

## Wallpaper

The wallpaper and default user picture come from the [config repo](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/docs/appearance.md#wallpaper), which also lists the credits for the snowflake.

## Documentation

- [Unattended installs](docs/unattended.md): every question answered ahead of time
- [settings.json and unfree packages](docs/settings.md)
- [Troubleshooting](docs/troubleshooting.md): install problems, and where to report them
- [Manual install](docs/manual-install.md): the same steps without the scripts
- [Hardware](docs/hardware.md): what's required, what's tested and what the installer detects
- [Working on the installer](docs/development.md): files, CI and the install test
- [The config's docs](https://github.com/the-wittch/ephemeral-nixos-config#documentation): everything after the install
