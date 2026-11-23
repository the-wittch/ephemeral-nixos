# settings.json and unfree packages

## settings.json

The installer writes your answers to `settings.json` in `/persist/etc/nixos`. The config owns what goes in it: its [`settings.example.json`](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/settings.example.json) lists every setting with its default, and its [settings page](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/docs/settings.md) explains each one. Installs older than v1.3 have a `settings.nix` instead. Config v4.0.0 and later don't read it; the rebuild stops and prints the command that converts it to `settings.json`.

Apps picked during the install go in `apps.json`, next to it, the same file `ephemeral apps` writes.

## Unfree packages

Nothing unfree builds unless it's listed by name. Apps from `ephemeral apps` list their own. For anything else, list it in the module that installs it, such as `/persist/etc/nixos/modules/terraform.nix`:

```nix
{
  flake.modules.nixos.pc =
    { pkgs, ... }:
    {
      environment.systemPackages = [ pkgs.terraform ];
      ephemeral.unfreePackages = [ "terraform" ];
    };
}
```

A package that isn't listed fails the build, and the error names it: the name to add is the one before the version.
