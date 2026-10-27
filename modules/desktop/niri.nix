{
  flake.modules.nixos.niri =
    { pkgs, ... }:
    {
      programs.niri.enable = true;
      programs.noctalia = {
        enable = true;
        systemd.enable = true;
        recommendedServices.enable = true;
      };
      services.displayManager.noctalia-greeter.enable = true;

      services.gnome.gnome-keyring.enable = true;
      security.pam.services.login.enableGnomeKeyring = true;
      security.pam.services.greetd.enableGnomeKeyring = true;

      environment.systemPackages = with pkgs; [
        xwayland-satellite
        fuzzel
        brightnessctl
        playerctl
      ];
    };
}
