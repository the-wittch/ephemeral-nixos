{
  flake.modules.nixos.pc = {
    hardware.graphics.enable = true;

    hardware.bluetooth = {
      enable = true;
      powerOnBoot = true;
    };

    services.power-profiles-daemon.enable = true;
  };
}
