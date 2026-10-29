# nix build path:.#checks.x86_64-linux.install -L
# runs scripts/install.sh in a VM with no network, then boots the disk it wrote
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

  hash = "$y$j9T$jaQ.MaSahZXl8SYNSWtSD0$Jn8oyD73Z4h51tFFi6xcKlEWHRESYftAr..1WaGThz4";
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
    {
      checks.install = pkgs.testers.runNixOSTest {
        name = "install";

        nodes.installer =
          { pkgs, ... }:
          {
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
              target.send_console("disko\n")
              target.wait_for_unit("multi-user.target")

          installer.start()
          installer.wait_for_unit("multi-user.target")
          installer.succeed("cp -r ${inputs.self} /root/installer && chmod -R u+w /root/installer")

          with subtest("install.sh partitions, encrypts and installs"):
              installer.succeed(
                  "cd /root/installer && env"
                  " DISK=${answers.diskDevice} CONFIRM=YES"
                  " INSTALL_HOSTNAME=${answers.hostname} INSTALL_USERNAME=${answers.username}"
                  " INSTALL_DESKTOP=${answers.desktop} NVIDIA_DRIVER=${answers.nvidia}"
                  " KEYBOARD_LAYOUT=${answers.keyboardLayout} KEYBOARD_VARIANT="
                  " LOCALE=${answers.locale} EXTRAS=false FINGERPRINT=false"
                  " SWAP_SIZE=${answers.swapSize} TIMEZONE=${answers.timeZone}"
                  " HARDWARE_CONFIG=/root/installer/tests/install/hardware-configuration.nix"
                  " PASSWORD_HASH=${lib.escapeShellArg hash}"
                  " IN_DISKO_TEST=1"
                  " ./scripts/install.sh >&2"
              )

          with subtest("settings.json has the answers, and the fresh-install copy is read-only"):
              got = json.loads(installer.succeed("cat /mnt/persist/etc/nixos/settings.json"))
              want = json.loads('${builtins.toJSON answers}')
              assert got == want, f"settings.json differs:\n{got}\n{want}"
              installer.succeed("cmp /mnt/persist/etc/nixos/flake.lock /mnt/persist/etc/nixos-install/flake.lock")
              assert installer.succeed("stat -c %a /mnt/persist/etc/nixos-install/flake.nix").strip() == "444"
              installer.succeed("test -s /mnt/persist/passwords/${answers.username}")

          with subtest("reinstall with the test backdoor, through the machine's own modules/"):
              installer.succeed(
                  "install -D -m 0644 /root/installer/tests/install/hosts.nix /mnt/persist/etc/nixos/modules/hosts.nix",
                  "install -m 0644 /root/installer/tests/install/_instrumentation.nix /mnt/persist/etc/nixos/modules/",
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
