{
  lib,
  buildGoModule,
  makeWrapper,
  runCommand,
  librsvg,
  imagemagick,
  gawk,
  util-linux,
  mkpasswd,
  pciutils,
  networkmanager,
  # the config's assets/wallpaper.svg, for the snowflake on the welcome screen
  wallpaper,
  # flake.nix, flake.lock and scripts/, which install.sh copies and runs
  source,
}:
let
  logo =
    runCommand "ephemeral-install-logo"
      {
        nativeBuildInputs = [
          librsvg
          imagemagick
          gawk
        ];
      }
      ''
        bash ${./logo.sh} ${wallpaper} "$out"
      '';
in
buildGoModule {
  pname = "ephemeral-install";
  version = "0";

  src = lib.fileset.toSource {
    root = ./.;
    fileset = lib.fileset.unions [
      ./go.mod
      ./go.sum
      (lib.fileset.fileFilter (f: f.hasExt "go") ./.)
    ];
  };
  vendorHash = "sha256-4NZ5eEAnaXggxhspYHLRZuvYK6SlOAG62QA0lAgwpbE=";

  env.CGO_ENABLED = "0";
  nativeBuildInputs = [ makeWrapper ];

  postInstall = ''
    mv $out/bin/installer $out/bin/ephemeral-install
    mkdir -p $out/share
    cp -r ${source} $out/share/ephemeral-nixos
    wrapProgram $out/bin/ephemeral-install \
      --set-default EPHEMERAL_LOGO ${logo} \
      --prefix PATH : ${
        lib.makeBinPath [
          util-linux
          mkpasswd
          pciutils
          networkmanager
        ]
      }
  '';

  meta = {
    description = "The Ephemeral Nix installer's screens, in front of scripts/install.sh";
    mainProgram = "ephemeral-install";
  };
}
