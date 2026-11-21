# What the installer still repeats from the config, held against the config release it pins.
# Everything else comes from that release directly: the settings rules and menus from its
# settings-contract package and settings output, the apps from app-catalog, the wallpaper and
# the welcome screen's logo from its packages.
{
  config,
  inputs,
  lib,
  ...
}:
let
  script = name: builtins.readFile ../../scripts/${name};
  # every capture of the regex in text, in order
  captures =
    regex: text:
    lib.concatMap (m: if builtins.isList m then [ (builtins.head m) ] else [ ]) (
      builtins.split regex text
    );
in
{
  perSystem =
    { pkgs, system, ... }:
    let
      contract = lib.importJSON inputs.ephemeral.packages.${system}.settings-contract;
      settings = [ "username" ] ++ lib.attrNames contract.defaults;
      installSh = script "install.sh";
      commonSh = script "common.sh";
      initrdKey = import "${inputs.ephemeral}/modules/_initrd-ssh-key.nix";
      disks = config.flake.nixosConfigurations.install-plain.config.disko.devices.disk;
      parts = lib.concatMap lib.attrValues (map (d: d.content.partitions) (lib.attrValues disks));

      problems =
        # validate_partlabels and teardown look for these before the layout is built
        lib.concatMap (
          p:
          lib.optional (!lib.hasInfix "\"${p.label}\"" commonSh) "common.sh doesn't know partition ${p.label}"
        ) parts
        ++ lib.concatMap (
          p:
          lib.optional (
            p.content.type == "luks" && !lib.hasInfix "cryptsetup status ${p.content.name}" commonSh
          ) "common.sh's teardown doesn't close ${p.content.name}"
        ) parts
        # the bootstrap scripts run before there's a clone with scripts/nix.conf in it
        ++
          lib.concatMap
            (
              name:
              lib.optional (
                !lib.hasInfix "NIX_CONFIG=\"\${NIX_CONFIG-}\n${script "nix.conf"}\"" (script name)
              ) "${name}'s Nix settings differ from scripts/nix.conf"
            )
            [
              "bootstrap.sh"
              "bootstrap-tui.sh"
            ]
        # a server's key for unlocking over SSH, which install.sh makes before the boot loader needs it
        ++ map (path: "install.sh doesn't make ${path}, where the config looks for it") (
          builtins.filter (path: !lib.hasInfix "/mnt${path}" installSh) [
            initrdKey.key
            initrdKey.boot
          ]
        )
        # what install.sh writes into settings.json, and what --answers reads as settings
        ++ map (key: "install.sh writes ${key}, which the config has no setting for") (
          lib.subtractLists settings (captures "\\[ \"([a-zA-Z]+)\" " installSh)
        )
        ++ map (key: "--answers reads ${key} as a setting, which the config doesn't have") (
          lib.subtractLists (
            settings
            ++ [
              # the installer's own, which never reach settings.json
              "passphrase"
              "password"
              "apps"
              "controldResolver"
              "hardwareConfig"
              "confirmDisk"
            ]
          ) (captures "json:\"([a-zA-Z]+)\"" (builtins.readFile ../../installer/install.go))
        );
    in
    {
      checks.contract =
        assert lib.assertMsg (problems == [ ]) ''
          the installer and config release ${inputs.ephemeral.shortRev or "?"} disagree:
          ${lib.concatStringsSep "\n" problems}'';
        pkgs.runCommand "contract" { } "touch $out";
    };
}
