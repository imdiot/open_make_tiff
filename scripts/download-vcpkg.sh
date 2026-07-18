#!/usr/bin/env bash
# Usage: download-vcpkg.sh <vcpkg_ref>   (tag or commit of microsoft/vcpkg)
# Clones microsoft/vcpkg (pinned, shallow) into third-party/vcpkg-root/ and
# bootstraps the tool. Not a git submodule; the clone is gitignored.
set -euo pipefail

REF="${1:?vcpkg ref (tag/commit) required}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
VCPKG_ROOT_DIR="$PROJECT_ROOT/third-party/vcpkg-root"
BIN="$VCPKG_ROOT_DIR/vcpkg"
MARKER="$VCPKG_ROOT_DIR/.vcpkg-ref"

if [ -f "$MARKER" ] && [ "$(cat "$MARKER")" = "$REF" ] && [ -x "$BIN" ]; then
    echo "vcpkg ($REF) already present at $VCPKG_ROOT_DIR"
    exit 0
fi

echo "Cloning microsoft/vcpkg @ $REF into $VCPKG_ROOT_DIR ..."
rm -rf "$VCPKG_ROOT_DIR"
if ! git clone --depth 1 --branch "$REF" https://github.com/microsoft/vcpkg.git "$VCPKG_ROOT_DIR" 2>/dev/null; then
    # REF is likely a commit, not a tag/branch — shallow clone then fetch it
    git clone --depth 1 https://github.com/microsoft/vcpkg.git "$VCPKG_ROOT_DIR"
    git -C "$VCPKG_ROOT_DIR" fetch --depth 1 origin "$REF"
    git -C "$VCPKG_ROOT_DIR" checkout "$REF"
fi

echo "Bootstrapping vcpkg-tool ..."
( cd "$VCPKG_ROOT_DIR" && bash ./bootstrap-vcpkg.sh -disableMetrics )

echo "$REF" > "$MARKER"
echo "vcpkg ($REF) ready at $VCPKG_ROOT_DIR"
