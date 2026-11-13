# nix build path:.#checks.x86_64-linux.scripts -L
{
  perSystem =
    { pkgs, ... }:
    {
      checks.scripts =
        pkgs.runCommand "scripts"
          {
            nativeBuildInputs = with pkgs; [
              bash
              coreutils
              gawk
              gnugrep
            ];
            COMMON = ../../scripts/common.sh;
          }
          ''
            bash ${./_scripts.sh}
            touch $out
          '';
    };
}
