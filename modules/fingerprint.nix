{
  flake.modules.nixos.fingerprint = {
    services.fprintd.enable = true;
    environment.persistence."/persist".directories = [
      {
        directory = "/var/lib/fprint";
        mode = "0700";
      }
    ];
  };
}
