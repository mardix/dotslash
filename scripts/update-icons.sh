#!/bin/sh
# Regenerates icons.json — the list of icon names the icon picker searches —
# from the homarr-labs/dashboard-icons repository. Needs git, sort and comm.
#
#   ./scripts/update-icons.sh
#
# Output: {"svg":[...names with an SVG...],"png":[...names that only have a PNG...]}
set -eu

out="${1:-$(dirname "$0")/../icons.json}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

git clone --quiet --filter=blob:none --no-checkout --depth 1 \
  https://github.com/homarr-labs/dashboard-icons.git "$tmp/repo"

git -C "$tmp/repo" ls-tree -r --name-only HEAD > "$tmp/paths"

names() { # $1 = folder/extension (svg or png)
  grep -E "^$1/[a-z0-9._-]+\.$1\$" "$tmp/paths" | sed -e "s#^$1/##" -e "s#\.$1\$##" | LC_ALL=C sort -u
}
names svg > "$tmp/svg"
names png > "$tmp/png"
LC_ALL=C comm -13 "$tmp/svg" "$tmp/png" > "$tmp/pngonly"

join() { sed 's/.*/"&"/' "$1" | paste -sd, -; }
printf '{"svg":[%s],"png":[%s]}' "$(join "$tmp/svg")" "$(join "$tmp/pngonly")" > "$tmp/icons.json"

count_svg=$(wc -l < "$tmp/svg" | tr -d ' ')
count_png=$(wc -l < "$tmp/pngonly" | tr -d ' ')
if [ "$count_svg" -lt 500 ]; then
  echo "update-icons: only $count_svg icons found, keeping the existing file" >&2
  exit 1
fi
mv "$tmp/icons.json" "$out"
echo "update-icons: wrote $out ($count_svg svg, $count_png png-only)"
