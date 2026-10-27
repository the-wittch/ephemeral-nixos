{
  flake.modules.nixos.plasma =
    { pkgs, ... }:
    {
      environment.persistence."/persist".directories = [ "/var/lib/plasmalogin" ];

      services.xserver.enable = true;

      services.desktopManager.plasma6.enable = true;
      services.displayManager.plasma-login-manager.enable = true;

      boot.plymouth = {
        theme = "breeze";
        themePackages = [ pkgs.kdePackages.breeze-plymouth ];
      };

      security.pam.services.login.kwallet.enable = true;
      security.pam.services.plasmalogin.kwallet.enable = true;

      environment.plasma6.excludePackages = [ pkgs.kdePackages.plasma-welcome ];

      environment.systemPackages = with pkgs.kdePackages; [
        kate
        ark
        okular
        gwenview
        spectacle
        discover
        partitionmanager
      ];
    };
}
