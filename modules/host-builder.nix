{
  config,
  inputs,
  lib,
  ...
}:

let
  readData = import ./settings/_read-data.nix { inherit lib; };
  loadSettings = flakeRoot: import ./settings/_load.nix { inherit lib flakeRoot; };

  build =
    name: host:
    let
      settings = loadSettings host.root;
      hardware = host.root + "/hardware-configuration.nix";
      system = inputs.nixpkgs.lib.nixosSystem {
        modules = [
          config.flake.modules.nixos.pc
          config.flake.modules.nixos.${settings.desktop}
          config.flake.modules.nixos.${if settings.useChezmoi then "chezmoi" else "dotfiles"}
          {
            ephemeral.settings = settings;
            ephemeral.smbShares = readData (host.root + "/smb-shares") [ ];
          }
          (
            if builtins.pathExists hardware then
              hardware
            else
              {
                warnings = [
                  "no hardware-configuration.nix in ${toString host.root}; building without initrd modules or microcode"
                ];
              }
          )
        ]
        ++ lib.optional settings.useControlD config.flake.modules.nixos.controld
        ++ host.modules;
      };
    in
    {
      ${name} = system;
    }
    // lib.optionalAttrs (settings.hostname != name) { ${settings.hostname} = system; };
in
{
  options.ephemeral.hosts = lib.mkOption {
    description = "Machines. Each reads settings.json, smb-shares.json and hardware-configuration.nix from its root.";
    default = { };
    type = lib.types.attrsOf (
      lib.types.submodule {
        options = {
          root = lib.mkOption {
            type = lib.types.path;
            description = "Directory holding the machine files, usually the flake root.";
          };
          modules = lib.mkOption {
            type = lib.types.listOf lib.types.deferredModule;
            default = [ ];
            description = "Extra NixOS modules for this host only.";
          };
        };
      }
    );
  };

  config.flake.nixosConfigurations = lib.concatMapAttrs build config.ephemeral.hosts;
}
