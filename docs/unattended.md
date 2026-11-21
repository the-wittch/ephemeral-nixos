# Unattended installs

Every question can be answered ahead of time, so only the LUKS passphrase is typed in:

| variable | answer |
|----------|--------|
| `DISK` | the whole disk, e.g. `/dev/nvme0n1` or a `/dev/disk/by-id/` name |
| `INSTALL_HOSTNAME`, `INSTALL_USERNAME` | names |
| `INSTALL_DESKTOP` | `plasma`, `gnome`, `niri`, `umbriel`, or `none` for a server without a desktop |
| `NVIDIA_DRIVER` | `none`, `open` or `legacy` |
| `KEYBOARD_LAYOUT`, `KEYBOARD_VARIANT`, `LOCALE` | e.g. `de`, `nodeadkeys`, `de_DE.UTF-8` |
| `PROFILE` | `full` or `minimal` (the small install) |
| `EXTRAS`, `OPENSNITCH`, `PASS_SSH_AGENT`, `FINGERPRINT`, `SECURE_BOOT` | `true` or `false` |
| `SERVER` | `true` for a server: SSH with keys only, no sleep, and the disk unlocked over SSH after a reboot. On by itself with `INSTALL_DESKTOP=none` |
| `SSH_KEYS` | the public keys that may log into a server, one per line, e.g. `ssh-ed25519 AAAA… you@laptop` |
| `PASS_SSH_VAULT` | the Proton Pass vault with your SSH keys; no quotes, `$`, backticks or backslashes |
| `ENCRYPTED_DNS` | `none` (the network's DNS, the default), `cloudflare`, `quad9` or `adguard`: DNS over TLS to that provider. `custom` uses `CUSTOM_DNS` |
| `CUSTOM_DNS` | your own DNS over TLS servers for `custom`, as `address#name` separated by spaces, e.g. `194.242.2.2#dns.mullvad.net` |
| `CONTROLD_RESOLVER` | your Control D resolver id; empty for none. Can't be set together with `ENCRYPTED_DNS` |
| `APPS` | apps from the `ephemeral-apps` catalog, e.g. `obsidian,glow`, on top of the ones ticked by default; `-name` leaves out a default, e.g. `-vlc` |
| `SWAP_SIZE`, `TIMEZONE` | e.g. `16G`, `Europe/Berlin` |
| `PASSWORD_HASH` | the login password as a hash, from `mkpasswd -m yescrypt` |
| `HARDWARE_CONFIG` | a `hardware-configuration.nix` to use instead of detecting one |
| `CONFIRM` | `YES` skips the question that has you type the disk's name before it's erased |
| `ALLOW_SAME_PARTLABELS` | `1` installs while another disk has partitions from an earlier Ephemeral Nix install. They have the same labels the new system boots by, so it may start the old one; normally refused |
| `FORCE_DISK` | `1` allows a disk that's in use (mounted, swap, or an open LUKS, LVM or RAID device), holds an installer image, or is part of an LVM volume group, ZFS pool or RAID array; normally refused |

Pass them with `sudo env`, as in the README's install commands. Plain `sudo VAR=...` usually drops them.

## From the ISO

`ephemeral-install --answers answers.json` installs from a file, without the screens:

```json
{
  "diskDevice": "/dev/nvme0n1",
  "confirmDisk": "nvme0n1",
  "desktop": "gnome",
  "hostname": "my-laptop",
  "username": "alice",
  "timeZone": "Europe/Berlin",
  "passphrase": "the LUKS passphrase",
  "password": "the login password",
  "apps": ["obsidian"]
}
```

It takes these keys of [`settings.json`](settings.md): `diskDevice`, `desktop`, `nvidia`, `keyboardLayout`, `keyboardVariant`, `locale`, `timeZone`, `profile`, `extras`, `useOpenSnitch`, `usePassSshAgent`, `passSshVault`, `encryptedDns`, `customDns`, `fingerprint`, `secureBoot`, `server`, `sshKeys`, `hostname`, `username` and `swapSize`. Besides them it takes `passphrase`, `password`, `apps`, `controldResolver`, `hardwareConfig` (a path, like `HARDWARE_CONFIG`) and `confirmDisk`, the disk's name as the screens make you type it. `laptop` and `useControlD` may be there too, as in a `settings.json`, but the installer sets them itself, from the battery and from `controldResolver`. Any other key stops it before it starts, with the key named: a misspelled one would otherwise leave its setting at the default without a word. The other settings, such as `useTailscale`, go in `settings.json` after the install. Keys left out get the screens' defaults: Plasma, `us`, `en_US.UTF-8`, `UTC`, the full system with OpenSnitch, not a server, `nixos`, `nixuser` and 4G of swap. A server needs `sshKeys` with at least one key, as a list of whole public keys; `github:` names only work on the screens. The file holds both secrets in plain text, so `chmod 600` it and delete it afterwards; the installer warns when others can read it.

The screens and `--answers` ignore `FORCE_DISK`, `ALLOW_SAME_PARTLABELS` and `HARDWARE_CONFIG` from the environment they were started in; set them for `install.sh` or `bootstrap.sh` instead.
