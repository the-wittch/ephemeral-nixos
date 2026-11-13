# Working on the installer

This repo is the installer and nothing else. Its `flake.nix`, shown in the [README](../README.md#how-it-works), is the machine flake that gets installed.

| path | what |
|------|------|
| `flake.nix`, `flake.lock` | the machine flake, copied to `/persist/etc/nixos` and `/persist/etc/nixos-install` |
| `hardware-configuration.example.nix` | used until the installer has detected your hardware |
| `installer/`, `modules/installer.nix` | the installer the ISO starts: a Go program (Bubble Tea) that asks the questions one screen at a time, then runs `install.sh` with the answers |
| `scripts/tui.sh`, `scripts/bootstrap-tui.sh` | build and start the installer's screens from a clone, for the NixOS ISO and the curl install; without a terminal they fall back to `install.sh`'s prompts |
| `scripts/bootstrap.sh` | installer without menus, for piping from curl |
| `scripts/install.sh` | the part that partitions and installs |
| `modules/iso.nix` | the Ephemeral Nix ISO: the installer on the minimal NixOS ISO, in Catppuccin colours. Its live system is `flake.modules.nixos.iso` |
| `modules/tests/install.nix`, `tests/install/` | the end-to-end test: boots the ISO's live system offline, installs from its read-only copy of the scripts, then boots the disk it wrote |

CI checks the scripts and the machine flake on every pull request: the secret scan, actionlint and zizmor on the workflows, shellcheck, nixfmt, the Go tests, `govulncheck`, and `scripts`, which runs `common.sh`'s checks and `teardown` against fake `lsblk`, `findmnt` and `cryptsetup`, so what they refuse and what they undo is tested without a disk. The install test takes a while and only runs when you start it: `gh workflow run install-test.yml --ref <branch>`, or locally:

```bash
printf '{"username": "alice"}\n' > settings.json
cp hardware-configuration.example.nix hardware-configuration.nix
nix build path:.#checks.x86_64-linux.install -L
```

Releases mostly run themselves. Every Tuesday, the day after the config's weekly release, `follow config` opens a PR for the newest config release and runs the install test on it. If the test passes, the PR is merged, the next patch is tagged and made a draft release, the ISO is built, booted, attested and attached, the release is published, and a `pin` PR updates the install instructions.

Releases are immutable: once published, their tag and files can't change. That's why a release starts as a draft and the ISO workflow publishes it after uploading the ISO; it refuses to upload to a release that's already out. A failed ISO build leaves the draft; `gh workflow run iso.yml -f tag=<tag>` builds it again.

Before the ISO is attached, `.github/iso-boot.sh` boots it in QEMU with UEFI and no disk, and waits for the installer's welcome screen. It reads the screen with OCR, since the default boot entry has no serial console. The install test never builds the image, so this is what catches a broken one. If the screen doesn't come up within 5 minutes, the release stays a draft. To try it on a branch without releasing anything: `gh workflow run iso.yml --ref <branch> -f tag=v0.0.0 -f boot_test_only=true`.

## The installer screens

`installer/` asks the questions, checks the answers with the same rules as `common.sh`, and runs `install.sh` with them, so the disk work stays in one place. The disk passphrase goes to `install.sh` on standard input, the way disko asks for it, and the login password only as a yescrypt hash in `PASSWORD_HASH`, so neither is in the environment or the log in plain text.

`install.sh` runs in a session of its own, writing to `/run/ephemeral-install/output` and leaving its exit status in `status` there. The screens only follow that file, so quitting them, or a crash, doesn't stop the install, and starting them again picks it back up. Problems in the screens themselves go to `/tmp/ephemeral-install-ui.log`.

`--answers file.json` installs without the screens: the same keys as `settings.json`, plus `passphrase`, `password`, `apps`, `confirmDisk` (the disk's name, such as `vda`, as the screens make you type it) and optionally `hardwareConfig`. The install test uses it, so every release checks the passphrase handoff: the test unlocks the installed disk with the passphrase from that file.

To go through every screen without touching a disk:

```bash
nix run path:.#installer -- --dry-run
```

Run it in a terminal of at least 80×24. Under `TERM=linux` it uses only characters the console fonts have; elsewhere it uses a few more. With at least 30 rows and 90 columns it draws a frame. `--no-effects`, `EPHEMERAL_NO_EFFECTS=1` or `e` on the welcome screen turns off the animations; the variable doesn't get through the ISO's `sudo` wrapper, the flag does. For a dry run with made-up hardware, point `EPHEMERAL_LSBLK_JSON` at a file in the format of `lsblk -J -b -o NAME,PATH,TYPE,SIZE,TRAN,MODEL,MOUNTPOINTS,FSTYPE,LABEL,PARTLABEL`; `EPHEMERAL_DRY_RUN_FAIL=1` ends the pretend install with an error. A dry run starts a pretend `install.sh` the same way as the real one, detached and with the passphrase on stdin, so everything but the disk work gets exercised.

Each screen has a file of its own with its keys and what it draws (`disk.go`, `review.go` and so on), and `model.go` moves between them. `nav_test.go` goes through the screens with key presses: going back, the steps that get skipped, changing an answer from the review, the typed disk name, and leaving an install running and picking it up again.

The welcome screen shows the installer's version, from the newest `## X.Y.Z` heading in `CHANGELOG.md`, and the config release it installs.

`nix build path:.#checks.x86_64-linux.installer` builds it and runs the Go tests. After changing a Go dependency, set `vendorHash` in `installer/package.nix` to the hash the build reports.
