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
              jq
            ];
            COMMON = ../../scripts/common.sh;
          }
          ''
            bash ${./_scripts.sh}
            touch $out
          '';
    };
}
