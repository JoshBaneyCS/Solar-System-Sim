#!/usr/bin/env bash
set -euo pipefail

# Windows packaging script: creates a .zip with .exe, assets
# Usage: bash packaging/package-windows.sh
# Expects: bin/solar-sim.exe, bin/solar-sim-headless.exe, assets/
# Runs under Git Bash on Windows CI runners

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

GOARCH="${GOARCH:-amd64}"
ARCHIVE_NAME="solar-sim-windows-${GOARCH}"
STAGING="packaging/staging-windows"

echo "Packaging Windows .zip (${GOARCH})..."

# Clean staging
rm -rf "$STAGING" "${ARCHIVE_NAME}.zip"
mkdir -p "$STAGING/$ARCHIVE_NAME"

# Copy binaries
cp bin/solar-sim.exe "$STAGING/$ARCHIVE_NAME/"
if [ -f bin/solar-sim-headless.exe ]; then
    cp bin/solar-sim-headless.exe "$STAGING/$ARCHIVE_NAME/"
fi

# Copy assets
if [ -d assets ]; then
    cp -r assets "$STAGING/$ARCHIVE_NAME/assets"
fi

# Copy icon
if [ -f Icon.png ]; then
    cp Icon.png "$STAGING/$ARCHIVE_NAME/solar-sim.png"
fi

# Create README
cat > "$STAGING/$ARCHIVE_NAME/README.txt" << 'EOF'
Solar System Simulator
======================

Run the GUI:
  solar-sim.exe gui

Run headless simulation:
  solar-sim-headless.exe run --years 1 --export output.csv

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
