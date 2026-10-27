{
  flake.modules.nixos.pc =
    { pkgs, ... }:
    {
      environment.systemPackages = with pkgs; [
        vim
        git
        curl
        wget
        firefox
        mkpasswd
        ghostty
        distrobox
        vlc
        wl-clipboard
        htop
        file
        unzip
        zip
        p7zip
        usbutils
        pciutils
        dig
        ripgrep
        wireguard-tools
        xdg-user-dirs
      ];

      environment.sessionVariables.TERMINAL = "ghostty";

      systemd.user.services.xdg-user-dirs = {
        description = "Create default XDG user directories";
        wantedBy = [ "default.target" ];
        unitConfig.ConditionUser = "!@system";
        serviceConfig = {
          Type = "oneshot";
          RemainAfterExit = true;
          ExecStart = "${pkgs.xdg-user-dirs}/bin/xdg-user-dirs-update";
        };
      };
    };
}
