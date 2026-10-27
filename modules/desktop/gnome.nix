{
  flake.modules.nixos.gnome =
    { pkgs, ... }:
    {
      environment.persistence."/persist".directories = [ "/var/lib/gdm" ];

      services.xserver.enable = true;

      services.displayManager.gdm.enable = true;
      services.desktopManager.gnome.enable = true;
      services.gnome.gnome-initial-setup.enable = false;

      services.gnome.gnome-keyring.enable = true;
      security.pam.services.login.enableGnomeKeyring = true;

      environment.systemPackages = with pkgs; [
        gnome-tweaks
        nautilus
        gnome-text-editor
        loupe
        file-roller
      ];
    };
}
