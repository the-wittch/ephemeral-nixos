{
  flake.modules.nixos.pc = {
    environment.persistence."/persist".directories = [ "/var/lib/containers" ];

    # rootless. the podman group is root-equivalent
    virtualisation.podman = {
      enable = true;
      dockerCompat = true;
      defaultNetwork.settings.dns_enabled = true;
      autoPrune = {
        enable = true;
        dates = "weekly";
        flags = [ "--all" ];
      };
    };
  };
}
