{
  flake.modules.nixos.base =
    { lib, pkgs, ... }:
    {
      programs.zsh = {
        enable = true;
        # ~/.zshrc runs compinit
        enableCompletion = lib.mkDefault false;
        # no new-user wizard before ~/.zshrc exists
        shellInit = ''
          zsh-newuser-install() { :; }
        '';
      };
      users.defaultUserShell = pkgs.zsh;
      environment.pathsToLink = [ "/share/zsh" ];
    };

  flake.modules.homeManager.dotfiles = {
    programs.zsh = {
      enable = true;
      enableCompletion = true;
      autosuggestion.enable = true;
      syntaxHighlighting.enable = true;
      history = {
        size = 50000;
        save = 50000;
        share = true;
        extended = true;
        ignoreAllDups = true;
        ignoreSpace = true;
      };
    };
  };
}
