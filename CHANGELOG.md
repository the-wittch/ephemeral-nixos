# Changelog

Breaking changes are listed first in each release, with what to do about them. Releases before 3.0.0 are on the [releases page](https://github.com/the-wittch/ephemeral-nixos/releases).

## 3.7.5

### Changed

- New installs get config [v4.0.0](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/CHANGELOG.md).
- The workflows that pin a new config and release the installer keep their GitHub token out of the repository checkout.

## 3.7.4

### Fixed

- Every install since v3.0.0 left `/persist/etc/nixos` readable only by root, so `ephemeral-update` said "no flake at /persist/etc/nixos". The folder is readable by everyone again, like the files in it, and still writable only by root. Config v3.11.0 fixes it on machines that are already installed, when they rebuild.
- The install test checks that the user can read the installed flake.

### Changed

- New installs get config [v3.11.0](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/CHANGELOG.md).
- `x` ticks options and apps, the same key as in `ephemeral-apps` and `ephemeral-setup`. Space still works.

## 3.7.3

### Changed

- New installs get config [v3.10.0](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/CHANGELOG.md).
- The installer follows the config's new weekly releases: each Tuesday it moves to the newest one, and if the install test passes, a new installer and ISO are released. New installs start with fixes from that week.
- Releases are immutable: once published, a release's tag and files can't change. `gh release verify-asset ephemeral-nix-*.iso -R the-wittch/ephemeral-nixos` checks that an ISO is the one the latest release went out with.
- The Go dependencies get security updates from Dependabot. `golang.org/x/sys` moves to v0.44.0 for GO-2026-5024, which the installer never called.

## 3.7.2

### Fixed

- The app list scrolls. Categories longer than the screen, such as Terminal with 37 apps, were cut off with no way to reach the rest.
- App names and descriptions no longer run together: the name column fits the longest name in the category. The selected app's full description and any warning show under the list, so a shortened row hides nothing.

### Changed

- New installs get config [v3.9.0](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/CHANGELOG.md).
- The installer uses the whole screen, up to 140 columns wide and all of its height. On a 1080p laptop it had left about a third of the width empty.
- When the screen has room, the installer is framed, with the step in the top border. The title decodes out of scrambled characters on the welcome screen, the snowflake flashes mauve and blue as it appears, a light travels along the install's progress bar, and "Installed" glitches in at the end. `e` on the welcome screen turns the animations off, as does `ephemeral-install --no-effects`.

## 3.7.1

### Fixed

- The app list loads on the ISO. It failed every time, because nix wouldn't read the installer's copy of the flake through the link in `/etc`. If it fails for another reason, the screen says why, and `r` tries again.
- The vault name and Control D resolver id fields say what to type, and ticking their option puts the cursor in them.
- The machine and secrets fields show which one has the cursor, say what goes in it, and `enter` goes through them one at a time on both screens.

### Changed

- Problems show in one place, just above the keys, and go away at the next key.
- With no network, the welcome screen starts on Connect to Wi-Fi and moves to Install once it's online.
- Wi-Fi networks joined on the ISO are copied to the new system, root-only like NetworkManager keeps them, so it's online at the first login.
- The snowflake shows on the welcome screen whenever it fits, so on 768 px screens too.
- More space between rows when the screen has room.

## 3.7.0

### Changed

- With a non-US keyboard layout, the console switches to it before the passphrase, so it's typed the way it will be at boot.
- The install runs on its own: leaving the installer, or the installer crashing, doesn't stop it, and running `ephemeral-install` again picks it back up.
- Disks under 32G are marked as tight for a full install.
- The welcome screen shows the installer's version next to the config version it installs.
- `scripts/tui.sh` starts the same installer from a clone of the repo, in place of its own menus. Without a terminal it still falls back to `install.sh`'s prompts.
- The install test goes through the installer's `--answers` mode, so each release checks that the disk unlocks with the passphrase it was given.

## 3.6.0

### Changed

- New installs get config [v3.8.3](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/CHANGELOG.md).
- The ISO starts a new installer: one question per screen, `esc` to go back, input checked as you type, and a review screen where any answer can be changed before the disk is touched. Installing needs the disk's name typed, not just a yes. While it installs, it shows the steps instead of raw build output, and if something fails it shows where and offers to try again with the same answers. `install.sh` still does all the disk work, and unattended installs are unchanged.
- The disk passphrase and login password are asked before anything is written, instead of halfway through the install.
- The ISO's console uses Terminus at 24 px and has mauve in its palette, for the installer's colours and the snowflake on its welcome screen.

## 3.5.1

### Fixed

- The disk check asks lsblk whether a device is a whole disk instead of guessing from its name. `/dev/disk/by-id/` names of whole disks are accepted now, and partitions are refused whatever they're called.
- A disk with an open LUKS, LVM or RAID device on it counts as in use, like a mounted one.

### Changed

- New installs get config [v3.8.1](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/CHANGELOG.md).

## 3.5.0

### Added

- Umbriel as a fourth desktop choice, with the Noctalia shell like niri. It's a young compositor, so its settings and keybinds may still change between releases.

### Changed

- New installs get config [v3.8.0](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/CHANGELOG.md).
- New installs get config [v3.8.0](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/CHANGELOG.md): Plasma tiles with Krohnkite and takes its colors from the wallpaper.

## 3.4.2

### Changed

- New installs get config [v3.7.0](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/CHANGELOG.md).

## 3.4.1

### Changed

- New installs get config [v3.6.2](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/CHANGELOG.md).

## 3.4.0

### Changed

- New installs get config [v3.5.0](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/CHANGELOG.md).
- The installer refuses a disk with mounted filesystems or swap, such as the USB stick it runs from, and the menu leaves those disks out. `FORCE_DISK=1` overrides it.
- The ISO is named after its release, such as `ephemeral-nix-v3.4.0.iso`.

### Fixed

- The installer took any Proton Pass vault name, including ones with quotes or `$` that the login shell would run as a command. It refuses those now.
- `bootstrap.sh` and `bootstrap-tui.sh` stopped with a permissions error when run from the ISO's copy of the scripts.
- The README didn't say that an install without the menus uses the user `nixuser` and skips the apps.

### Added

- A private way to report security problems, in `SECURITY.md`. The install problem form asks how you installed.
- A hardware page: what's required, what's been tested and what the installer detects. Also CONTRIBUTING.md and a pull request template.
- The installer's output is saved to `/tmp/ephemeral-install.log`, and a successful install keeps a copy in `/var/log/ephemeral-install.log` for bug reports.

## 3.3.0

### Added

- For the full system, the installer offers the same apps as `ephemeral-apps`, so they're there on the first boot. Unattended installs use `APPS`, e.g. `APPS=obsidian,glow`.

### Changed

- New installs get config [v3.4.0](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/CHANGELOG.md).

## 3.2.5

### Changed

- The install test now boots the ISO's own live system and installs from its read-only copy of the scripts, so it catches problems that only show up on the ISO.
- CI checks on every pull request that the ISO still has Wi-Fi, NetworkManager and the installer.

### Fixed

- When a release already had a changelog entry, the monthly pin would have added a second heading for the same version.

## 3.2.4

### Changed

- New installs get config [v3.3.1](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/CHANGELOG.md), with the Terminal apps in `ephemeral-apps`.

### Added

- Keyboard layout, variant, language and timezone are picked from a list you can type to narrow down, in place of typing the exact name. Something not in the list can still be typed in, such as `us,de` for two layouts.
- The ISO comes with a build attestation. `gh attestation verify` checks it was built by this repo's CI.
- Monthly releases run on their own: once the install test passes on the `follow config` PR, it's merged and released.

### Fixed

- The README said the installer used config v3.2.3, which doesn't exist. It uses v3.2.1.

## 3.2.3

### Fixed

- The ISO's installer stopped right away with a permissions error. It tried to make `install.sh` executable, but the ISO runs it from the read-only Nix store.
- Installs from the ISO left the files in `/persist/etc/nixos` read-only. They are now copied as normal, editable files.
- After you set up Wi-Fi in `nmtui`, the ISO waits for the connection before it checks again, so the installer starts instead of asking a second time.

## 3.2.2

### Fixed

- Wi-Fi on the ISO: NetworkManager showed every Wi-Fi card as unavailable, because the ISO turned off the wpa_supplicant it needs.

## 3.2.0

### Added

- The Ephemeral Nix ISO: the installer on the minimal NixOS ISO, branded, starting on its own. Each release has it attached.
- A monthly workflow that opens a PR pointing the installer at the newest config release.
- The README is split: the overview stays, the reference moves to `docs/`, one topic per page.

### Changed

- The installer asks what to install and installs it: the full system by default, with OpenSnitch, the Proton Pass SSH agent and Control D as options, or a small install to finish later with `ephemeral-setup`. Unattended installs use `PROFILE`, `OPENSNITCH`, `PASS_SSH_AGENT`, `PASS_SSH_VAULT` and `CONTROLD_RESOLVER`.
- The end-to-end install test installs the full system.

## 3.1.0

### Added

- An end-to-end test that runs `install.sh` in an offline VM, then unlocks and boots the disk it wrote. It runs on demand, not on every pull request.
- Unattended installs: `PASSWORD_HASH` and `HARDWARE_CONFIG` answer the last two questions ahead of time.
- Offline installs: when the system is already built in the live environment, the installer copies it to the disk instead of building it again.

### Changed

- The disk is confirmed once; disko no longer asks a second time.
- Partitioning runs the machine's own disko script instead of `nix run disko`, so an install no longer downloads a second copy of nixpkgs. `flake.nix` drops its `disko` input.

## 3.0.0

### Breaking

- **No modules of its own.** The installer's `flake.nix` is now the machine flake: it installs the `minimal` profile of [ephemeral-nixos-config](https://github.com/the-wittch/ephemeral-nixos-config) v3.0.0 through `lib.mkFlake`. The system it installs is the same as before. Existing machines aren't affected; to move one onto this layout, see the config repo's [CHANGELOG](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/CHANGELOG.md).
- **chezmoi is gone**, with `CHEZMOI_REPO` and `CHEZMOI_REV`.
- **`FLAKE_INPUTS` is gone.** Installs always use the pinned `flake.lock`.

### Added

- A read-only copy of the installed config in `/persist/etc/nixos-install`, to rebuild back to the fresh-install system.
- Opting into the full config is one setting, `"profile": "full"`, instead of switching repos.
- Issue templates.
