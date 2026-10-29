{
  flake.modules.nixos.base =
    { lib, ... }:
    let
      inherit (lib) mkOption types;
      defaults = import ./_defaults.nix;
      flag = description: {
        type = types.bool;
        inherit description;
      };
    in
    {
      options.ephemeral = {
        settings =
          lib.mapAttrs
            (
              name: spec:
              mkOption (spec // lib.optionalAttrs (name != "username") { default = defaults.${name}; })
            )
            {
              hostname = {
                type = types.nonEmptyStr;
                description = "Hostname.";
              };
              username = {
                type = types.strMatching "[a-z_][a-z0-9_-]{0,31}";
                description = "The one login user.";
              };
              timeZone = {
                type = types.nonEmptyStr;
                description = "Time zone, e.g. Europe/Berlin.";
              };
              locale = {
                type = types.strMatching "[A-Za-z_]+\\.UTF-8";
                description = "System language, e.g. de_DE.UTF-8.";
              };
              keyboardLayout = {
                type = types.strMatching "[a-z]{2,3}(,[a-z]{2,3})*";
                description = "XKB layout, also used for the console and the LUKS prompt, e.g. de or us,de.";
              };
              keyboardVariant = {
                type = types.strMatching "[a-z0-9_-]*(,[a-z0-9_-]*)*";
                description = "XKB variant, e.g. nodeadkeys. Empty for the default.";
              };
              diskDevice = {
                type = types.nonEmptyStr;
                description = "Whole disk the system is installed on.";
              };
              swapSize = {
                type = types.nonEmptyStr;
                description = "Swap file size, e.g. 16G.";
              };
              desktop = {
                type = types.enum [
                  "plasma"
                  "gnome"
                  "niri"
                ];
                description = "Desktop environment.";
              };
              nvidia = {
                type = types.enum [
                  "none"
                  "open"
                  "legacy"
                ];
                description = "NVIDIA driver: open for RTX 20 / GTX 16 and newer, legacy (580) for GTX 900 and 10 series.";
              };
              extras = flag "Extra apps and tools: Steam, VS Code, Brave, VLC, atuin, fzf, bat, comma and more.";
              laptop = flag "Laptop extras: thermald on Intel, battery reporting, sensors.";
              fingerprint = flag "Fingerprint login and sudo through fprintd.";
              useChezmoi = flag "chezmoi from /persist/etc/chezmoi-repo writes the dotfiles instead of Home Manager.";
              useControlD = flag "DNS over TLS to Control D; the id is in /persist/etc/controld-resolver.";
              useTailscale = flag "Tailscale VPN.";
              useFlakePassCli = flag "Build modules/packages/_pass-cli.nix instead of nixpkgs proton-pass-cli.";
              usePassSettings = flag "Merge settings.from-pass.json.";
              passSettingsVault = {
                type = types.str;
                description = "Proton Pass vault for settings-from-pass.";
              };
              passSettingsItem = {
                type = types.str;
                description = "Proton Pass item for settings-from-pass.";
              };
              usePassSshAgent = flag "Proton Pass SSH agent at login, serving the keys in passSshVault.";
              passSshVault = {
                type = types.nonEmptyStr;
                description = "Proton Pass vault the SSH agent and ssh-pubs-from-pass read keys from.";
              };
            };

        pickedApps = mkOption {
          type = types.listOf types.str;
          default = [ ];
          description = "Apps from apps.json, picked with ephemeral-apps.";
        };

        smbShares = mkOption {
          type = types.listOf (
            types.submodule {
              options = {
                server = mkOption { type = types.strMatching "[A-Za-z0-9._:-]+"; };
                share = mkOption { type = types.strMatching "[A-Za-z0-9._$ -]+"; };
                mountPoint = mkOption {
                  type = types.strMatching "/(mnt|media|home/[a-z_][a-z0-9_-]*)/[A-Za-z0-9._/-]+";
                };
                credentials = mkOption { type = types.strMatching "/persist/etc/smb/[A-Za-z0-9_-]+\\.creds"; };
              };
            }
          );
          default = [ ];
          description = "CIFS mounts written by smb-setup to smb-shares.json.";
        };
      };
    };
}
