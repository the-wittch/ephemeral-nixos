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
      appsFile = host.root + "/apps.json";
      picked = if builtins.pathExists appsFile then lib.importJSON appsFile else [ ];
      known =
        lib.throwIfNot (builtins.isList picked && lib.all builtins.isString picked)
          "apps.json has to be a list of app names"
          (builtins.filter (id: config.ephemeral.apps ? ${id}) picked);
      hardware = host.root + "/hardware-configuration.nix";
      system = inputs.nixpkgs.lib.nixosSystem {
        modules = [
          config.flake.modules.nixos.pc
          config.flake.modules.nixos.${settings.desktop}
          config.flake.modules.nixos.${if settings.useChezmoi then "chezmoi" else "dotfiles"}
          {
            ephemeral.settings = settings;
            ephemeral.smbShares = readData (host.root + "/smb-shares") [ ];
            ephemeral.pickedApps = picked;
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
        ++ map (name: config.flake.modules.nixos.${name}) (
          lib.attrNames (
            lib.filterAttrs (_: on: on) {
              controld = settings.useControlD;
              nvidia = settings.nvidia != "none";
              extras = settings.extras;
              laptop = settings.laptop;
              fingerprint = settings.fingerprint;
              tailscale = settings.useTailscale;
              passSshAgent = settings.usePassSshAgent;
            }
          )
        )
        ++ map (id: config.ephemeral.apps.${id}.nixos) known
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
