# Architecture

The rules every change to the installer is checked against. The ones for the system it installs, and for the project as a whole, are in the config's [ARCHITECTURE.md](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/ARCHITECTURE.md), and hold here too.

A change that breaks one of these rules needs a reason, written in the pull request and in this file.

## What it is

The installer adds no NixOS modules to the machines it installs. `flake.nix` is the machine flake that gets installed, pinned to a release of the config. This repo adds only the ISO, the installer's screens, `install.sh` and the install test.

| piece | what it does |
|---|---|
| `installer/` (Go) | asks the questions, checks the answers, shows the progress, and runs `install.sh` |
| `scripts/install.sh` | does all the disk work: checks, partitioning, installing |
| `scripts/common.sh` | the disk checks and the clean-up, shared by `install.sh`, `tui.sh` and a local `bootstrap.sh` |
| `scripts/nix.conf`, `scripts/fingerprint-vendors` | data both the screens and the scripts read |
| `modules/tests/` | the install test, the script tests and the contract check |
| `modules/iso.nix` | the ISO |


## What comes from the config

The installer keeps no copy of what the config owns. It reads it from the config release `flake.lock` pins, so a new installer and the config it installs always agree:

| what | from the config release | how the installer uses it |
|---|---|---|
| what each setting may be, and the allowed values | the `settings-contract` package | the screens check answers as you type; `--answers` checks the file; the Go tests check Go's matching agrees with the config's on every value in `installer/testdata/validation.json` |
| whether an answer is a setting the config accepts | the `settings` output | `install.sh` evaluates it after writing `settings.json` and stops, with nothing changed, if the config refuses a value |
| each menu: the desktops, NVIDIA drivers, profiles and DNS choices, in order, with their names and descriptions | the `settings-contract` package | the screens' menus and the plain prompts in `install.sh`, which builds the contract from `flake.lock` when the screens haven't passed it |
| the apps and which are ticked | the `app-catalog` package | the screens' app list and `apps.json` |
| the wallpaper and the welcome screen's snowflake | the `wallpaper` and `welcome-logo` packages | the ISO's boot splash and the welcome screen |
| the disk layout | the machine's `disko.devices` | `install.sh` builds disko's script from it |

What the installer keeps of its own: the wording of its other questions and its errors, and its disk checks. The screens find the NVIDIA card and fingerprint reader and identify the disk by running the same functions in `scripts/common.sh` that `install.sh` runs. A few things it has to repeat: the partition labels and `cryptroot`, needed before anything is staged; the Nix settings in the bootstrap scripts, which run before there is a clone; and the settings keys `install.sh` writes and `--answers` reads. `checks.contract` fails when any of them disagree with the pinned config release or with `scripts/nix.conf`.

## Stages

`install.sh` runs in this order, and `::stage` lines tell the screens which stage is running:

1. **prepare:** check the answers and the disk, write the machine's flake to a temporary folder, evaluate the whole system, build the partitioning script. Nothing outside `/tmp` is written. At the end it shows what the disk becomes and has the disk's name typed.
2. **partition:** record the disk in `/run/ephemeral-install.state`, then partition, encrypt and mount it.
3. **hardware**, then **install:** detect the hardware, write the secrets and the flake to `/persist`, install the system and the boot loader.

## Rules for anything that touches a disk

1. **Nothing is erased before everything that can be checked has been checked**: the answers, that the disk is a whole disk, not in use, not the installer stick, not part of an LVM volume group, ZFS pool or RAID array that erasing it would remove from other disks, with no other disk carrying this layout's partition names, room for the swap file, and that the system evaluates.
2. **The disk is erased only after its name has been typed** in the screens or the plain prompts, or given as `confirmDisk` with `--answers`. `CONFIRM=YES` for unattended installs is the only exception, and it's documented.
3. **What gets erased is the disk that was confirmed.** Its serial, WWN, size, model and physical path are recorded when it's confirmed, and checked again right before erasing.
4. **Only one install runs at a time.** `install.sh` holds a lock for as long as it and everything it started are running.
5. **The installer only undoes what it recorded opening.** `teardown` closes the swap file, the mount and `cryptroot` from an earlier attempt on the recorded disk, and refuses anything else.
6. **No check is weakened to make installing easier.** An override, such as `FORCE_DISK` or `ALLOW_SAME_PARTLABELS`, is explicit, named after what it allows, and documented.
7. **A failure says how far the install got**: nothing changed yet, the disk half formatted, or the disk formatted and mounted at `/mnt`. Trying again starts over safely from any of them.
8. **Logs are kept.** The attempt before is kept as `.1`, and a log never holds the passphrase or the password.
9. **The passphrase goes to `install.sh` only on its own file descriptor, which is read and closed before anything else runs, and from there only to disko; the password goes only as a hash.** `install.sh`'s standard input is empty in the screens, so a question it asks by mistake stops the install instead of taking the passphrase as its answer.

## Tests

1. **The install test runs before every release.** It covers a failed attempt and the retry after it, a config that doesn't evaluate leaving the disk untouched, the passphrase unlocking the installed disk, and the wipe after a reboot. The ISO image itself is booted to the installer's welcome screen before it's attached to a release.
2. **`common.sh`'s checks and `teardown` are tested against fake disk commands** on every pull request, including each refusal.
3. **A new check on a disk comes with a test of it refusing**, not only of it passing.
4. **What the installer repeats from the config is checked against the config release it pins**, by `checks.contract` and the Go tests, on every pull request and on every `follow config` run. A config release that changes a name the installer depends on fails there, before the installer moves to it.
