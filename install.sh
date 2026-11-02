#!/bin/sh
# Installs O'So AI Relay for the current account, on macOS and Linux:
#
#   curl -fsSL https://github.com/novalo-dev/Somcp-Relay/releases/latest/download/install.sh | sh
#
# It downloads the archive for this computer from the latest release, checks
# it against the release's SHA256SUMS.txt, and lets oso copy itself onto the
# PATH. Nothing needs administrator rights. The script is attached to each
# release, so the line above always runs a published version of it.
set -eu

base="https://github.com/novalo-dev/Somcp-Relay/releases/latest/download"

fail() {
	echo "O'So AI Relay: $1" >&2
	exit 1
}

case "$(uname -s)" in
Darwin)
	# A shell translated by Rosetta reports x86_64 on Apple silicon: ask
	# the hardware instead, so the native build is chosen.
	if [ "$(sysctl -n hw.optional.arm64 2>/dev/null || echo 0)" = 1 ]; then
		file="oso-ai-relay-macos-apple-silicon.zip"
	else
		file="oso-ai-relay-macos-intel.zip"
	fi
	;;
Linux)
	case "$(uname -m)" in
	x86_64 | amd64) file="oso-ai-relay-linux-x64.tar.gz" ;;
	aarch64 | arm64) file="oso-ai-relay-linux-arm64.tar.gz" ;;
	*) fail "no build for this processor ($(uname -m))" ;;
	esac
	;;
*) fail "no build for this system ($(uname -s))" ;;
esac

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $file..."
curl -fsSL -o "$tmp/$file" "$base/$file" || fail "the download failed"
curl -fsSL -o "$tmp/SHA256SUMS.txt" "$base/SHA256SUMS.txt" || fail "the download failed"

# A truncated or altered download must never be run.
expected=$(grep " $file\$" "$tmp/SHA256SUMS.txt" | cut -d ' ' -f 1)
if command -v sha256sum >/dev/null 2>&1; then
	actual=$(sha256sum "$tmp/$file" | cut -d ' ' -f 1)
else
	actual=$(shasum -a 256 "$tmp/$file" | cut -d ' ' -f 1)
fi
if [ -z "$expected" ] || [ "$expected" != "$actual" ]; then
	fail "the download does not match the release checksum"
fi

case "$file" in
*.zip) unzip -q -o "$tmp/$file" -d "$tmp" ;;
*) tar -xzf "$tmp/$file" -C "$tmp" ;;
esac
chmod +x "$tmp/oso"

# oso copies itself to ~/.local/bin, adds that folder to the PATH and says
# what to type next.
"$tmp/oso" addtopath
