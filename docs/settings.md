# settings.json and unfree packages

### settings.json

The installer writes your answers, for example:

```json
{
  "hostname": "nixos",
  "username": "nixuser",
  "diskDevice": "/dev/nvme0n1",
  "swapSize": "16G",
  "timeZone": "Europe/Berlin",
  "desktop": "gnome",
  "nvidia": "none",
  "keyboardLayout": "de",
  "keyboardVariant": "",
  "locale": "de_DE.UTF-8",
  "extras": false,
  "laptop": true,
  "fingerprint": false,
  "profile": "full",
  "useOpenSnitch": true,
  "useControlD": false,
  "usePassSshAgent": false
}
```

The config's [`settings.example.json`](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/settings.example.json) lists every setting with its default; the [config's settings page](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/docs/settings.md) explains each one. `diskDevice` is the whole disk, not a partition. `desktop` is `plasma`, `gnome` or `niri`. `nvidia` is `none`, `open` or `legacy`. Installs older than v1.3 have a `settings.nix` instead; it still loads, and the rebuild prints the command that converts it.

Apps picked during the install go in `apps.json`, next to it, the same file `ephemeral-apps` writes.

### Unfree packages

Nothing unfree builds unless it's listed by name, next to the package that needs it:

```nix
ephemeral.unfreePackages = [ "steam" "steam-unwrapped" ];
```

A package that isn't listed fails the build and tells you the name to add.
