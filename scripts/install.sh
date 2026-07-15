#!/bin/sh
# hookdrop CLI installer
#   curl -fsSL https://raw.githubusercontent.com/EOEboh/hookdrop/main/scripts/install.sh | sh
#
# Downloads the latest release for this OS/arch from GitHub Releases,
# verifies its sha256 against checksums.txt, and installs to
# /usr/local/bin (if writable) or ~/.local/bin.
set -eu

REPO="EOEboh/hookdrop"
BINARY="hookdrop"

os=$(uname -s)
case "$os" in
  Darwin) os="darwin" ;;
  Linux)  os="linux" ;;
  *) echo "Unsupported OS: $os (use GitHub Releases directly: https://github.com/$REPO/releases)"; exit 1 ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64)  arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "Unsupported architecture: $arch"; exit 1 ;;
esac

echo "Finding the latest release…"
tag=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
[ -n "$tag" ] || { echo "Could not determine the latest release."; exit 1; }
version=${tag#v}

archive="${BINARY}_${version}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$tag"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $archive ($tag)…"
curl -fsSL -o "$tmp/$archive" "$base/$archive"
curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt"

echo "Verifying checksum…"
expected=$(grep " $archive\$" "$tmp/checksums.txt" | awk '{print $1}')
[ -n "$expected" ] || { echo "No checksum found for $archive"; exit 1; }
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$tmp/$archive" | awk '{print $1}')
else
  actual=$(shasum -a 256 "$tmp/$archive" | awk '{print $1}')
fi
[ "$expected" = "$actual" ] || { echo "Checksum mismatch — aborting."; exit 1; }

tar -xzf "$tmp/$archive" -C "$tmp" "$BINARY"

if [ -w /usr/local/bin ]; then
  dest="/usr/local/bin"
else
  dest="$HOME/.local/bin"
  mkdir -p "$dest"
fi

install -m 0755 "$tmp/$BINARY" "$dest/$BINARY"
echo "Installed $BINARY $tag to $dest/$BINARY"

case ":$PATH:" in
  *":$dest:"*) ;;
  *) echo "Note: $dest is not on your PATH — add it to your shell profile." ;;
esac

echo "Run '$BINARY login' to get started."
