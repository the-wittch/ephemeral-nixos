{
  flake.modules.nixos.pc = {
    services.printing.enable = true;
    services.avahi = {
      enable = false;
      nssmdns4 = true;
      openFirewall = true;
    };
  };
}
