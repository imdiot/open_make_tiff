#!/bin/bash
set -euo pipefail

# Wails runs this hook from build/bin (CWD = build/bin).
APP="./OpenMakeTiff.app"

# 1) Clear both the old (MacOS) and new (Resources) locations to avoid stale leftovers.
rm -rf "${APP}/Contents/MacOS/third-party"
rm -rf "${APP}/Contents/Resources/third-party"

# 2) Place third-party under Resources/ so codesign treats it as data, not code.
#    exiftool and lib/ must stay siblings: the script resolves its lib/ via its own path.
cp -R ../../third-party/macos-universal "${APP}/Contents/Resources/third-party"

# 3) Ensure exiftool is executable (cp -R preserves it; make it explicit).
chmod 0755 "${APP}/Contents/Resources/third-party/exiftool"

# 4) Drop any .DS_Store so sealed-resource hashes stay stable.
find "${APP}" -name '.DS_Store' -delete

# 5) Re-sign. Wails signed the bundle before this hook, but copying files in
#    afterwards invalidates that signature, so --force replaces it.
#    No Hardened Runtime (would kill the unsigned exiftool child under adhoc),
#    no --timestamp (meaningless for adhoc), no entitlements.
codesign --force --deep --sign - "${APP}"

# 6) Fail the build if the signature is invalid — never ship a broken bundle.
codesign --verify --deep --strict --verbose=2 "${APP}"

# 7) Log signing info for the build output (expect Flags=adhoc).
codesign -dv --verbose=2 "${APP}"

echo "✓ third-party copied to Resources/ and bundle adhoc-signed"
