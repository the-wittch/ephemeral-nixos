{
  description = "An Ephemeral Nix machine";

  inputs = {
    ephemeral.url = "github:the-wittch/ephemeral-nixos-config/v3.1.0";
    # the installer runs nixpkgs tools by name
    nixpkgs.follows = "ephemeral/nixpkgs";
  };

  outputs =
    inputs:
    inputs.ephemeral.lib.mkFlake {
      inherit inputs;
      root = ./.;
    };
}
