{
  description = "An Ephemeral Nix machine";

  inputs = {
    ephemeral.url = "github:the-wittch/ephemeral-nixos-config/v3.0.0";
    # the installer runs these by name
    nixpkgs.follows = "ephemeral/nixpkgs";
    disko.follows = "ephemeral/disko";
  };

  outputs =
    inputs:
    inputs.ephemeral.lib.mkFlake {
      inherit inputs;
      root = ./.;
    };
}
