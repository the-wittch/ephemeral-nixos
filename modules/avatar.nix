{
  flake.modules.nixos.pc =
    { config, pkgs, ... }:
    let
      user = config.ephemeral.settings.username;
      avatar = pkgs.runCommand "ephemeral-avatar" { nativeBuildInputs = [ pkgs.librsvg ]; } ''
        sed '1s|width="3840" height="2160" viewBox="0 0 3840 2160"|width="512" height="512" viewBox="1350 330 1140 1140"|' \
          ${../assets/wallpaper.svg} > avatar.svg
        rsvg-convert -w 512 -h 512 avatar.svg -o $out
      '';
    in
    {
      # default only: a picture set in the desktop's settings stays
      systemd.services.ephemeral-avatar = {
        description = "Default user picture";
        wantedBy = [ "accounts-daemon.service" ];
        before = [
          "accounts-daemon.service"
          "display-manager.service"
        ];
        unitConfig.RequiresMountsFor = "/var/lib/AccountsService";
        serviceConfig = {
          Type = "oneshot";
          UMask = "0077";
          ReadWritePaths = [ "/var/lib/AccountsService" ];
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
          icon=/var/lib/AccountsService/icons/${user}
          entry=/var/lib/AccountsService/users/${user}
          [[ -d /var/lib/AccountsService/icons ]] || install -d -m 755 /var/lib/AccountsService/icons
          [[ -e $icon ]] || install -m 644 ${avatar} "$icon"
          install -d -m 700 /var/lib/AccountsService/users
          if [[ ! -e $entry ]]; then
            printf '[User]\nIcon=%s\nSystemAccount=false\n' "$icon" > "$entry"
          elif ! grep -q '^Icon=' "$entry"; then
            printf 'Icon=%s\n' "$icon" >> "$entry"
          fi
        '';
      };
    };
}
