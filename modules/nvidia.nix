{
  flake.modules.nixos.nvidia =
    { config, ... }:
    let
      open = config.ephemeral.settings.nvidia == "open";
      drivers = config.boot.kernelPackages.nvidiaPackages;
    in
    {
      ephemeral.unfreePackages = [
        "nvidia-x11"
        "nvidia-settings"
        "nvidia-kernel-modules"
      ];

      services.xserver.videoDrivers = [ "nvidia" ];
      hardware.graphics.enable = true;
      hardware.nvidia = {
        inherit open;
        package = if open then drivers.stable else drivers.legacy_580;
        modesetting.enable = true;
        nvidiaSettings = true;
      };
    };
}
