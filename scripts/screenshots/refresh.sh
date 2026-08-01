#!/usr/bin/env bash
# Regenerate assets/screenshots/*.svg from real runs of the tool.
#
# Requirements: python3 with `pyte`, and `bwrap` (bubblewrap). The captures are
# real: the tool downloads real release archives from GitHub and really installs
# fonts — bwrap just shadows $HOME with a throwaway directory under .work/ so
# nothing touches the machine's actual font directory.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo="$(cd "$here/../.." && pwd)"
work="${NFI_SHOT_WORK:-$here/.work}"
out="$repo/assets/screenshots"

export NFI_SHOT_WORK="$work"

for tool in python3 bwrap go; do
	command -v "$tool" >/dev/null || { echo "missing required tool: $tool" >&2; exit 1; }
done
python3 -c 'import pyte' 2>/dev/null || { echo "missing python package: pyte (pip install pyte)" >&2; exit 1; }

rm -rf "$work"
mkdir -p "$work/home-bare" "$work/home-configured/.config/nerd-fonts-installer" "$out"

cat >"$work/home-configured/.config/nerd-fonts-installer/config.yaml" <<'YAML'
release: latest
destination: ~/.local/share/fonts/NerdFonts
refresh_font_cache: true
families:
  - JetBrainsMono
  - Hack
  - FiraCode
YAML

go build -trimpath -o "$work/nerd-fonts-installer" "$repo/cmd/nerd-fonts-installer"

# The session shot runs a real shell, so the binary has to be on PATH inside the
# sandbox for the typed commands to read the way a reader would type them.
mkdir -p "$work/home-configured/.local/bin"
install -m 0755 "$work/nerd-fonts-installer" "$work/home-configured/.local/bin/"

python3 "$here/specs.py"

for spec in "$work"/spec-*.json; do
	name="$(basename "$spec" .json)"
	name="${name#spec-}"
	title="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["title"])' "$spec")"
	python3 "$here/capture.py" "$spec" "$work/$name.capture.json"
	python3 "$here/render.py" "$work/$name.capture.json" "$out/$name.svg" "$title"
done

echo "screenshots written to $out"
