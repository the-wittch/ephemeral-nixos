# Unattended installs

Every question can be answered ahead of time, so only the LUKS passphrase is typed in:

| variable | answer |
|----------|--------|
| `DISK` | the whole disk, e.g. `/dev/nvme0n1` |
| `INSTALL_HOSTNAME`, `INSTALL_USERNAME` | names |
| `INSTALL_DESKTOP` | `plasma`, `gnome` or `niri` |
| `NVIDIA_DRIVER` | `none`, `open` or `legacy` |
| `KEYBOARD_LAYOUT`, `KEYBOARD_VARIANT`, `LOCALE` | e.g. `de`, `nodeadkeys`, `de_DE.UTF-8` |
| `PROFILE` | `full` or `minimal` (the small install) |
| `EXTRAS`, `OPENSNITCH`, `PASS_SSH_AGENT`, `FINGERPRINT` | `true` or `false` |
| `PASS_SSH_VAULT` | the Proton Pass vault with your SSH keys |
| `CONTROLD_RESOLVER` | your Control D resolver id; empty for none |
| `APPS` | apps from the `ephemeral-apps` catalog, e.g. `obsidian,glow`; empty for none |
| `SWAP_SIZE`, `TIMEZONE` | e.g. `16G`, `Europe/Berlin` |
| `PASSWORD_HASH` | the login password as a hash, from `mkpasswd -m yescrypt` |
| `HARDWARE_CONFIG` | a `hardware-configuration.nix` to use instead of detecting one |
| `CONFIRM` | `YES` skips the "erase this disk" question |

Pass them with `sudo env`, as in the README's install commands. Plain `sudo VAR=...` usually drops them.
