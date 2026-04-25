#!/usr/bin/env bash
# Copies goreleaser-built burrowctl binaries into npm platform packages and publishes.
# Run after goreleaser has produced artifacts in dist/.
# Usage: VERSION=0.1.0 ./scripts/npm-publish.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:?VERSION is required}"

# goreleaser build output dirs (amd64 gets _v1 suffix for GOAMD64 microarch)
declare -A GORELEASER_DIRS=(
  ["linux-x64"]="dist/burrowctl_linux_amd64_v1"
  ["linux-arm64"]="dist/burrowctl_linux_arm64"
  ["darwin-x64"]="dist/burrowctl_darwin_amd64_v1"
  ["darwin-arm64"]="dist/burrowctl_darwin_arm64"
)

echo "=== npm publish burrowctl@${VERSION} ==="
echo ""

# --- platform packages ---
for PLATFORM in linux-x64 linux-arm64 darwin-x64 darwin-arm64; do
  PKG_DIR="${ROOT}/npm/burrowctl-${PLATFORM}"
  BIN_SRC="${ROOT}/${GORELEASER_DIRS[$PLATFORM]}/burrowctl"

  echo "--- ${PLATFORM} ---"

  if [[ ! -f "$BIN_SRC" ]]; then
    echo "✗ Binary not found: ${BIN_SRC}" >&2
    exit 1
  fi

  cp "$BIN_SRC" "${PKG_DIR}/burrowctl"
  chmod +x "${PKG_DIR}/burrowctl"

  node -e "
    const fs = require('fs');
    const pkg = JSON.parse(fs.readFileSync('${PKG_DIR}/package.json'));
    pkg.version = '${VERSION}';
    fs.writeFileSync('${PKG_DIR}/package.json', JSON.stringify(pkg, null, 2) + '\n');
  "

  (cd "$PKG_DIR" && npm publish --access public)
  echo "✓ Published @miniaxolotl/burrowctl-${PLATFORM}@${VERSION}"
done

echo ""

# --- main package ---
MAIN_DIR="${ROOT}/npm/burrowctl"

node -e "
  const fs = require('fs');
  const pkg = JSON.parse(fs.readFileSync('${MAIN_DIR}/package.json'));
  pkg.version = '${VERSION}';
  for (const dep of Object.keys(pkg.optionalDependencies)) {
    pkg.optionalDependencies[dep] = '${VERSION}';
  }
  fs.writeFileSync('${MAIN_DIR}/package.json', JSON.stringify(pkg, null, 2) + '\n');
"

(cd "$MAIN_DIR" && npm publish --access public)
echo "✓ Published @miniaxolotl/burrowctl@${VERSION}"
echo ""
echo "✓ Done"
