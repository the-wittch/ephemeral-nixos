# nix run path:.#installer -- --dry-run
{ inputs, lib, ... }:
let
  # the newest release in CHANGELOG.md; release.sh won't tag one without it
  version = lib.pipe ../CHANGELOG.md [
    builtins.readFile
    (lib.splitString "\n")
    (map (builtins.match "## ([0-9]+\\.[0-9]+\\.[0-9]+).*"))
    (lib.findFirst (m: m != null) (throw "CHANGELOG.md has no ## X.Y.Z heading"))
    lib.head
  ];
in
{
  perSystem =
    { pkgs, ... }:
    let
      installer = pkgs.callPackage ../installer/package.nix {
        inherit version;
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
