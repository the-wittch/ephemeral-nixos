# Architecture

The rules every change to the installer is checked against. The ones for the system it installs, and for the project as a whole, are in the config's [ARCHITECTURE.md](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/ARCHITECTURE.md), and hold here too.

A change that breaks one of these rules needs a reason, written in the pull request and in this file.

## What it is

The installer has no NixOS modules of its own. `flake.nix` is the machine flake that gets installed, pinned to a release of the config. This repo adds only the ISO, the installer's screens, `install.sh` and the install test.

| piece | what it does |
|---|---|
| `installer/` (Go) | asks the questions, checks the answers, shows the progress, and runs `install.sh` |
| `scripts/install.sh` | does all the disk work: checks, partitioning, installing |
| `scripts/common.sh` | the checks and the clean-up both of them use |
| `modules/iso.nix` | the ISO |
| `modules/tests/` | the install test and the script tests |

## Stages

`install.sh` runs in this order, and `::stage` lines tell the screens which stage is running:

1. **prepare:** check the answers and the disk, write the machine's flake to a temporary folder, evaluate the whole system, build the partitioning script. Nothing outside `/tmp` is written.
2. **confirm:** show what the disk becomes, and have the disk's name typed.
3. **partition:** record the disk in `/run/ephemeral-install.state`, then partition, encrypt and mount it.
4. **hardware**, then **install:** detect the hardware, write the secrets and the flake to `/persist`, install the system and the boot loader.

## Rules for anything that touches a disk

1. **Nothing is erased before everything that can be checked has been checked**: the answers, that the disk is a whole disk, not in use, not the installer stick, not part of an LVM volume group, ZFS pool or RAID array that erasing it would remove from other disks, with no other disk carrying this layout's partition names, room for the swap file, and that the system evaluates.
2. **The disk is erased only after its name has been typed** in the screens or the plain prompts, or given as `confirmDisk` with `--answers`. `CONFIRM=YES` for unattended installs is the only exception, and it's documented.
3. **What gets erased is the disk that was confirmed.** Its serial, WWN, size and model are recorded when it's confirmed, and checked again right before erasing.
4. **Only one install runs at a time.** `install.sh` holds a lock for as long as it and everything it started are running.
5. **The installer only undoes what it recorded opening.** `teardown` closes the swap file, the mount and `cryptroot` from an earlier attempt on the recorded disk, and refuses anything else.
6. **No check is weakened to make installing easier.** An override, such as `FORCE_DISK` or `ALLOW_SAME_PARTLABELS`, is explicit, named after what it allows, and documented.
7. **A failure says how far the install got**: nothing changed yet, the disk half formatted, or the disk formatted and mounted at `/mnt`. Trying again starts over safely from any of them.
8. **Logs are kept.** The attempt before is kept as `.1`, and a log never holds the passphrase or the password.
9. **The passphrase goes to the partitioning only on standard input, and the password only as a hash.**

## Tests

1. **The install test runs before every release.** It covers a failed attempt and the retry after it, a config that doesn't evaluate leaving the disk untouched, the passphrase unlocking the installed disk, and the wipe after a reboot. The ISO image itself is booted to the installer's welcome screen before it's attached to a release.
2. **`common.sh`'s checks and `teardown` are tested against fake disk commands** on every pull request, including each refusal.
3. **A new check on a disk comes with a test of it refusing**, not only of it passing.
