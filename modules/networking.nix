{
  flake.modules.nixos.base =
    { pkgs, ... }:
    {
      networking.networkmanager = {
        enable = true;
        plugins = with pkgs; [
          networkmanager-openvpn
        ];
      };
      networking.firewall.enable = true;
    };
}
