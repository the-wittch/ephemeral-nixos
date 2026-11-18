# Manual install

The same steps as `install.sh`, by hand, from the NixOS ISO. Use the installer if you can: these steps only check the disk where you run its checks yourself, below. Unplug every disk you don't mean to erase.

Start from a clone of a release, checked as in the [README](../README.md#from-a-release-recommended), and run everything below in it. The NixOS ISO doesn't turn on flakes, so take the installer's Nix settings first:

```bash
export NIX_CONFIG="$(cat scripts/nix.conf)"
```

Write `settings.json` with your own username and disk; [settings.json](settings.md) lists the other keys:

```bash
printf '{\n  "username": "nixuser",\n  "diskDevice": "/dev/vda",\n  "desktop": "gnome",\n  "profile": "minimal"\n}\n' > settings.json
nano settings.json
```

Run the installer's checks on that disk. They refuse a partition, a disk that's in use or holds an installer image, one that's part of an LVM volume group, ZFS pool or RAID array, and another disk with partitions from an earlier install; nothing prints when it passes:

```bash
bash -c 'SCRIPT_DIR=scripts; source scripts/common.sh; validate_disk "$1" && validate_partlabels "$1"' _ /dev/vda
```

> [!WARNING]
> The next command erases the whole disk named in `diskDevice`. It lists the disk and asks you to type `yes`, then asks for the new LUKS passphrase.

```bash
sudo $(nix build --no-link --print-out-paths path:.#nixosConfigurations.nixos.config.system.build.destroyFormatMount)/bin/disko-destroy-format-mount
sudo btrfs subvolume list /mnt
```

The subvolume list should include `root-blank`. Then the hardware, the password and the flake, again with your own username in place of `nixuser`:

```bash
sudo nixos-generate-config --no-filesystems --root /mnt --show-hardware-config > hardware-configuration.nix
sudo mkdir -p /mnt/persist/passwords /mnt/persist/etc/ssh /mnt/persist/etc/nixos /mnt/persist/home/nixuser
sudo chmod 700 /mnt/persist/passwords /mnt/persist/home/nixuser
mkpasswd -m yescrypt | sudo tee /mnt/persist/passwords/nixuser
sudo chmod 600 /mnt/persist/passwords/nixuser
sudo cp flake.nix flake.lock .gitignore settings.json hardware-configuration.nix /mnt/persist/etc/nixos/
sudo cp -r /mnt/persist/etc/nixos /mnt/persist/etc/nixos-install
sudo chmod -R a-w /mnt/persist/etc/nixos-install
sudo nixos-install --no-root-passwd --flake path:/mnt/persist/etc/nixos#nixos
```

If the ISO has no `mkpasswd`, run `nix shell nixpkgs#mkpasswd` first. The home folder can belong to root for now; the first boot hands it to you. `/persist/etc/nixos-install` is the copy `ephemeral-update --as-installed` goes back to.
