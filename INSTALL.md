# Install cheat sheet

The short version of the [README](./README.md). The tag and commit for the current release are in [`RELEASE`](./RELEASE).

From a release, with menus:

```bash
TAG=v2.1.0
REV=0de5c8023c8438942bdc8f6e63ef1a8db55bffb4
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
curl -fsSL https://raw.githubusercontent.com/the-wittch/ephemeral-nixos/v2.1.0/scripts/bootstrap.sh | sudo env TAG=v2.1.0 EXPECTED_REV=0de5c8023c8438942bdc8f6e63ef1a8db55bffb4 SWAP_SIZE=4G bash -s -- /dev/vda
```

Rebuilding later:

```bash
cd /persist/etc/nixos
sudo nixos-rebuild switch --flake path:.#nixos
```
