{
  flake.modules.nixos.base =
    { config, pkgs, ... }:
    let
      settings = config.ephemeral.settings;
    in
    {
      users.mutableUsers = false;
      users.users.${settings.username} = {
        isNormalUser = true;
        shell = pkgs.zsh;
        extraGroups = [
          "wheel"
          "networkmanager"
          "video"
          "audio"
          "lp"
          "dialout"
        ];
        hashedPasswordFile = "/persist/passwords/${settings.username}";
      };
      users.users.root.hashedPassword = "!";

      security.sudo = {
        execWheelOnly = true;
        extraConfig = ''
          Defaults lecture = never
        '';
      };
    };
}
