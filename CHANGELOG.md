# Changelog

Breaking changes are listed first in each release, with what to do about them. Releases before 3.0.0 are on the [releases page](https://github.com/the-wittch/ephemeral-nixos/releases).

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
