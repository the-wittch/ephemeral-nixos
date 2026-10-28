{ lib, ... }:
let
  inherit (lib) mkOption types;
  text = types.strMatching "[^,]+";
in
{
  options.ephemeral.apps = mkOption {
    default = { };
    description = "Apps ephemeral-apps offers. Picks are kept in apps.json next to settings.json.";
    type = types.attrsOf (
      types.submodule (
        { name, ... }:
        {
          options = {
            label = mkOption {
              type = text;
              default = name;
            };
            category = mkOption {
              type = types.enum [
                "Browsers"
                "Communication"
                "Office"
                "Media"
                "Graphics"
                "Gaming"
                "Development"
                "Security"
                "Utilities"
              ];
            };
            description = mkOption { type = text; };
            package = mkOption {
              type = types.nullOr types.str;
              default = null;
              description = "Attribute path in pkgs, e.g. kdePackages.kdenlive.";
            };
            flatpak = mkOption {
              type = types.nullOr types.str;
              default = null;
              description = "Flathub id, installed for the user.";
            };
            unfree = mkOption {
              type = types.listOf types.str;
              default = [ ];
              description = "Names for ephemeral.unfreePackages.";
            };
            warn = mkOption {
              type = types.nullOr text;
              default = null;
              description = "Shown before the app is added.";
            };
            nixos = mkOption {
              type = types.deferredModule;
              default = { };
            };
            homeManager = mkOption {
              type = types.deferredModule;
              default = { };
            };
          };
        }
      )
    );
  };
}
