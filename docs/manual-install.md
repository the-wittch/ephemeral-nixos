# Manual install

```bash
printf '{\n  "username": "nixuser",\n  "diskDevice": "/dev/vda",\n  "desktop": "gnome",\n  "profile": "minimal"\n}\n' > settings.json
nano settings.json
sudo $(nix build --no-link --print-out-paths path:.#nixosConfigurations.nixos.config.system.build.destroyFormatMount)/bin/disko-destroy-format-mount
sudo btrfs subvolume list /mnt
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

The subvolume list should include `root-blank`. Use your own username in place of `nixuser`. The home folder can belong to root for now; the first boot hands it to you. `/persist/etc/nixos-install` is the copy `ephemeral-update --as-installed` goes back to.
