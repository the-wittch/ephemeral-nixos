{
  flake.modules.nixos.base =
    { config, lib, ... }:
    let
      settings = config.ephemeral.settings;
    in
    {
      networking.hostName = settings.hostname;
      time.timeZone = settings.timeZone;
      i18n.defaultLocale = settings.locale;

      services.xserver.xkb = {
        layout = settings.keyboardLayout;
        variant = settings.keyboardVariant;
      };
      console.useXkbConfig = true;

      nixpkgs.hostPlatform = lib.mkDefault "x86_64-linux";

      hardware.enableRedistributableFirmware = true;
      services.fwupd.enable = true;

      system.stateVersion = "25.05";
    };
}
