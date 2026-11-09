# shellcheck shell=bash
# The wallpaper's snowflake and its pixels as truecolor characters, for the welcome screen.
# Args: wallpaper.svg out-file
set -euo pipefail
svg=$1 out=$2
cols=48 rows=14

# the snowflake alone, arms filled so they read as characters
sed -E \
  -e '/^<rect width="3840"/d' -e '/^<circle/d' -e '/^<path d="M /d' -e '/^<text/d' \
  -e '/^<rect x=/{/softglow/!d}' -e '/^<rect x=/s/ opacity="[0-9.]+"//' \
  -e 's/ filter="url\(#(glow|softglow)\)"//' \
  -e 's/fill-opacity="[0-9.]+"/fill-opacity="1"/' \
  "$svg" >flake.svg
rsvg-convert -w 3840 flake.svg -o flake.png

magick flake.png -crop 1660x1020+1320+390 +repage \
  -channel A -morphology Dilate Disk:3 +channel \
  -filter box -resize "${cols}x${rows}!" -depth 8 txt:- |
  awk '
    NR == 1 { next }
    {
      split($1, xy, /[,:]/); y = xy[2]
      gsub(/[()]/, "", $2); split($2, c, ",")
      a = c[4] / 255
      ch = a > 0.7 ? "@" : a > 0.45 ? "#" : a > 0.25 ? "+" : a > 0.08 ? "." : " "
      m = c[1]; if (c[2] > m) m = c[2]; if (c[3] > m) m = c[3]
      if (ch == " " || m == 0) cell = " "
      else cell = sprintf("\033[38;2;%d;%d;%dm%s", c[1] * 255 / m, c[2] * 255 / m, c[3] * 255 / m, ch)
      line[y] = line[y] cell
    }
    END { for (y = 0; y in line; y++) { s = line[y]; sub(/ +$/, "", s); print s "\033[0m" } }' >"$out"
