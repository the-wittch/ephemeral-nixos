{
  flake.modules.nixos.base =
    { lib, ... }:
    let
      inherit (lib) mkOption types;
      str =
        description:
        mkOption {
          inherit description;
          type = types.nonEmptyStr;
        };
      flag =
        description:
        mkOption {
          inherit description;
          type = types.bool;
          default = false;
        };
    in
    {
      options.ephemeral = {
        settings = {
          hostname = str "Hostname.";
          username = mkOption {
            type = types.strMatching "[a-z_][a-z0-9_-]{0,31}";
            description = "The one login user.";
          };
          diskDevice = str "Whole disk the system is installed on.";
          swapSize = str "Swap file size, e.g. 16G.";
          timeZone = str "Time zone, e.g. Europe/Berlin.";
          desktop = mkOption {
            type = types.enum [
              "plasma"
              "gnome"
              "niri"
            ];
            description = "Desktop environment.";
          };
          useControlD = flag "DNS over TLS to Control D; the id is in /persist/etc/controld-resolver.";
          useChezmoi = flag "chezmoi from /persist/etc/chezmoi-repo writes the dotfiles instead of Home Manager.";
          useFlakePassCli = flag "Build modules/packages/_pass-cli.nix instead of nixpkgs proton-pass-cli.";
          usePassSettings = flag "Merge settings.from-pass.json.";
          passSettingsVault = mkOption {
            type = types.str;
            default = "Personal";
            description = "Proton Pass vault for settings-from-pass.";
          };
          passSettingsItem = mkOption {
            type = types.str;
            default = "nixos-settings";
            description = "Proton Pass item for settings-from-pass.";
          };
        };

        smbShares = mkOption {
          type = types.listOf (types.attrsOf types.str);
          default = [ ];
          description = "CIFS mounts written by smb-setup to smb-shares.json.";
        };
      };
    };
}
