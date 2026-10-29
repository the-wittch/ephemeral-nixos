# Changelog

Breaking changes are listed first in each release, with what to do about them. Releases before 3.0.0 are on the [releases page](https://github.com/the-wittch/ephemeral-nixos/releases).

## 3.0.0

### Breaking

- **No modules of its own.** The installer's `flake.nix` is now the machine flake: it installs the `minimal` profile of [ephemeral-nixos-config](https://github.com/the-wittch/ephemeral-nixos-config) v3.0.0 through `lib.mkFlake`. The system it installs is the same as before. Existing machines aren't affected; to move one onto this layout, see the config repo's [CHANGELOG](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/CHANGELOG.md).
- **chezmoi is gone**, with `CHEZMOI_REPO` and `CHEZMOI_REV`.
- **`FLAKE_INPUTS` is gone.** Installs always use the pinned `flake.lock`.

### Added

- A read-only copy of the installed config in `/persist/etc/nixos-install`, to rebuild back to the fresh-install system.
- Opting into the full config is one setting, `"profile": "full"`, instead of switching repos.
- Issue templates.
