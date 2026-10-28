{
  flake.modules.nixos.laptop =
    { config, ... }:
    {
      services.thermald.enable = config.hardware.cpu.intel.updateMicrocode;
      services.upower.enable = true;
      hardware.sensor.iio.enable = true;
    };
}
