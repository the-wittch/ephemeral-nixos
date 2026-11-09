# nix build path:.#checks.x86_64-linux.install -L
{
  config,
  inputs,
  lib,
  ...
}:
let
  fixture = ../../tests/install;
  answers = lib.importJSON (fixture + "/settings.json");

  # what install.sh builds, and the same machine with the test driver's backdoor
  plain = config.flake.nixosConfigurations.install-plain.config;
  booted = config.flake.nixosConfigurations.install-booted.config;

  sources =
    let
      collect = input: [ input.outPath ] ++ lib.concatMap collect (lib.attrValues (input.inputs or { }));
    in
    lib.unique (lib.concatMap collect (lib.attrValues inputs));

  # typed into the boot prompt below, so the test fails if disko got anything else
  passphrase = "correct horse battery staple";
in
{
  ephemeral.hosts = {
    install-plain.root = fixture;
    install-booted = {
      root = fixture;
      modules = [ (fixture + "/_instrumentation.nix") ];
    };
  };

  perSystem =
    { pkgs, ... }:
    let
      # what the screens would collect, keyed like settings.json
      answersFile = pkgs.writeText "answers.json" (
        builtins.toJSON (
          answers
          // {
            apps = lib.importJSON (fixture + "/apps.json");
            inherit passphrase;
            password = "alice";
            hardwareConfig = "${fixture}/hardware-configuration.nix";
          }
        )
      );

      # a network joined with nmtui on the live system
      wifi = pkgs.writeText "home.nmconnection" ''
        [connection]
        id=home
        type=wifi

        [wifi]
        ssid=home
      '';
    in
    {
      checks.install = pkgs.testers.runNixOSTest {
        name = "install";
        # the installer profile sets nixpkgs.overlays
        node.pkgsReadOnly = false;

        # the ISO without its image: the same profiles, scripts and live system
        nodes.installer =
          { modulesPath, pkgs, ... }:
          {
            imports = [
              (modulesPath + "/profiles/base.nix")
              (modulesPath + "/profiles/installation-device.nix")
              config.flake.modules.nixos.iso
            ];
            virtualisation = {
              diskImage = "./target.qcow2";
              diskSize = 20 * 1024;
              emptyDiskImages = [ 2048 ];
              rootDevice = "/dev/vdb";
              fileSystems."/".autoFormat = true;
              useEFIBoot = true;
              memorySize = 6144;
              cores = 4;
            };
            boot.supportedFilesystems = [ "btrfs" ];
            environment.systemPackages = with pkgs; [
              nixos-install-tools
              btrfs-progs
              cryptsetup
            ];
            nix.settings = {
              experimental-features = [
                "nix-command"
                "flakes"
              ];
              substituters = lib.mkForce [ ];
              hashed-mirrors = null;
              connect-timeout = 1;
              flake-registry = "";
            };
            system.extraDependencies = [
              plain.system.build.toplevel
              plain.system.build.destroyFormatMount
              booted.system.build.toplevel
            ]
            ++ sources;
          };

        nodes.target = {
          virtualisation = {
            diskImage = "./target.qcow2";
            useBootLoader = true;
            useEFIBoot = true;
            useDefaultFilesystems = false;
            efi.keepVariables = false;
            fileSystems."/" = {
              device = "/dev/disk/by-label/this-is-not-real";
              fsType = "ext4";
            };
            memorySize = 4096;
            cores = 2;
          };
        };

        testScript = ''
          import json

          def unlock_and_boot():
              target.start()
              target.wait_for_console_text("passphrase")
              target.send_console("${passphrase}\n")
              target.wait_for_unit("multi-user.target")

          installer.start()
          installer.wait_for_unit("multi-user.target")

          with subtest("the live system starts the installer, which runs from the read-only store"):
              installer.wait_until_tty_matches("1", "Not connected")
              installer.succeed("test -x /etc/ephemeral-nixos/scripts/install.sh")
              installer.fail("touch /etc/ephemeral-nixos/scripts/install.sh")

          with subtest("the installer partitions, encrypts and installs, the passphrase going to disko on stdin"):
              installer.succeed("install -D -m 0600 ${wifi} /etc/NetworkManager/system-connections/home.nmconnection")
              installer.succeed("ephemeral-install --answers ${answersFile} >&2")

          with subtest("settings.json and apps.json have the answers, and the fresh-install copy is read-only"):
              got = json.loads(installer.succeed("cat /mnt/persist/etc/nixos/settings.json"))
              want = json.loads('${builtins.toJSON answers}')
              assert got == want, f"settings.json differs:\n{got}\n{want}"
              installer.succeed("cmp ${fixture}/apps.json /mnt/persist/etc/nixos/apps.json")
              installer.succeed("test -s /mnt/var/log/ephemeral-install.log")
              installer.succeed("cmp /mnt/persist/etc/nixos/flake.lock /mnt/persist/etc/nixos-install/flake.lock")
              assert installer.succeed("stat -c %a /mnt/persist/etc/nixos-install/flake.nix").strip() == "444"
              assert installer.succeed("stat -c %a /mnt/persist/etc/nixos/flake.nix").strip() == "644"
              installer.succeed("grep -q '^\\$y\\$' /mnt/persist/passwords/${answers.username}")
              installer.succeed("cmp ${wifi} /mnt/persist/etc/NetworkManager/system-connections/home.nmconnection")
              assert installer.succeed("stat -c '%a %U' /mnt/persist/etc/NetworkManager/system-connections/home.nmconnection").strip() == "600 root"

          with subtest("reinstall with the test backdoor, through the machine's own modules/"):
              installer.succeed(
                  "install -D -m 0644 ${fixture}/hosts.nix /mnt/persist/etc/nixos/modules/hosts.nix",
                  "install -m 0644 ${fixture}/_instrumentation.nix /mnt/persist/etc/nixos/modules/",
                  "system=$(nix eval --raw path:/mnt/persist/etc/nixos#nixosConfigurations.nixos.config.system.build.toplevel.outPath)"
                  " && nix copy --no-check-sigs --to /mnt \"$system\""
                  " && nixos-install --no-root-passwd --no-channel-copy --system \"$system\" >&2",
              )
              installer.succeed("swapoff -a && umount -R /mnt && cryptsetup close cryptroot && sync")
              installer.shutdown()

          target.state_dir = installer.state_dir

          with subtest("the installed disk unlocks and boots to the login screen"):
              unlock_and_boot()
              target.wait_for_unit("display-manager.service")
              failed = target.succeed("systemctl --failed --no-legend --plain").strip()
              assert failed == "", f"failed units:\n{failed}"
              target.succeed("btrfs subvolume list / | grep -q root-blank")
              target.succeed("test -s /persist/passwords/${answers.username}")
              target.succeed("command -v glow")
              # the fixture is only in the installer's store, so check what's in the file
              target.succeed("grep -qx ssid=home /etc/NetworkManager/system-connections/home.nmconnection")
              target.succeed("systemctl is-active --quiet root-reset-check.service")

          with subtest("root is wiped on reboot, /persist is kept"):
              target.succeed("touch /root-was-here /persist/persist-was-here")
              target.shutdown()
              unlock_and_boot()
              target.fail("test -e /root-was-here")
              target.succeed("test -e /persist/persist-was-here")
        '';
      };
    };
}
