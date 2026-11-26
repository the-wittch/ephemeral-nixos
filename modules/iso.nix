# nix build path:.#iso
{
  config,
  inputs,
  lib,
  withSystem,
  ...
}:
let
  mocha = {
    base = "1e1e2e";
    mauve = "cba6f7";
    blue = "89b4fa";
    red = "f38ba8";
  };

  live =
    { pkgs, ... }:
    let
      installer = withSystem pkgs.stdenv.hostPlatform.system ({ config, ... }: config.packages.installer);

      install = pkgs.writeShellApplication {
        name = "ephemeral-install";
        text = ''
          exec sudo ${lib.getExe installer} "$@"
        '';
      };
    in
    {
      system.nixos.distroName = "Ephemeral Nix";

      # 24px Terminus keeps the installer readable on large screens
      console.font = "ter-v24n";
      console.packages = [ pkgs.terminus_font ];

      console.colors = [
        mocha.base
        mocha.red
        "a6e3a1"
        "f9e2af"
        mocha.blue
        mocha.mauve
        "94e2d5"
        "bac2de"
        "585b70"
        mocha.red
        "a6e3a1"
        "f9e2af"
        mocha.blue
        mocha.mauve
        "94e2d5"
        "a6adc8"
      ];

      environment.etc."ephemeral-nixos".source = "${installer}/share/ephemeral-nixos";
      environment.etc."ephemeral-nixos-lists".source = "${installer}/share/ephemeral-nixos-lists";
      environment.systemPackages = [
        install
        pkgs.git
      ];

      # A Steam Deck's or an Ally's controller moves around the screens and, on a handheld's
      # passphrase fields, types the passphrase. A pad that isn't a known one is left alone.
      boot.kernelModules = [ "uinput" ];
      # otherwise the Deck's controller sends no button presses until its menu button is held
      boot.kernelParams = [ "hid_steam.lizard_mode=0" ];
      systemd.services.handheld-input = {
        description = "Controller buttons as keys for the installer";
        wantedBy = [ "multi-user.target" ];
        serviceConfig = {
          ExecStart = "${
            lib.getExe inputs.ephemeral.packages.${pkgs.stdenv.hostPlatform.system}.handheld-input
          } menu";
          Restart = "on-failure";
          DevicePolicy = "closed";
          DeviceAllow = [
            "/dev/uinput rw"
            "/dev/tty0 r"
            "char-input r"
          ];
          CapabilityBoundingSet = [ "" ];
          NoNewPrivileges = true;
          ProtectSystem = "strict";
          ProtectHome = true;
          PrivateTmp = true;
          PrivateNetwork = true;
          RestrictAddressFamilies = [ "none" ];
          ProtectKernelTunables = true;
          ProtectKernelModules = true;
          ProtectKernelLogs = true;
          ProtectControlGroups = true;
          ProtectClock = true;
          ProtectHostname = true;
          RestrictNamespaces = true;
          RestrictRealtime = true;
          RestrictSUIDSGID = true;
          LockPersonality = true;
          MemoryDenyWriteExecute = true;
          SystemCallArchitectures = "native";
          SystemCallFilter = [ "@system-service" ];
          SystemCallErrorNumber = "EPERM";
        };
      };

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
  image =
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
            rsvg-convert -w ${toString width} -h ${toString height} ${
              inputs.ephemeral.packages.${pkgs.stdenv.hostPlatform.system}.wallpaper
            } -o $out
          '';
    in
    {
      imports = [ "${inputs.nixpkgs}/nixos/modules/installer/cd-dvd/installation-cd-minimal.nix" ];

      nixpkgs.hostPlatform = "x86_64-linux";
      image.baseName = lib.mkForce "ephemeral-nix-${config.system.nixos.release}";
      isoImage = {
        volumeID = "EPHEMERAL_NIX";
        splashImage = splash 800 600;
        efiSplashImage = splash 1920 1080;
        grubTheme = null;
      };
    };
  system = inputs.nixpkgs.lib.nixosSystem {
    modules = [
      image
      config.flake.modules.nixos.iso
    ];
  };
  iso = system.config;
in
{
  flake.modules.nixos.iso = live;

  perSystem =
    { pkgs, ... }:
    {
      packages.iso = iso.system.build.isoImage;

      # test VMs turn Wi-Fi off, so the install test can't check it
      checks.iso-eval =
        assert lib.assertMsg iso.networking.networkmanager.enable "the ISO lost NetworkManager";
        assert lib.assertMsg iso.networking.wireless.enable
          "the ISO lost wpa_supplicant, which NetworkManager needs for Wi-Fi";
        assert lib.assertMsg (iso.environment.etc ? "ephemeral-nixos") "the ISO lost the installer scripts";
        assert lib.assertMsg (
          iso.environment.etc ? "ephemeral-nixos-lists"
        ) "the ISO lost the keyboard and language lists";
        assert lib.assertMsg (lib.any (
          p: p.name or "" == "ephemeral-install"
        ) iso.environment.systemPackages) "the ISO lost ephemeral-install";
        pkgs.runCommand "iso-eval" { } "touch $out";
    };
}
