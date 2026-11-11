#!/bin/sh
#
# Installs a Weave release on Linux or macOS:
#
#   curl -fsSL https://raw.githubusercontent.com/Kazuto/weave/main/install.sh | sh
#
# Environment:
#   WEAVE_INSTALL_DIR  Where to install (default: ~/.local/bin)
#   WEAVE_VERSION      Release to install, such as v0.1.0 (default: latest)

set -eu

RELEASES_URL="https://github.com/Kazuto/weave/releases"

tmp=""
staged=""

fail() {
	printf 'Error: %s\n' "$1" >&2
	exit 1
}

cleanup() {
	if [ -n "$tmp" ]; then
		rm -rf "$tmp"
	fi
	if [ -n "$staged" ]; then
		rm -f "$staged"
	fi
}

fetch() {
	curl -fsSL --proto '=https' --tlsv1.2 "$@"
}

detect_os() {
	case "$(uname -s)" in
	Linux) echo linux ;;
	Darwin) echo darwin ;;
	MINGW* | MSYS* | CYGWIN*) fail "this script doesn't support Windows. Download a release from $RELEASES_URL" ;;
	*) fail "unsupported operating system: $(uname -s)" ;;
	esac
}

detect_arch() {
	case "$(uname -m)" in
	x86_64 | amd64) echo amd64 ;;
	aarch64 | arm64) echo arm64 ;;
	*) fail "unsupported architecture: $(uname -m)" ;;
	esac
}

is_release_version() {
	case "$1" in
	*[!v0-9.]*) return 1 ;;
	esac
	printf '%s\n' "$1" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'
}

latest_version() {
	url=$(fetch -I -o /dev/null -w '%{url_effective}' "$RELEASES_URL/latest") || fail "couldn't look up the latest release"
	printf '%s\n' "${url##*/}"
}

sha256_of() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d ' ' -f 1
	else
		shasum -a 256 "$1" | cut -d ' ' -f 1
	fi
}

# Warns when `weave` won't run the copy that was just installed.
check_path() {
	found=$(command -v weave 2>/dev/null || true)
	if [ "$found" = "$1/weave" ]; then
		return
	fi

	if [ -n "$found" ]; then
		printf '\nWarning: %s comes first on your PATH, so `weave` still runs that copy.\n' "$found"
		printf 'Remove it, or put %s earlier on your PATH.\n' "$1"
	else
		printf '\n%s is not on your PATH. Add this to your shell profile:\n' "$1"
		printf '  export PATH="%s:$PATH"\n' "$1"
	fi
}

main() {
	for tool in curl tar gzip awk grep cut uname mktemp; do
		command -v "$tool" >/dev/null 2>&1 || fail "$tool is required"
	done
	if ! command -v sha256sum >/dev/null 2>&1 && ! command -v shasum >/dev/null 2>&1; then
		fail "sha256sum or shasum is required"
	fi

	os=$(detect_os)
	arch=$(detect_arch)

	version="${WEAVE_VERSION:-}"
	if [ -z "$version" ]; then
		version=$(latest_version)
	fi
	is_release_version "$version" || fail "invalid release version: $version"

	archive="weave-$version-$os-$arch.tar.gz"
	binary="weave-$os-$arch"
	install_dir="${WEAVE_INSTALL_DIR:-${HOME:-}/.local/bin}"
	if [ -z "${WEAVE_INSTALL_DIR:-}" ] && [ -z "${HOME:-}" ]; then
		fail "HOME is not set. Choose a directory with WEAVE_INSTALL_DIR"
	fi

	tmp=$(mktemp -d)

	printf 'Downloading weave %s for %s/%s\n' "$version" "$os" "$arch"
	fetch -o "$tmp/checksums.txt" "$RELEASES_URL/download/$version/checksums.txt" ||
		fail "couldn't download checksums for $version. Its files may still be uploading, so try again in a few minutes. Releases up to v0.0.10 don't have checksums"
	fetch -o "$tmp/$archive" "$RELEASES_URL/download/$version/$archive" ||
		fail "couldn't download $archive. Its files may still be uploading, or $os/$arch may not be supported"

	expected=$(awk -v name="$archive" '$2 == name || $2 == "*" name { print $1 }' "$tmp/checksums.txt")
	[ -n "$expected" ] || fail "no checksum published for $archive"
	actual=$(sha256_of "$tmp/$archive")
	[ "$actual" = "$expected" ] || fail "checksum mismatch for $archive: expected $expected, got $actual"

	tar -xzf "$tmp/$archive" -C "$tmp" "$binary" || fail "couldn't extract $binary from $archive"

	permission_hint="Choose another directory with WEAVE_INSTALL_DIR, or run this script with sudo"
	mkdir -p "$install_dir" 2>/dev/null || fail "can't create $install_dir. $permission_hint"
	install_dir=$(cd "$install_dir" && pwd)

	# Stage next to the target so the final rename is atomic, and because /tmp may be mounted noexec.
	staged="$install_dir/.weave-install.$$"
	cp "$tmp/$binary" "$staged" 2>/dev/null || fail "can't write to $install_dir. $permission_hint"
	chmod 755 "$staged"

	# This script may be running from a pipe, so don't let the binary read it as input.
	reported=$("$staged" version </dev/null 2>/dev/null) || fail "the downloaded binary failed to run on this system"
	[ "$reported" = "weave $version" ] || fail "the downloaded binary doesn't report version $version"

	mv -f "$staged" "$install_dir/weave"
	staged=""

	printf 'Installed weave %s to %s/weave\n' "$version" "$install_dir"
	check_path "$install_dir"
}

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# Everything runs from here, so a partial download can't run a partial script.
main
