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
      contract = inputs.ephemeral.packages.${pkgs.stdenv.hostPlatform.system}.settings-contract;
      # the config's own verdict on each value in testdata/validation.json, matched the way the
      # config matches, for the Go tests to agree with
      verdicts =
        let
          inherit (lib.importJSON contract) rules;
          allows =
            key: value:
            if builtins.isList rules.${key} then
              builtins.elem value rules.${key}
            else
              builtins.match rules.${key} value != null;
        in
        pkgs.writeText "verdicts.json" (
          builtins.toJSON (
            lib.mapAttrs (
              key:
              map (value: {
                inherit value;
                ok = allows key value;
              })
            ) (lib.importJSON ../installer/testdata/validation.json)
          )
        );
      installer = pkgs.callPackage ../installer/package.nix {
        inherit version contract verdicts;
        logo = inputs.ephemeral.packages.${pkgs.stdenv.hostPlatform.system}.welcome-logo;
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
