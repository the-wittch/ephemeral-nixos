{
  flake.modules.nixos.pc =
    {
      config,
      lib,
      pkgs,
      ...
    }:
    {
      options.ephemeral.flathubRepo = lib.mkOption {
        type = lib.types.package;
        readOnly = true;
        description = "The pinned flathub.flatpakrepo file.";
        # pinned: it holds flathub's signing key
        default = pkgs.fetchurl {
          url = "https://dl.flathub.org/repo/flathub.flatpakrepo";
          hash = "sha256-M3HdJQ5h2eFjNjAHP+/aFTzUQm9y9K+gwzc64uj+oDo=";
        };
      };

      config.services.flatpak.enable = true;
      config.environment.persistence."/persist".directories = [ "/var/lib/flatpak" ];

      config.systemd.services.flatpak-flathub = {
        description = "Add Flathub Flatpak remote";
        wantedBy = [ "multi-user.target" ];
        path = [ pkgs.flatpak ];
        serviceConfig = {
          Type = "oneshot";
          RemainAfterExit = true;
          StateDirectory = "flatpak";
          ProtectSystem = "strict";
          ProtectHome = true;
          PrivateTmp = true;
          PrivateDevices = true;
          PrivateNetwork = true;
          NoNewPrivileges = true;
          CapabilityBoundingSet = [ "" ];
          ProtectKernelTunables = true;
          ProtectKernelModules = true;
          ProtectKernelLogs = true;
          ProtectControlGroups = true;
          ProtectClock = true;
          ProtectHostname = true;
          RestrictAddressFamilies = [ "AF_UNIX" ];
          RestrictNamespaces = true;
          RestrictSUIDSGID = true;
          LockPersonality = true;
          MemoryDenyWriteExecute = true;
          SystemCallArchitectures = "native";
        };
        script = ''
          flatpak remote-add --system --if-not-exists flathub ${config.ephemeral.flathubRepo}
        '';
      };
    };
}
