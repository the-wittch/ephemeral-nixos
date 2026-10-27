{
  flake.modules.nixos.base =
    { config, lib, ... }:
    {
      networking.hostName = config.ephemeral.settings.hostname;
      time.timeZone = config.ephemeral.settings.timeZone;
      i18n.defaultLocale = "en_US.UTF-8";

      nixpkgs.hostPlatform = lib.mkDefault "x86_64-linux";

      hardware.enableRedistributableFirmware = true;
      services.fwupd.enable = true;

      system.stateVersion = "25.05";
    };
}
