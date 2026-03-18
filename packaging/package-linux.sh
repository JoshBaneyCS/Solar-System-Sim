#!/usr/bin/env bash
set -euo pipefail

# Linux packaging script: creates a .tar.gz with binary and desktop file
# Usage: bash packaging/package-linux.sh
# Expects: bin/solar-sim (Bevy GUI binary with embedded assets)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

ARCH="${GOARCH:-amd64}"
ARCHIVE_NAME="solar-sim-linux-${ARCH}"
STAGING="packaging/staging-linux"

echo "Packaging Linux .tar.gz (${ARCH})..."

# Clean staging
rm -rf "$STAGING" "${ARCHIVE_NAME}.tar.gz"
mkdir -p "$STAGING/$ARCHIVE_NAME"

# Copy binary (assets are embedded)
cp bin/solar-sim "$STAGING/$ARCHIVE_NAME/"
chmod +x "$STAGING/$ARCHIVE_NAME/solar-sim"

if [ -f bin/solar-sim-headless ]; then
    cp bin/solar-sim-headless "$STAGING/$ARCHIVE_NAME/"
    chmod +x "$STAGING/$ARCHIVE_NAME/solar-sim-headless"
fi

# Copy desktop file and icon
if [ -f packaging/solar-sim.desktop ]; then
    cp packaging/solar-sim.desktop "$STAGING/$ARCHIVE_NAME/"
fi
if [ -f Icon.png ]; then
    cp Icon.png "$STAGING/$ARCHIVE_NAME/solar-sim.png"
fi

# Create README
cat > "$STAGING/$ARCHIVE_NAME/README.txt" << 'EOF'
Solar System Simulator v0.1.5
=============================

Run the simulator:
  ./solar-sim

All textures and assets are embedded in the binary.

Optional: Install desktop integration:
  cp solar-sim.desktop ~/.local/share/applications/
  cp solar-sim.png ~/.local/share/icons/

System requirements:
  - Vulkan or OpenGL 3.3+ (mesa or proprietary drivers)
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
