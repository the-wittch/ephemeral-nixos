#!/usr/bin/env bash
# changelog-config.sh CHANGELOG.md 3.2.5 v3.3.2
# notes the new config in that version's entry, adding the entry if there isn't one yet
set -euo pipefail

file=$1
version=$2
config=$3
line="- New installs get config [$config](https://github.com/the-wittch/ephemeral-nixos-config/blob/main/CHANGELOG.md)."

grep -qF -- "$line" "$file" && exit 0

awk -v heading="## $version" -v line="$line" '
  function changed() { printf "### Changed\n\n%s\n\n", line; done = 1 }
  !done && !seen && /^## / && $0 != heading { print heading "\n"; changed(); print; next }
  $0 == heading { seen = 1; print; next }
  seen && !done && /^## / { changed(); print; next }
  seen && !done && $0 == "### Changed" { print; getline; print; print line; done = 1; next }
  { print }
  END { if (seen && !done) { print ""; changed() } }
' "$file" >"$file.new"
mv "$file.new" "$file"
