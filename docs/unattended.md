# Unattended installs

Every question can be answered ahead of time, so only the LUKS passphrase is typed in:

| variable | answer |
|----------|--------|
| `DISK` | the whole disk, e.g. `/dev/nvme0n1` or a `/dev/disk/by-id/` name |
| `INSTALL_HOSTNAME`, `INSTALL_USERNAME` | names |
| `INSTALL_DESKTOP` | `plasma`, `gnome`, `niri` or `umbriel` |
| `NVIDIA_DRIVER` | `none`, `open` or `legacy` |
| `KEYBOARD_LAYOUT`, `KEYBOARD_VARIANT`, `LOCALE` | e.g. `de`, `nodeadkeys`, `de_DE.UTF-8` |
| `PROFILE` | `full` or `minimal` (the small install) |
| `EXTRAS`, `OPENSNITCH`, `PASS_SSH_AGENT`, `FINGERPRINT` | `true` or `false` |
| `PASS_SSH_VAULT` | the Proton Pass vault with your SSH keys; no quotes, `$`, backticks or backslashes |
| `ENCRYPTED_DNS` | `none` (the network's DNS, the default), `cloudflare`, `quad9` or `adguard`: DNS over TLS to that provider. `custom` uses `CUSTOM_DNS` |
| `CUSTOM_DNS` | your own DNS over TLS servers for `custom`, as `address#name` separated by spaces, e.g. `194.242.2.2#dns.mullvad.net` |
| `CONTROLD_RESOLVER` | your Control D resolver id; empty for none. Can't be set together with `ENCRYPTED_DNS` |
| `APPS` | apps from the `ephemeral-apps` catalog, e.g. `obsidian,glow`; empty for none |
| `SWAP_SIZE`, `TIMEZONE` | e.g. `16G`, `Europe/Berlin` |
| `PASSWORD_HASH` | the login password as a hash, from `mkpasswd -m yescrypt` |
| `HARDWARE_CONFIG` | a `hardware-configuration.nix` to use instead of detecting one |
| `CONFIRM` | `YES` skips the question that has you type the disk's name before it's erased |
| `ALLOW_SAME_PARTLABELS` | `1` installs while another disk has partitions from an earlier Ephemeral Nix install. They have the same labels the new system boots by, so it may start the old one; normally refused |
| `FORCE_DISK` | `1` allows a disk that's in use (mounted, swap, or an open LUKS, LVM or RAID device), holds an installer image, or is part of an LVM volume group, ZFS pool or RAID array; normally refused |

Pass them with `sudo env`, as in the README's install commands. Plain `sudo VAR=...` usually drops them.
