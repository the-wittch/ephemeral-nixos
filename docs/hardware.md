# Hardware

## Required

- An x86_64 PC that boots in UEFI mode. The installer stops on legacy BIOS boots.
- Secure Boot turned off in the firmware settings.
- One whole disk for the system. It's erased; there's no dual boot.
- Enough RAM to build the system. The install test runs with 6 GB; less hasn't been tried.

## What's tested

- **Every release:** CI installs from the ISO's live system onto a QEMU/KVM virtual machine with UEFI, then unlocks and boots the installed disk. The config's boot tests also boot both profiles in a VM.
- **By hand:** the maintainer has installed from the ISO on two laptops.
- **Not tested:** NVIDIA drivers, laptops with two GPUs, and Wi-Fi in a VM (CI checks only that the ISO has what Wi-Fi needs).

If yours works or doesn't, an [issue](https://github.com/the-wittch/ephemeral-nixos/issues/new/choose) with the output of `lspci -nn` helps the next person.

## Detected during the install

- **Laptops:** a battery turns on the laptop extras (battery reporting, sensors, and thermald on Intel).
- **Fingerprint readers:** a USB reader from Goodix, Synaptics, Validity, Elan, Egis, FocalTech, AuthenTec, UPEK or FPC is offered for login and sudo.
- **NVIDIA cards:** the installer shows the card and asks which driver to use. Laptops with an Intel or AMD GPU as well also need [PRIME](https://wiki.nixos.org/wiki/NVIDIA) set up in a module of your own.
- **Everything else** (disk controllers, Wi-Fi chips, CPU microcode) comes from `nixos-generate-config` and the redistributable firmware NixOS ships.

## Thunderbolt and USB4

Devices have to be approved once, in the desktop's settings or with `boltctl`. Approvals are kept across reboots. The IOMMU limits what a plugged-in device can read from memory.
