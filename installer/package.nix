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
  kbd,
  xkeyboard-config,
  glibcLocales,
  # the config's assets/wallpaper.svg, for the snowflake on the welcome screen
  wallpaper,
  version,
  # flake.nix, flake.lock and scripts/, which install.sh copies and runs
  source,
}:
let
  # copies, so it doesn't carry all of glibcLocales and xkeyboard-config
  lists = runCommand "ephemeral-nixos-lists" { } ''
    mkdir $out
    cp ${xkeyboard-config}/share/X11/xkb/rules/base.lst $out/base.lst
    grep '\.UTF-8/UTF-8' ${glibcLocales}/share/i18n/SUPPORTED > $out/locales
  '';

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
  inherit version;

  src = lib.fileset.toSource {
    root = ./.;
    fileset = lib.fileset.unions [
      ./go.mod
      ./go.sum
      (lib.fileset.fileFilter (f: f.hasExt "go") ./.)
    ];
  };
  vendorHash = "sha256-SrgBd+Kqtd16g9enhR+Dj53iOpqS8roofuzkX+zY/AE=";

  env.CGO_ENABLED = "0";
  ldflags = [ "-X main.version=v${version}" ];
  nativeBuildInputs = [ makeWrapper ];

  postInstall = ''
    mv $out/bin/installer $out/bin/ephemeral-install
    mkdir -p $out/share
    cp -r ${source} $out/share/ephemeral-nixos
    ln -s ${lists} $out/share/ephemeral-nixos-lists
    wrapProgram $out/bin/ephemeral-install \
      --set-default EPHEMERAL_LOGO ${logo} \
      --prefix PATH : ${
        lib.makeBinPath [
          util-linux
          mkpasswd
          pciutils
          networkmanager
          kbd
        ]
      }
  '';

  meta = {
    description = "The Ephemeral Nix installer's screens, in front of scripts/install.sh";
    mainProgram = "ephemeral-install";
  };
}
