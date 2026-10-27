# Install cheat sheet

Full context: [README](./README.md). Pin values: [`RELEASE`](./RELEASE).

**Pinned TUI** (recommended):

```bash
TAG=v1.5.1
REV=b9699fdd4fb503fec24525763d019a50f8ef354f

git clone --branch "$TAG" --depth 1 https://github.com/the-wittch/ephemeral-nixos.git ~/nix
cd ~/nix
[[ $(git rev-parse HEAD) == "$REV" ]] || { echo "hash mismatch"; exit 1; }
sudo ./scripts/tui.sh
```

**Floating main:**

```bash
git clone https://github.com/the-wittch/ephemeral-nixos.git ~/nix
cd ~/nix && sudo ./scripts/tui.sh
```

**Plain** (whole disk):

```bash
curl -fsSL https://raw.githubusercontent.com/the-wittch/ephemeral-nixos/v1.5.1/scripts/bootstrap.sh \
  | sudo env TAG=v1.5.1 EXPECTED_REV=b9699fdd4fb503fec24525763d019a50f8ef354f SWAP_SIZE=4G \
    bash -s -- /dev/vda
```

Rebuild later:

```bash
cd /persist/etc/nixos
sudo nixos-rebuild switch --flake path:.#nixos
```
