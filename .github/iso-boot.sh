#!/usr/bin/env bash
# .github/iso-boot.sh <iso>
# The default boot entry has no serial console, so this reads the screen with OCR.
set -euo pipefail

iso=${1:?usage: iso-boot.sh <iso>}
[[ -f $iso ]] || { echo "no file $iso" >&2; exit 2; }
[[ -w /dev/kvm ]] || { echo "needs /dev/kvm" >&2; exit 2; }

work=$(mktemp -d)
ovmf=$(nix build --inputs-from path:. nixpkgs#OVMF.fd --no-link --print-out-paths)/FV
cp "$ovmf/OVMF_VARS.fd" "$work/vars.fd"
chmod u+w "$work/vars.fd"

qemu-system-x86_64 -enable-kvm -m 4096 -smp 2 \
  -drive if=pflash,format=raw,readonly=on,file="$ovmf/OVMF_CODE.fd" \
  -drive if=pflash,format=raw,file="$work/vars.fd" \
  -cdrom "$iso" -boot d \
  -vga std -display none \
  -monitor unix:"$work/monitor",server,nowait \
  -daemonize -pidfile "$work/pid"
trap 'kill "$(cat "$work/pid")" 2>/dev/null || true; rm -rf "$work"' EXIT

for _ in $(seq 1 60); do
  sleep 5
  echo "screendump $work/screen.ppm" | socat - UNIX-CONNECT:"$work/monitor" >/dev/null
  [[ -s $work/screen.ppm ]] || continue
  # tesseract does better with dark text on light
  magick "$work/screen.ppm" -negate -resize 200% "$work/screen.png"
  text=$(tesseract "$work/screen.png" - 2>/dev/null || true)
  if grep -q "Open a shell" <<<"$text"; then
    echo "the ISO booted to the installer's welcome screen"
    exit 0
  fi
done

echo "the installer's welcome screen didn't come up within 5 minutes. The screen last read:" >&2
printf '%s\n' "${text:-(nothing)}" | sed 's/^/  /' >&2
exit 1
