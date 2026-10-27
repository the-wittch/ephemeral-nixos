let
  uri = "file:///run/current-system/sw/share/backgrounds/ephemeral/wallpaper.png";
in
{
  flake.modules.nixos.pc =
    {
      config,
      lib,
      pkgs,
      ...
    }:
    {
      options.ephemeral.wallpaper = lib.mkOption {
        type = lib.types.package;
        readOnly = true;
        description = "The rendered wallpaper; the png is share/backgrounds/ephemeral/wallpaper.png.";
        # nixos logo by simon frankau, tim cuthbertson and daniel baker. cc-by-4.0, github.com/NixOS/branding
        default =
          pkgs.runCommand "ephemeral-wallpaper"
            {
              nativeBuildInputs = [ pkgs.librsvg ];
              FONTCONFIG_FILE = pkgs.makeFontsConf { fontDirectories = [ pkgs.jetbrains-mono ]; };
            }
            ''
              mkdir -p $out/share/backgrounds/ephemeral
              rsvg-convert -w 3840 -h 2160 ${../assets/wallpaper.svg} \
                -o $out/share/backgrounds/ephemeral/wallpaper.png
            '';
      };

      config = {
        environment.systemPackages = [ config.ephemeral.wallpaper ];
        environment.pathsToLink = [ "/share/backgrounds" ];
      };
    };

  flake.modules.nixos.gnome = {
    services.desktopManager.gnome.extraGSettingsOverrides = ''
      [org.gnome.desktop.background]
      picture-uri='${uri}'
      picture-uri-dark='${uri}'
      picture-options='zoom'

      [org.gnome.desktop.screensaver]
      picture-uri='${uri}'
    '';

    programs.dconf.profiles.gdm.databases = [
      {
        settings."org/gnome/desktop/background" = {
          picture-uri = uri;
          picture-uri-dark = uri;
          picture-options = "zoom";
        };
      }
    ];
  };

  flake.modules.nixos.plasma = {
    services.displayManager.plasma-login-manager.settings = {
      Greeter.WallpaperPluginId = "org.kde.image";
      "Greeter][Wallpaper][org.kde.image][General".Image = uri;
    };
  };
}
