{
  flake.modules.nixos.pc =
    { pkgs, ... }:
    {
      fonts.packages = with pkgs; [
        noto-fonts-color-emoji
        noto-fonts-cjk-sans
        liberation_ttf
      ];
      programs.dconf.enable = true;
    };
}
