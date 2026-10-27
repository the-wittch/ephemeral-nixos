{ lib, flakeRoot }:

let
  readData = import ./_read-data.nix { inherit lib; };

  defaults = {
    hostname = "nixos";
    username = "nixuser";
    diskDevice = "/dev/nvme0n1";
    swapSize = "16G";
    timeZone = "UTC";
    desktop = "plasma";
    useControlD = false;
    useChezmoi = false;
    useFlakePassCli = false;
    usePassSettings = false;
    passSettingsVault = "Personal";
    passSettingsItem = "nixos-settings";
  };

  # a missing file must not become username nixuser
  fromSettings = readData (flakeRoot + "/settings") (throw ''
    settings.json is not in this flake.
    Installed machine:
      sudo nixos-rebuild switch --flake path:/persist/etc/nixos#nixos
    Or copy /persist/etc/nixos/settings.json next to flake.nix.
  '');

  # the pass file can't turn itself on
  usePass = (defaults // fromSettings).usePassSettings or false;

  fromPass = if usePass then readData (flakeRoot + "/settings.from-pass") { } else { };

  merged =
    if !(fromSettings ? username) || fromSettings.username == "" then
      throw "settings must set username"
    else
      defaults // fromSettings // fromPass;
  merged' =
    if (merged.controldResolver or "") != "" || (merged.chezmoiRepo or "") != "" then
      throw ''
        Don't put controldResolver or chezmoiRepo in settings.
        Control D id: /persist/etc/controld-resolver and useControlD = true.
        chezmoi: /persist/etc/chezmoi-repo as "<git-url> <commit>" and useChezmoi = true.
      ''
    else
      lib.warnIf (merged ? allowUnfree)
        "settings.allowUnfree is no longer used. Unfree packages are listed in ephemeral.unfreePackages."
        merged;
in
assert merged'.hostname != "";
assert merged'.username != "";
assert merged'.diskDevice != "";
assert merged'.swapSize != "";
assert merged'.timeZone != "";
assert builtins.elem merged'.desktop [
  "plasma"
  "gnome"
  "niri"
];
removeAttrs merged' [
  "controldResolver"
  "chezmoiRepo"
  "allowUnfree"
]
