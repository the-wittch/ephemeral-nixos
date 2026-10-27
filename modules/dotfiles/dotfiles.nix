{
  # no home manager on first boot, so the system zshrc does completion
  flake.modules.nixos.dotfiles = {
    programs.zsh.enableCompletion = true;
  };
}
