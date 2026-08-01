#!/bin/sh
# nerd-fonts-installer installer
#
#   curl -fsSL https://worxbend.github.io/nerd-fonts-installer/install.sh | sh
#
# Downloads the release archive for this machine, verifies its SHA-256 against
# the release's checksums.txt, and installs the binary into a bin directory.
#
# Environment:
#   NFI_VERSION      release tag to install          (default: latest)
#   NFI_INSTALL_DIR  where to put the binary         (default: ~/.local/bin)
#
# Nothing is installed system-wide and nothing needs root unless you point
# NFI_INSTALL_DIR at a system path yourself.

set -eu

REPO="worxbend/nerd-fonts-installer"
BIN="nerd-fonts-installer"
VERSION="${NFI_VERSION:-latest}"
INSTALL_DIR="${NFI_INSTALL_DIR:-$HOME/.local/bin}"

# Colors only when stderr is a terminal, so piped output stays clean.
if [ -t 2 ]; then
	C_DIM=$(printf '\033[2m')
	C_RED=$(printf '\033[31m')
	C_GREEN=$(printf '\033[32m')
	C_CYAN=$(printf '\033[36m')
	C_BOLD=$(printf '\033[1m')
	C_OFF=$(printf '\033[0m')
else
	C_DIM='' C_RED='' C_GREEN='' C_CYAN='' C_BOLD='' C_OFF=''
fi

say() { printf '%s==>%s %s\n' "$C_CYAN" "$C_OFF" "$1" >&2; }
ok() { printf '%s ok%s  %s\n' "$C_GREEN" "$C_OFF" "$1" >&2; }
note() { printf '%s    %s%s\n' "$C_DIM" "$1" "$C_OFF" >&2; }
die() {
	printf '%serror%s %s\n' "$C_RED" "$C_OFF" "$1" >&2
	exit 1
}

need() {
	command -v "$1" >/dev/null 2>&1 || die "this installer needs '$1' and could not find it"
}

need uname
need tar
need mkdir
need install

# ---------------------------------------------------------------- downloader

if command -v curl >/dev/null 2>&1; then
	fetch() { curl -fsSL -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
	fetch() { wget -qO "$2" "$1"; }
else
	die "this installer needs 'curl' or 'wget' and could not find either"
fi

# ------------------------------------------------------------------ platform

os=$(uname -s)
case "$os" in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) die "unsupported operating system: $os (releases are built for Linux and macOS)" ;;
esac

arch=$(uname -m)
case "$arch" in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) die "unsupported architecture: $arch (releases are built for amd64 and arm64)" ;;
esac

archive="${BIN}_${VERSION}_${os}_${arch}.tar.gz"
base="https://github.com/${REPO}/releases/download/${VERSION}"

say "Installing ${C_BOLD}${BIN}${C_OFF} ${VERSION} for ${os}/${arch}"

# ------------------------------------------------------------------ download

tmp=$(mktemp -d 2>/dev/null || mktemp -d -t nfi)
trap 'rm -rf "$tmp"' EXIT INT TERM

fetch "${base}/${archive}" "${tmp}/${archive}" ||
	die "could not download ${base}/${archive}
      check that the release tag '${VERSION}' exists and has an asset for ${os}/${arch}"
ok "downloaded $archive"

# ------------------------------------------------------------------- verify

if fetch "${base}/checksums.txt" "${tmp}/checksums.txt" 2>/dev/null; then
	if command -v sha256sum >/dev/null 2>&1; then
		sum=$(sha256sum "${tmp}/${archive}" | cut -d' ' -f1)
	elif command -v shasum >/dev/null 2>&1; then
		sum=$(shasum -a 256 "${tmp}/${archive}" | cut -d' ' -f1)
	else
		sum=''
		note "no sha256sum or shasum available; skipping checksum verification"
	fi

	if [ -n "$sum" ]; then
		# Entries are "<digest>  ./<archive>"; match on the trailing name so a
		# leading "./" (or its absence) does not matter.
		want=$(awk -v a="$archive" '{ n = $2; sub(/^\.\//, "", n); if (n == a) print $1 }' \
			"${tmp}/checksums.txt")
		[ -n "$want" ] || die "no checksum listed for $archive"
		[ "$sum" = "$want" ] || die "checksum mismatch for $archive
      expected $want
      got      $sum"
		ok "checksum verified"
	fi
else
	note "checksums.txt unavailable; installing without verification"
fi

# ------------------------------------------------------------------ install

tar -xzf "${tmp}/${archive}" -C "$tmp" || die "could not extract $archive"

# The archive holds a single top-level directory; find the binary rather than
# assuming the layout.
binpath=$(find "$tmp" -type f -name "$BIN" -print 2>/dev/null | head -n 1)
[ -n "$binpath" ] || die "no '$BIN' binary found inside $archive"

mkdir -p "$INSTALL_DIR" || die "could not create $INSTALL_DIR"
install -m 0755 "$binpath" "${INSTALL_DIR}/${BIN}" ||
	die "could not write ${INSTALL_DIR}/${BIN}
      pick a writable location with NFI_INSTALL_DIR=/some/dir"
ok "installed ${C_BOLD}${INSTALL_DIR}/${BIN}${C_OFF}"

# -------------------------------------------------------------------- report

case ":${PATH}:" in
*":${INSTALL_DIR}:"*) ;;
*)
	note ""
	note "${INSTALL_DIR} is not on your PATH. Add this to your shell config:"
	note ""
	note "    export PATH=\"${INSTALL_DIR}:\$PATH\""
	;;
esac

if [ -x "${INSTALL_DIR}/${BIN}" ]; then
	printf '\n' >&2
	"${INSTALL_DIR}/${BIN}" --version >&2 || true
fi

printf '\n' >&2
say "Next: create a config, then preview before installing"
note ""
note "    mkdir -p ~/.config/${BIN}"
note "    \$EDITOR ~/.config/${BIN}/config.yaml"
note ""
note "    ${BIN} --dry-run"
note "    ${BIN}"
note ""
note "Or skip the config and let the picker build one:"
note ""
note "    ${BIN} --interactive"
note ""
note "Docs: https://worxbend.github.io/${BIN}/"
