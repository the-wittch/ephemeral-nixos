# nix build path:.#iso
{ inputs, lib, ... }:
let
  mocha = {
    base = "1e1e2e";
    mauve = "cba6f7";
    blue = "89b4fa";
    red = "f38ba8";
  };

  source = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../flake.nix
      ../flake.lock
      ../.gitignore
      ../hardware-configuration.example.nix
      ../scripts
    ];
  };

  iso =
    { config, pkgs, ... }:
    let
      splash =
        width: height:
        pkgs.runCommand "ephemeral-splash-${toString width}.png"
          {
            nativeBuildInputs = [ pkgs.librsvg ];
            FONTCONFIG_FILE = pkgs.makeFontsConf { fontDirectories = [ pkgs.jetbrains-mono ]; };
          }
          ''
            rsvg-convert -w ${toString width} -h ${toString height} ${inputs.ephemeral}/assets/wallpaper.svg -o $out
          '';

      install = pkgs.writeShellApplication {
        name = "ephemeral-install";
        runtimeInputs = with pkgs; [
          gum
          curl
          networkmanager
        ];
        text = ''
          gum style --border rounded --border-foreground "#${mocha.mauve}" --padding "1 4" --margin "1 0" \
            "$(gum style --bold --foreground "#${mocha.mauve}" "Ephemeral Nix")" \
            "NixOS on an encrypted disk, with a root that's wiped on every boot"

          until curl -fsS --max-time 5 -o /dev/null https://cache.nixos.org/nix-cache-info; do
            gum confirm "No internet connection yet. Set one up?" --affirmative "Network setup" --negative "Shell" || exit 0
            nmtui
            gum spin --title "Waiting for the connection..." -- nm-online -q -t 20 || true
          done

          exec sudo --preserve-env=GUM_CHOOSE_CURSOR_FOREGROUND,GUM_CHOOSE_SELECTED_FOREGROUND,GUM_CHOOSE_HEADER_FOREGROUND,GUM_INPUT_CURSOR_FOREGROUND,GUM_CONFIRM_SELECTED_BACKGROUND \
            /etc/ephemeral-nixos/scripts/tui.sh
        '';
      };
    in
    {
      imports = [ "${inputs.nixpkgs}/nixos/modules/installer/cd-dvd/installation-cd-minimal.nix" ];

      nixpkgs.hostPlatform = "x86_64-linux";
      system.nixos.distroName = "Ephemeral Nix";
      image.baseName = lib.mkForce "ephemeral-nix-${config.system.nixos.release}";
      isoImage = {
        volumeID = "EPHEMERAL_NIX";
        splashImage = splash 800 600;
        efiSplashImage = splash 1920 1080;
        grubTheme = null;
      };

      console.colors = [
        mocha.base
        mocha.red
        "a6e3a1"
        "f9e2af"
        mocha.blue
        "f5c2e7"
        "94e2d5"
        "bac2de"
        "585b70"
        mocha.red
        "a6e3a1"
        "f9e2af"
        mocha.blue
        "f5c2e7"
        "94e2d5"
        "a6adc8"
      ];

      environment.variables = {
        GUM_CHOOSE_CURSOR_FOREGROUND = "#${mocha.mauve}";
        GUM_CHOOSE_SELECTED_FOREGROUND = "#${mocha.mauve}";
        GUM_CHOOSE_HEADER_FOREGROUND = "#${mocha.blue}";
        GUM_INPUT_CURSOR_FOREGROUND = "#${mocha.mauve}";
        GUM_CONFIRM_SELECTED_BACKGROUND = "#${mocha.mauve}";
      };

      environment.etc."ephemeral-nixos".source = source;
      environment.systemPackages = [
        install
        pkgs.gum
        pkgs.git
      ];

      services.getty.helpLine = lib.mkForce ''

        Run `ephemeral-install` to install Ephemeral Nix. `nmtui` sets up Wi-Fi.
      '';
      programs.bash.loginShellInit = ''
        if [ "$(tty)" = /dev/tty1 ] && [ -z "''${EPHEMERAL_INSTALL_SHOWN:-}" ]; then
          export EPHEMERAL_INSTALL_SHOWN=1
          ephemeral-install || true
        fi
      '';
    };
in
{
  perSystem =
    { system, ... }:
    {
      packages.iso = (inputs.nixpkgs.lib.nixosSystem { modules = [ iso ]; }).config.system.build.isoImage;
    };
}
