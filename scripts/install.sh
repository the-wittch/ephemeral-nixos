#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"

[[ $EUID -eq 0 ]] || die "run as root"
require_uefi

FLAKE_ROOT=$(cd "$(dirname "$SCRIPT_DIR")" && pwd)
cd "$FLAKE_ROOT"

command -v nix >/dev/null || die "nix not found"
command -v lsblk >/dev/null || die "lsblk not found"

# held until this script and everything it started have exited, so a second install can't
# format the same disk or tear down this one's mounts
exec 9>/run/ephemeral-install.lock
flock -n 9 || die "another install is running; ephemeral-install shows its progress"

# in /run, so it only covers this boot; the screens' own folder is emptied at every start
STATE=/run/ephemeral-install.state

if [[ ${1:-} == --teardown ]]; then
  teardown
  exit 0
fi

LOG=/tmp/ephemeral-install.log
# the attempt before is often the one worth reporting
if [[ -s $LOG ]]; then
  mv -f "$LOG" "$LOG.1"
fi
(umask 077 && : >"$LOG")
exec > >(tee -a "$LOG") 2>&1
info "a copy of this output goes to $LOG"

STAGE_NAME=prepare
set -E
trap on_error ERR

export NIX_CONFIG="${NIX_CONFIG-}
experimental-features = nix-command flakes
http2 = false
download-attempts = 5
connect-timeout = 10
"

teardown

INSTALL_HOSTNAME=${INSTALL_HOSTNAME:-nixos}
INSTALL_USERNAME=${INSTALL_USERNAME:-nixuser}
SWAP_SIZE=${SWAP_SIZE:-4G}
TIMEZONE=${TIMEZONE:-UTC}
FLAKE_ATTR=${FLAKE_ATTR:-nixos}
PASSWORD_HASH=${PASSWORD_HASH:-}
HARDWARE_CONFIG=${HARDWARE_CONFIG:-}

if [[ -z ${DISK:-} ]]; then
  info "disks:"
  lsblk -dpn -o NAME,TYPE,SIZE,TRAN,MODEL | awk '$2=="disk"{print}' | sed 's/^/    /'
  echo
  read -r -p "whole disk (e.g. /dev/vda): " DISK
fi

validate_disk "$DISK"
# the screens pass what they saw when the name was typed in; otherwise it's what's there now
DISK_ID=${DISK_ID:-$(disk_identity "$DISK")}
check_disk_identity() {
  [[ $(disk_identity "$DISK") == "$DISK_ID" ]] ||
    die "$DISK isn't the disk that was chosen any more; a disk was plugged in or out. Nothing was changed; choose the disk again"
}
check_disk_identity

if [[ -z ${INSTALL_DESKTOP:-} ]]; then
  echo
  info "desktop environment:"
  echo "  1) plasma  — KDE Plasma 6"
  echo "  2) gnome   — GNOME"
  echo "  3) niri    — niri + Noctalia shell"
  echo "  4) umbriel — Umbriel + Noctalia shell (new, may change between releases)"
  echo
  read -r -p "Choice [1]: " desk_choice
  case ${desk_choice:-1} in
    1 | plasma | Plasma | kde | KDE) INSTALL_DESKTOP=plasma ;;
    2 | gnome | Gnome | GNOME) INSTALL_DESKTOP=gnome ;;
    3 | niri | Niri) INSTALL_DESKTOP=niri ;;
    4 | umbriel | Umbriel) INSTALL_DESKTOP=umbriel ;;
    *) die "invalid desktop choice: ${desk_choice}" ;;
  esac
fi

validate_desktop "$INSTALL_DESKTOP"

if [[ -z ${NVIDIA_DRIVER:-} ]]; then
  if gpu=$(nvidia_gpu); then
    echo
    info "found: $gpu"
    echo "  1) open    — RTX 20 / GTX 16 series and newer"
    echo "  2) legacy  — GTX 900 / GTX 10 series"
    echo "  3) none    — keep the open-source nouveau driver"
    echo
    read -r -p "Install NVIDIA drivers? [1]: " nv_choice
    case ${nv_choice:-1} in
      1 | open) NVIDIA_DRIVER=open ;;
      2 | legacy) NVIDIA_DRIVER=legacy ;;
      3 | none) NVIDIA_DRIVER=none ;;
      *) die "invalid choice: ${nv_choice}" ;;
    esac
  else
    NVIDIA_DRIVER=none
  fi
fi
validate_nvidia "$NVIDIA_DRIVER"

if [[ -z ${KEYBOARD_LAYOUT:-} ]]; then
  read -r -p "Keyboard layout [us]: " KEYBOARD_LAYOUT
  KEYBOARD_LAYOUT=${KEYBOARD_LAYOUT:-us}
  read -r -p "Keyboard variant (empty for the default): " KEYBOARD_VARIANT
fi
KEYBOARD_VARIANT=${KEYBOARD_VARIANT:-}
validate_keyboard "$KEYBOARD_LAYOUT" "$KEYBOARD_VARIANT"
warn_luks_layout "$KEYBOARD_LAYOUT"
if [[ -z ${LOCALE:-} ]]; then
  read -r -p "Language [en_US.UTF-8]: " LOCALE
  LOCALE=${LOCALE:-en_US.UTF-8}
fi
validate_locale "$LOCALE"

if [[ -z ${PROFILE:-} ]]; then
  echo
  info "what to install:"
  echo "  1) full   — everything below, with the options you pick"
  echo "  2) small  — a plain desktop now; the rest later with ephemeral-setup"
  echo
  read -r -p "Choice [1]: " profile_choice
  case ${profile_choice:-1} in
    1 | full) PROFILE=full ;;
    2 | small | minimal) PROFILE=minimal ;;
    *) die "invalid choice: ${profile_choice}" ;;
  esac
fi
validate_profile "$PROFILE"

if [[ $PROFILE == full ]]; then
  if [[ -z ${EXTRAS:-} ]]; then
    read -r -p "Install extra apps and tools (Steam, VS Code, Brave, VLC, atuin, comma and more)? [y/N]: " ex
    EXTRAS=false
    [[ ${ex:-n} =~ ^[Yy] ]] && EXTRAS=true
  fi
  if [[ -z ${OPENSNITCH:-} ]]; then
    read -r -p "OpenSnitch: ask before programs connect out? [Y/n]: " os
    OPENSNITCH=true
    [[ ${os:-y} =~ ^[Nn] ]] && OPENSNITCH=false
  fi
  if [[ -z ${PASS_SSH_AGENT:-} ]]; then
    read -r -p "Proton Pass SSH agent at login? [y/N]: " pa
    PASS_SSH_AGENT=false
    [[ ${pa:-n} =~ ^[Yy] ]] && PASS_SSH_AGENT=true
  fi
  if [[ $PASS_SSH_AGENT == true && -z ${PASS_SSH_VAULT:-} ]]; then
    read -r -p "Proton Pass vault with your SSH keys [Personal]: " PASS_SSH_VAULT
  fi
  if [[ -z ${ENCRYPTED_DNS:-} ]]; then
    read -r -p "Encrypted DNS: none (the network's), cloudflare, quad9, adguard or custom [none]: " ENCRYPTED_DNS
  fi
  if [[ ${ENCRYPTED_DNS:-} == custom && -z ${CUSTOM_DNS:-} ]]; then
    read -r -p "Servers as address#name, separated by spaces: " CUSTOM_DNS
  fi
  if [[ -z ${CONTROLD_RESOLVER+x} && ${ENCRYPTED_DNS:-none} == none ]]; then
    read -r -p "Control D resolver id (empty for none): " CONTROLD_RESOLVER
  fi
fi
EXTRAS=${EXTRAS:-false}
OPENSNITCH=${OPENSNITCH:-true}
PASS_SSH_AGENT=${PASS_SSH_AGENT:-false}
PASS_SSH_VAULT=${PASS_SSH_VAULT:-Personal}
CONTROLD_RESOLVER=${CONTROLD_RESOLVER:-}
ENCRYPTED_DNS=${ENCRYPTED_DNS:-none}
CUSTOM_DNS=${CUSTOM_DNS:-}
APPS=${APPS:-}
APPS=${APPS//,/ }
validate_apps "$APPS"
validate_bool EXTRAS "$EXTRAS"
validate_bool OPENSNITCH "$OPENSNITCH"
validate_bool PASS_SSH_AGENT "$PASS_SSH_AGENT"
validate_vault "$PASS_SSH_VAULT"
validate_controld "$CONTROLD_RESOLVER"
validate_dns "$ENCRYPTED_DNS"
if [[ $ENCRYPTED_DNS == custom ]]; then
  validate_custom_dns "$CUSTOM_DNS"
else
  CUSTOM_DNS=
fi
if [[ $PROFILE == minimal ]]; then
  EXTRAS=false
  PASS_SSH_AGENT=false
  CONTROLD_RESOLVER=
  APPS=
fi
if [[ -n $CONTROLD_RESOLVER && $ENCRYPTED_DNS != none ]]; then
  die "pick one of Control D and ENCRYPTED_DNS=$ENCRYPTED_DNS; DNS can only go to one of them"
fi

LAPTOP=false
has_battery && LAPTOP=true
if [[ -z ${FINGERPRINT:-} ]]; then
  FINGERPRINT=false
  if reader=$(fingerprint_reader); then
    read -r -p "Fingerprint reader found: $reader. Use it for login and sudo? [Y/n]: " fp
    [[ ${fp:-y} =~ ^[Yy] ]] && FINGERPRINT=true
  fi
fi
[[ $FINGERPRINT == true || $FINGERPRINT == false ]] || die "FINGERPRINT must be true or false"
validate_hostname "$INSTALL_HOSTNAME"
validate_username "$INSTALL_USERNAME"
validate_swap "$SWAP_SIZE"
validate_timezone "$TIMEZONE"
validate_flake_attr "$FLAKE_ATTR"
if [[ -n $PASSWORD_HASH ]]; then
  [[ $PASSWORD_HASH == '$'* && $PASSWORD_HASH != *[[:space:]:]* ]] || die "PASSWORD_HASH must be a crypt hash, e.g. from mkpasswd -m yescrypt"
fi
if [[ -n $HARDWARE_CONFIG ]]; then
  [[ -f $HARDWARE_CONFIG ]] || die "HARDWARE_CONFIG is not a file: $HARDWARE_CONFIG"
  HARDWARE_CONFIG=$(realpath "$HARDWARE_CONFIG")
fi
validate_swap_fits "$DISK" "$SWAP_SIZE"
validate_partlabels "$DISK"
info "desktop=$INSTALL_DESKTOP profile=$PROFILE nvidia=$NVIDIA_DRIVER keyboard=$KEYBOARD_LAYOUT${KEYBOARD_VARIANT:+($KEYBOARD_VARIANT)} locale=$LOCALE extras=$EXTRAS laptop=$LAPTOP fingerprint=$FINGERPRINT${APPS:+ apps=$APPS}"
info "opensnitch=$OPENSNITCH pass-ssh-agent=$PASS_SSH_AGENT${CONTROLD_RESOLVER:+ controld=on} dns=$ENCRYPTED_DNS"

STAGE=""
cleanup() {
  # leave STAGE first; a deleted cwd breaks nix
  if [[ -n ${STAGE:-} && -d ${STAGE:-} ]]; then
    cd / || true
    rm -rf "$STAGE"
  fi
}
trap cleanup EXIT

STAGE=$(mktemp -d /tmp/nixos-flake-stage.XXXXXX)
info "staging flake at $STAGE"

# the ISO runs from the read-only store; the copies have to be editable
cp --no-preserve=mode "$FLAKE_ROOT/flake.nix" "$FLAKE_ROOT/flake.lock" "$FLAKE_ROOT/.gitignore" "$STAGE/"
cp --no-preserve=mode "$FLAKE_ROOT/hardware-configuration.example.nix" "$STAGE/hardware-configuration.nix"

cd "$STAGE"

# values go in through the environment, so nothing needs escaping
# shellcheck disable=SC2016 # the ${...} below is nix interpolation
S_HOSTNAME=$INSTALL_HOSTNAME S_USERNAME=$INSTALL_USERNAME S_DISK=$DISK \
  S_SWAP=$SWAP_SIZE S_TZ=$TIMEZONE S_DESKTOP=$INSTALL_DESKTOP S_NVIDIA=$NVIDIA_DRIVER \
  S_KB=$KEYBOARD_LAYOUT S_KBV=$KEYBOARD_VARIANT S_LOCALE=$LOCALE S_LAPTOP=$LAPTOP S_FP=$FINGERPRINT S_EXTRAS=$EXTRAS \
  S_PROFILE=$PROFILE S_OPENSNITCH=$OPENSNITCH S_AGENT=$PASS_SSH_AGENT S_VAULT=$PASS_SSH_VAULT S_CONTROLD=$CONTROLD_RESOLVER S_DNS=$ENCRYPTED_DNS S_CUSTOMDNS=$CUSTOM_DNS \
  nix eval --impure --raw --expr '
    let
      e = builtins.getEnv;
      fields = [
        [ "hostname" (e "S_HOSTNAME") ]
        [ "username" (e "S_USERNAME") ]
        [ "diskDevice" (e "S_DISK") ]
        [ "swapSize" (e "S_SWAP") ]
        [ "timeZone" (e "S_TZ") ]
        [ "desktop" (e "S_DESKTOP") ]
        [ "profile" (e "S_PROFILE") ]
        [ "nvidia" (e "S_NVIDIA") ]
        [ "keyboardLayout" (e "S_KB") ]
        [ "keyboardVariant" (e "S_KBV") ]
        [ "locale" (e "S_LOCALE") ]
        [ "extras" (e "S_EXTRAS" == "true") ]
        [ "laptop" (e "S_LAPTOP" == "true") ]
        [ "fingerprint" (e "S_FP" == "true") ]
        [ "useOpenSnitch" (e "S_OPENSNITCH" == "true") ]
        [ "useControlD" (e "S_CONTROLD" != "") ]
        [ "usePassSshAgent" (e "S_AGENT" == "true") ]
      ]
      ++ (if e "S_AGENT" == "true" then [ [ "passSshVault" (e "S_VAULT") ] ] else [ ])
      # only written when set, so the installer also works with a config release from before it
      ++ (if e "S_DNS" != "none" then [ [ "encryptedDns" (e "S_DNS") ] ] else [ ])
      ++ (
        if e "S_DNS" == "custom" then
          [ [ "customDns" (builtins.filter (s: builtins.isString s && s != "") (builtins.split " +" (e "S_CUSTOMDNS"))) ] ]
        else
          [ ]
      );
      line = f: "  ${builtins.toJSON (builtins.elemAt f 0)}: ${builtins.toJSON (builtins.elemAt f 1)}";
    in
    "{\n" + builtins.concatStringsSep ",\n" (map line fields) + "\n}\n"
  ' >settings.json

if [[ -n ${APPS// /} ]]; then
  # shellcheck disable=SC2086 # one app per word
  printf '%s\n' $APPS | sort -u | sed 's/.*/  "&"/; $!s/$/,/' | { echo '['; cat; echo ']'; } >apps.json
fi

[[ -s flake.lock ]] || die "flake.lock is missing; the installer pins a config release"

# copied now: one on a stick at /mnt would be gone once disko has unmounted it
if [[ -n $HARDWARE_CONFIG ]]; then
  cp "$HARDWARE_CONFIG" "$STAGE/hardware-configuration.nix"
fi

# a mistake in the settings or the config would otherwise only show once the disk is erased
info "checking that the system builds from these settings"
info "the first time downloads nixpkgs and can sit at 0/N for several minutes"
nix eval --raw "path:${STAGE}#nixosConfigurations.${FLAKE_ATTR}.config.system.build.toplevel.drvPath" >/dev/null
DISKO=$(nix build -L --no-link --print-out-paths \
  "path:${STAGE}#nixosConfigurations.${FLAKE_ATTR}.config.system.build.destroyFormatMount")

info "$DISK ($(lsblk -dno SIZE -- "$DISK" | tr -d " ") $(lsblk -dno MODEL -- "$DISK" | sed "s/ *$//")) becomes a 1G boot partition and the rest encrypted, with a $SWAP_SIZE swap file inside."
info "Everything on it now is erased:"
lsblk "$DISK" | sed 's/^/    /'
if [[ ${CONFIRM:-} != YES ]]; then
  read -r -p "Type $(basename "$DISK") to erase it and install: " confirm
  [[ $confirm == "$(basename "$DISK")" ]] || die "aborted; nothing was changed"
fi
check_disk_identity

stage partition
info "partitioning (LUKS passphrase next)"
(
  umask 077
  printf '%s\n' "$DISK" >"$STATE"
)
"$DISKO/bin/disko-destroy-format-mount" --yes-wipe-all-disks

findmnt /mnt >/dev/null 2>&1 || die "/mnt is not mounted after disko"
findmnt /mnt/persist >/dev/null 2>&1 || die "/mnt/persist is not mounted after disko"
findmnt /mnt/boot >/dev/null 2>&1 || die "/mnt/boot is not mounted after disko"
btrfs subvolume list /mnt | grep -q 'root-blank' || die "root-blank missing after disko"

stage hardware
if [[ -n $HARDWARE_CONFIG ]]; then
  info "using $HARDWARE_CONFIG"
elif command -v nixos-generate-config >/dev/null; then
  info "detecting hardware"
  nixos-generate-config --no-filesystems --root /mnt --show-hardware-config >"$STAGE/hardware-configuration.nix" \
    || die "nixos-generate-config failed"
else
  info "warning: no nixos-generate-config; using the generic hardware-configuration.nix"
fi

mkdir -p /mnt/persist/passwords /mnt/persist/etc/ssh /mnt/persist/etc/nixos
chmod 700 /mnt/persist/passwords
chmod 755 /mnt/persist/etc/ssh
# has to exist before the bind mount; activation chowns it
install -d -m 0700 "/mnt/persist/home/${INSTALL_USERNAME}"
if [[ -n $CONTROLD_RESOLVER ]]; then
  (
    umask 077
    printf '%s\n' "$CONTROLD_RESOLVER" >/mnt/persist/etc/controld-resolver
  )
fi
# networks joined here with nmtui, so the new system is online at the first login
connections=(/etc/NetworkManager/system-connections/*.nmconnection)
if [[ -f ${connections[0]} ]]; then
  install -d -m 0700 /mnt/persist/etc/NetworkManager/system-connections
  install -m 0600 -o root -g root "${connections[@]}" /mnt/persist/etc/NetworkManager/system-connections/
  info "copied ${#connections[@]} network connection(s) from this live system"
fi

if [[ -n $PASSWORD_HASH ]]; then
  HASH=$PASSWORD_HASH
else
  info "login password for ${INSTALL_USERNAME}"
  if command -v mkpasswd >/dev/null; then
    HASH=$(mkpasswd -m yescrypt)
  else
    HASH=$(nix shell --inputs-from "path:${STAGE}" nixpkgs#mkpasswd -c mkpasswd -m yescrypt)
  fi
fi
[[ -n $HASH ]] || die "empty password hash (mkpasswd failed or cancelled)"

(
  umask 077
  printf '%s\n' "$HASH" > "/mnt/persist/passwords/${INSTALL_USERNAME}"
)
chmod 600 "/mnt/persist/passwords/${INSTALL_USERNAME}"
sync

tar -C "$STAGE" -cf - . | tar -C /mnt/persist/etc/nixos -xf -
# tar gives it the stage's 0700, which keeps the user from reading their own flake
chmod 0755 /mnt/persist/etc/nixos
[[ -f /mnt/persist/etc/nixos/settings.json ]] || die "settings.json missing after copy"
[[ -f /mnt/persist/etc/nixos/flake.nix ]] || die "flake.nix missing after copy"

# the system as installed, to rebuild back to after /persist/etc/nixos has moved on
mkdir -p /mnt/persist/etc/nixos-install
tar -C "$STAGE" -cf - . | tar -C /mnt/persist/etc/nixos-install -xf -
chown -R 0:0 /mnt/persist/etc/nixos-install
chmod -R a-w /mnt/persist/etc/nixos-install

cd /
cleanup
trap - EXIT
STAGE=""

install_system() {
  local system
  system=$(nix eval --raw "path:$1#nixosConfigurations.${FLAKE_ATTR}.config.system.build.toplevel.outPath") || return 1
  # already built here (an offline install): nixos-install --flake would rebuild it from source
  if nix path-info "$system" >/dev/null 2>&1; then
    nix copy --no-check-sigs --to /mnt "$system" || return 1
    nixos-install --no-root-passwd --no-channel-copy --system "$system"
  else
    nixos-install --no-root-passwd --flake "path:$1#${FLAKE_ATTR}"
  fi
}

stage install
info "installing from /mnt/persist/etc/nixos"
if ! install_system /mnt/persist/etc/nixos; then
  echo "error: the install failed" >&2
  echo "  retry (if /mnt is still mounted):" >&2
  echo "    cd / && sudo nixos-install --no-root-passwd --flake path:/mnt/persist/etc/nixos#${FLAKE_ATTR}" >&2
  echo "  store repair: sudo nix-store --verify --check-contents --repair" >&2
  echo "  the full output is in $LOG" >&2
  exit 1
fi

install -m 600 "$LOG" /mnt/var/log/ephemeral-install.log ||
  echo "warning: couldn't keep a copy of this output in the new system; it's still in $LOG" >&2

cat <<EOF

Installed. Remove the USB stick and reboot, unlock the disk, and log in as ${INSTALL_USERNAME}.
After the first login:
  ephemeral-doctor   check that everything is as it should be
  ephemeral-apps     pick more apps
  ephemeral-update   rebuild after a change, and get new releases
  ephemeral-help     every command

EOF
