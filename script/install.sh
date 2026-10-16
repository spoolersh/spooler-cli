#!/bin/sh
 
# Installs the spooler binary from a release on GitHub:
#
#   curl -fsSL https://raw.githubusercontent.com/spoolersh/spooler-cli/main/script/install.sh | sh
#
# SPOOLER_VERSION picks the release (0.1.0), the latest by default;
# SPOOLER_INSTALL_DIR picks the directory, ~/.local/bin by default. The
# archive is checked against the release's checksums.txt before anything is
# written. Nothing else is touched: no sudo, no shell configuration.
#
# The body is a function called on the last line, so a download cut short
# runs nothing.
set -eu

repo=spoolersh/spooler-cli

fail() {
	echo "install.sh: $*" >&2
	exit 1
}

# fetch downloads url to the file out, over https only.
fetch() {
	if command -v curl >/dev/null 2>&1; then
		curl --proto '=https' --tlsv1.2 -fsSL -o "$2" "$1"
	elif command -v wget >/dev/null 2>&1; then
		wget --https-only -qO "$2" "$1"
	else
		fail "neither curl nor wget is installed"
	fi
}

# sha256 prints the SHA-256 of the file, with whichever tool the system has.
sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d ' ' -f 1
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | cut -d ' ' -f 1
	else
		fail "neither sha256sum nor shasum is installed"
	fi
}

main() {
	os=$(uname -s | tr '[:upper:]' '[:lower:]')
	case $os in
	linux | darwin) ;;
	*) fail "no release binary for the operating system $os" ;;
	esac

	arch=$(uname -m)
	case $arch in
	x86_64 | amd64) arch=amd64 ;;
	aarch64 | arm64) arch=arm64 ;;
	*) fail "no release binary for the architecture $arch" ;;
	esac

	version=${SPOOLER_VERSION:-}
	version=${version#v}
	if [ -n "$version" ]; then
		release="https://github.com/$repo/releases/download/v$version"
	else
		release="https://github.com/$repo/releases/latest/download"
	fi
	dir=${SPOOLER_INSTALL_DIR:-$HOME/.local/bin}

	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT

	[ ! -d "$dir/spooler" ] ||
		fail "$dir/spooler is a directory"

	# checksums.txt has a fixed name, so it is fetched first and names the
	# archive, which carries the version; that is how "latest" is resolved,
	# once: the archive is then fetched from that release by its tag, so a
	# release landing in between cannot split the two.
	fetch "$release/checksums.txt" "$tmp/checksums.txt" ||
		fail "no release found at $release"
	name=$(awk -v s="_${os}_${arch}.tar.gz" \
		'index($2, s) == length($2) - length(s) + 1 { print $2 }' \
		"$tmp/checksums.txt")
	[ -n "$name" ] || fail "no archive for ${os}/${arch} in the release"
	sum=$(awk -v n="$name" '$2 == n { print $1 }' "$tmp/checksums.txt")
	version=${name#spooler-cli_}
	version=${version%_"${os}"_"${arch}".tar.gz}
	release="https://github.com/$repo/releases/download/v$version"

	fetch "$release/$name" "$tmp/$name" ||
		fail "download $release/$name failed"
	act=$(sha256 "$tmp/$name")
	[ "$act" = "$sum" ] ||
		fail "checksum of $name is $act; want $sum"

	tar -xzf "$tmp/$name" -C "$tmp" spooler
	mkdir -p "$dir"
	install -m 0755 "$tmp/spooler" "$dir/spooler"

	echo "installed spooler v$version to $dir/spooler"
	case ":$PATH:" in
	*":$dir:"*) ;;
	*) echo "$dir is not on PATH" ;;
	esac
}

main
