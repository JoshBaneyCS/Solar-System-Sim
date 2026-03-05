#!/usr/bin/env bash
set -euo pipefail

# Linux packaging script: creates a .tar.gz with binary, assets, and desktop file
# Usage: bash packaging/package-linux.sh
# Expects: bin/solar-sim (GUI), bin/solar-sim-headless, assets/, Icon.png

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

GOARCH="${GOARCH:-$(go env GOARCH)}"
ARCHIVE_NAME="solar-sim-linux-${GOARCH}"
STAGING="packaging/staging-linux"

echo "Packaging Linux .tar.gz (${GOARCH})..."

# Clean staging
rm -rf "$STAGING" "${ARCHIVE_NAME}.tar.gz"
mkdir -p "$STAGING/$ARCHIVE_NAME"

# Copy binaries
cp bin/solar-sim "$STAGING/$ARCHIVE_NAME/"
if [ -f bin/solar-sim-headless ]; then
    cp bin/solar-sim-headless "$STAGING/$ARCHIVE_NAME/"
fi

# Copy assets
if [ -d assets ]; then
    cp -r assets "$STAGING/$ARCHIVE_NAME/assets"
fi

# Copy desktop file and icon
cp packaging/solar-sim.desktop "$STAGING/$ARCHIVE_NAME/"
if [ -f Icon.png ]; then
    cp Icon.png "$STAGING/$ARCHIVE_NAME/solar-sim.png"
fi

# Create README
cat > "$STAGING/$ARCHIVE_NAME/README.txt" << 'EOF'
Solar System Simulator
======================

Run the GUI:
  ./solar-sim gui

Run headless simulation:
  ./solar-sim-headless run --years 1 --export output.csv

Optional: Install desktop integration:
  cp solar-sim.desktop ~/.local/share/applications/
  cp solar-sim.png ~/.local/share/icons/

System requirements:
  - OpenGL 2.1+ (mesa or proprietary drivers)
  - X11 or Wayland

For more information, see: https://github.com/joshbaney/solar-system-simulator
EOF

# Create tarball
cd "$STAGING"
tar czf "$PROJECT_DIR/${ARCHIVE_NAME}.tar.gz" "$ARCHIVE_NAME"
cd "$PROJECT_DIR"

# Clean up staging
rm -rf "$STAGING"

echo "Created: ${ARCHIVE_NAME}.tar.gz"
