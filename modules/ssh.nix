{
  flake.modules.nixos.base =
    { lib, ... }:
    {
      services.openssh = {
        enable = lib.mkDefault false;
        openFirewall = false;
        settings = {
          PasswordAuthentication = false;
          KbdInteractiveAuthentication = false;
          PermitRootLogin = "no";
          X11Forwarding = false;
          AllowAgentForwarding = false;
        };
        hostKeys = [
          {
            path = "/persist/etc/ssh/ssh_host_ed25519_key";
            type = "ed25519";
          }
        ];
      };
    };
}
