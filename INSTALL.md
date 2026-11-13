# Install cheat sheet

The short version of the [README](./README.md). The tag and commit for the current release are in [`RELEASE`](./RELEASE).

Easiest: download the `.iso` from the latest [release](https://github.com/the-wittch/ephemeral-nixos/releases), check it with `sha256sum -c SHA256SUMS`, write it to a USB stick and boot it. The installer starts on its own.

From the NixOS ISO, from a release, with menus:

```bash
TAG=v3.7.5
REV=da53f6fb345c6c633c4590584dcc44e55424eb03
git clone --branch "$TAG" --depth 1 https://github.com/the-wittch/ephemeral-nixos.git ~/nix
cd ~/nix
[[ $(git rev-parse HEAD) == "$REV" ]] || echo "hash mismatch, stop here"
sudo ./scripts/tui.sh
```

From main:

```bash
git clone https://github.com/the-wittch/ephemeral-nixos.git ~/nix
cd ~/nix
sudo ./scripts/tui.sh
```

Without menus, onto a whole disk. The user is `nixuser` unless you set `INSTALL_USERNAME`:

```bash
curl -fsSL https://raw.githubusercontent.com/the-wittch/ephemeral-nixos/v3.7.5/scripts/bootstrap.sh | sudo env TAG=v3.7.5 EXPECTED_REV=da53f6fb345c6c633c4590584dcc44e55424eb03 SWAP_SIZE=4G bash -s -- /dev/vda
```

Rebuilding later:

```bash
cd /persist/etc/nixos
sudo nixos-rebuild switch --flake path:.#nixos
```
