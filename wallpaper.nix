{
  lib,
  pkgs,
  settings,
  ...
}:

let
  # nixos logo by tim cuthbertson, cc-by-4.0
  wallpaper =
    pkgs.runCommand "ephemeral-wallpaper"
      {
        nativeBuildInputs = [ pkgs.librsvg ];
        FONTCONFIG_FILE = pkgs.makeFontsConf { fontDirectories = [ pkgs.jetbrains-mono ]; };
      }
      ''
        mkdir -p $out/share/backgrounds/ephemeral
        rsvg-convert -w 3840 -h 2160 ${./assets/wallpaper.svg} \
          -o $out/share/backgrounds/ephemeral/wallpaper.png
      '';

  # stable across rebuilds. plasma desktop and niri: pick it in their settings
  uri = "file:///run/current-system/sw/share/backgrounds/ephemeral/wallpaper.png";

  isGnome = settings.desktop == "gnome";
  isPlasma = settings.desktop == "plasma";
in
{
  environment.systemPackages = [ wallpaper ];
  environment.pathsToLink = [ "/share/backgrounds" ];

  # default only. one picked in settings wins
  services.desktopManager.gnome.extraGSettingsOverrides = lib.mkIf isGnome ''
    [org.gnome.desktop.background]
    picture-uri='${uri}'
    picture-uri-dark='${uri}'
    picture-options='zoom'

    [org.gnome.desktop.screensaver]
    picture-uri='${uri}'
  '';

  programs.dconf.profiles.gdm.databases = lib.mkIf isGnome [
    {
      settings."org/gnome/desktop/background" = {
        picture-uri = uri;
        picture-uri-dark = uri;
        picture-options = "zoom";
      };
    }
  ];

  services.displayManager.plasma-login-manager.settings = lib.mkIf isPlasma {
    Greeter.WallpaperPluginId = "org.kde.image";
    "Greeter][Wallpaper][org.kde.image][General".Image = uri;
  };
}
