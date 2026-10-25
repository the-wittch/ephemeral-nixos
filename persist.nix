{
  config,
  pkgs,
  lib,
  settings,
  ...
}:

let
  owner = lib.escapeShellArg "${settings.username}:${config.users.users.${settings.username}.group}";
in
{
  fileSystems."/persist".neededForBoot = true;
  fileSystems."/var/log".neededForBoot = true;

  # base system state. feature modules persist their own
  environment.persistence."/persist" = {
    hideMounts = true;
    directories = [
      "/home"
      "/var/lib/nixos"
      "/var/lib/AccountsService"
      {
        directory = "/var/lib/bluetooth";
        mode = "0700";
      }
      "/var/lib/NetworkManager"
      {
        directory = "/etc/NetworkManager/system-connections";
        mode = "0700";
      }
      "/var/lib/cups"
      {
        directory = "/var/lib/fwupd";
        user = "fwupd-refresh";
        group = "fwupd-refresh";
      }
      "/var/lib/systemd/backlight"
      "/var/lib/systemd/rfkill"
      "/var/lib/systemd/timers"
      "/var/lib/systemd/linger"
      {
        directory = "/var/lib/systemd/timesync";
        user = "systemd-timesync";
        group = "systemd-timesync";
      }
    ];
    files = [
      "/etc/machine-id"
      "/var/lib/systemd/random-seed"
      # systemd-creds host key (libvirt)
      "/var/lib/systemd/credential.secret"
    ];
  };

  # impermanence binds files from the initrd, where systemd-creds can't run.
  # make the key here, and on first boot swap impermanence's link for a mount.
  systemd.services.credential-secret-setup = {
    description = "Create the systemd credential key on /persist";
    wantedBy = [ "sysinit.target" ];
    after = [ "local-fs.target" ];
    before = [ "sysinit.target" ];
    unitConfig.DefaultDependencies = false;
    path = [
      config.systemd.package
      pkgs.coreutils
      pkgs.util-linux
    ];
    serviceConfig = {
      Type = "oneshot";
      RemainAfterExit = true;
      UMask = "0077";
    };
    script = ''
      key=/persist/var/lib/systemd/credential.secret
      dest=/var/lib/systemd/credential.secret
      if [[ ! -s $key ]]; then
        install -d -m 0755 "$(dirname "$key")"
        SYSTEMD_CREDENTIAL_SECRET=$key systemd-creds setup
      fi
      # systemd won't follow a symlink to the key
      if [[ -L $dest ]]; then
        rm "$dest"
        touch "$dest"
        mount --bind "$key" "$dest"
      fi
    '';
  };

  # impermanence copies these modes onto the fresh root. 0700 locks users out of /etc
  system.activationScripts.persistParentModes = {
    text = ''
      for d in /persist /persist/etc /persist/var /persist/var/lib /persist/home; do
        if [ -d "$d" ]; then
          chmod 0755 "$d"
          chown root:root "$d"
        fi
      done
    '';
  };
  system.activationScripts.createPersistentStorageDirs.deps = [ "persistParentModes" ];

  # blank root drops /home/$USER. create it here.
  system.activationScripts.ensureUserHome = {
    deps = [
      "users"
      "groups"
    ];
    text = ''
      home=${lib.escapeShellArg "/persist/home/${settings.username}"}
      if [ ! -d "$home" ]; then
        mkdir -p "$home"
        chmod 0700 "$home"
      fi
      chown ${owner} "$home"
    '';
  };

  system.activationScripts.persistSecrets = {
    deps = [ "users" ];
    text = ''
      mkdir -p /persist/passwords
      chmod 700 /persist/passwords
      for f in /persist/passwords/*; do
        if [ -e "$f" ]; then
          chmod 600 "$f"
        fi
      done

      mkdir -p /persist/etc/ssh
      chmod 755 /persist/etc/ssh
      for f in /persist/etc/ssh/ssh_host_*_key; do
        if [ -f "$f" ]; then
          chmod 600 "$f"
        fi
      done
      for f in /persist/etc/ssh/ssh_host_*_key.pub; do
        if [ -f "$f" ]; then
          chmod 644 "$f"
        fi
      done
    '';
  };
}
