#!/usr/bin/env bash
set -euo pipefail

# Windows packaging script: creates a .zip with .exe
# Usage: bash packaging/package-windows.sh
# Expects: bin/solar-sim.exe (Bevy GUI binary with embedded assets)
# Runs under Git Bash on Windows CI runners

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

ARCH="${GOARCH:-amd64}"
ARCHIVE_NAME="solar-sim-windows-${ARCH}"
STAGING="packaging/staging-windows"

echo "Packaging Windows .zip (${ARCH})..."

# Clean staging
rm -rf "$STAGING" "${ARCHIVE_NAME}.zip"
mkdir -p "$STAGING/$ARCHIVE_NAME"

# Copy binary (assets are embedded)
cp bin/solar-sim.exe "$STAGING/$ARCHIVE_NAME/"

if [ -f bin/solar-sim-headless.exe ]; then
    cp bin/solar-sim-headless.exe "$STAGING/$ARCHIVE_NAME/"
fi

# Copy icon
if [ -f Icon.png ]; then
    cp Icon.png "$STAGING/$ARCHIVE_NAME/solar-sim.png"
fi

# Create README
cat > "$STAGING/$ARCHIVE_NAME/README.txt" << 'EOF'
Solar System Simulator v0.1.5
=============================

Run the simulator:
  Double-click solar-sim.exe

All textures and assets are embedded in the binary.

System requirements:
  - Windows 10 or later
  - DirectX 12, Vulkan, or OpenGL 3.3+ capable GPU

For more information, see: https://github.com/joshbaney/solar-system-simulator
EOF

# Create zip using PowerShell (available on Windows runners)
if command -v powershell.exe &>/dev/null; then
    WIN_SRC=$(cygpath -w "$STAGING/$ARCHIVE_NAME")
    WIN_DST=$(cygpath -w "$(pwd)/${ARCHIVE_NAME}.zip")
    powershell.exe -Command "Compress-Archive -Path '${WIN_SRC}\*' -DestinationPath '${WIN_DST}' -Force"
elif command -v zip &>/dev/null; then
    cd "$STAGING"
    zip -r "$PROJECT_DIR/${ARCHIVE_NAME}.zip" "$ARCHIVE_NAME"
    cd "$PROJECT_DIR"
else
    echo "Error: Neither PowerShell nor zip available for creating archive"
    exit 1
fi

# Clean up staging
rm -rf "$STAGING"

echo "Created: ${ARCHIVE_NAME}.zip"
