# Working on the installer

This repo is the installer and nothing else. Its `flake.nix`, shown in the [README](../README.md#how-it-works), is the machine flake that gets installed.

| path | what |
|------|------|
| `flake.nix`, `flake.lock` | the machine flake, copied to `/persist/etc/nixos` and `/persist/etc/nixos-install` |
| `hardware-configuration.example.nix` | used until the installer has detected your hardware |
| `scripts/tui.sh`, `scripts/bootstrap-tui.sh` | installer with menus |
| `scripts/bootstrap.sh` | installer without menus, for piping from curl |
| `scripts/install.sh` | the part that partitions and installs |
| `modules/iso.nix` | the Ephemeral Nix ISO: the installer on the minimal NixOS ISO, in Catppuccin colours |
| `modules/tests/install.nix`, `tests/install/` | the end-to-end test: runs `install.sh` in an offline VM, then boots the disk it wrote |

CI checks the scripts and the machine flake on every pull request. The install test takes a while and only runs when you start it: `gh workflow run install-test.yml --ref <branch>`, or locally:

```bash
printf '{"username": "alice"}\n' > settings.json
cp hardware-configuration.example.nix hardware-configuration.nix
nix build path:.#checks.x86_64-linux.install -L
```

Releases mostly run themselves. On the 3rd of each month, `follow config` opens a PR for the newest config release and runs the install test on it. If the test passes, the PR is merged, the next patch is tagged and released, the ISO is built and attested, and a `pin` PR updates the install instructions.
