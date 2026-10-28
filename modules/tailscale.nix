{
  flake.modules.nixos.tailscale =
    { config, ... }:
    let
      package = config.services.tailscale.package;
    in
    {
      services.tailscale.enable = true;
      environment.persistence."/persist".directories = [
        {
          directory = "/var/lib/tailscale";
          mode = "0700";
        }
      ];

      # opensnitch denies by default
      services.opensnitch.rules.tailscaled = {
        name = "tailscaled";
        enabled = true;
        action = "allow";
        duration = "always";
        operator = {
          type = "simple";
          sensitive = false;
          operand = "process.path";
          data = "${package}/bin/.tailscaled-wrapped";
        };
      };
    };
}
