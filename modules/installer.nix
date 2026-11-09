# nix run path:.#installer -- --dry-run
{ inputs, lib, ... }:
{
  perSystem =
    { pkgs, ... }:
    let
      installer = pkgs.callPackage ../installer/package.nix {
        wallpaper = "${inputs.ephemeral}/assets/wallpaper.svg";
        source = lib.fileset.toSource {
          root = ../.;
          fileset = lib.fileset.unions [
            ../flake.nix
            ../flake.lock
            ../.gitignore
            ../hardware-configuration.example.nix
            ../scripts
          ];
        };
      };
    in
    {
      packages.installer = installer;
      # building it runs the Go tests
      checks.installer = installer;
    };
}
