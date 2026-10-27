# Install cheat sheet

The short version of the [README](./README.md). The tag and commit for the current release are in [`RELEASE`](./RELEASE).

From a release, with menus:

```bash
TAG=v2.0.0
REV=d422ad35cb81287c35450b1ee1e1f2551b276690
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

Without menus, onto a whole disk:

```bash
curl -fsSL https://raw.githubusercontent.com/the-wittch/ephemeral-nixos/v2.0.0/scripts/bootstrap.sh | sudo env TAG=v2.0.0 EXPECTED_REV=d422ad35cb81287c35450b1ee1e1f2551b276690 SWAP_SIZE=4G bash -s -- /dev/vda
```

Rebuilding later:

```bash
cd /persist/etc/nixos
sudo nixos-rebuild switch --flake path:.#nixos
```
