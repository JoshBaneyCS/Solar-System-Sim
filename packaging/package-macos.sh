#!/usr/bin/env bash
set -euo pipefail

# macOS packaging script: creates a .dmg containing .app bundle + headless binary
# Usage: bash packaging/package-macos.sh
# Expects: bin/solar-sim (GUI), bin/solar-sim-headless, assets/, Icon.png

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

GOARCH="${GOARCH:-$(go env GOARCH)}"
APP_NAME="Solar System Simulator"
BUNDLE_NAME="SolarSystemSimulator.app"
DMG_NAME="solar-sim-darwin-${GOARCH}.dmg"
STAGING="packaging/staging-macos"

echo "Packaging macOS .dmg (${GOARCH})..."

# Clean staging
rm -rf "$STAGING" "$DMG_NAME"
mkdir -p "$STAGING/$BUNDLE_NAME/Contents/MacOS"
mkdir -p "$STAGING/$BUNDLE_NAME/Contents/Resources"

# Copy GUI binary
cp bin/solar-sim "$STAGING/$BUNDLE_NAME/Contents/MacOS/solar-sim"

# Copy assets into Resources
if [ -d assets ]; then
    cp -r assets "$STAGING/$BUNDLE_NAME/Contents/Resources/assets"
fi

# Create Info.plist
cat > "$STAGING/$BUNDLE_NAME/Contents/Info.plist" << 'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>Solar System Simulator</string>
    <key>CFBundleDisplayName</key>
    <string>Solar System Simulator</string>
    <key>CFBundleIdentifier</key>
    <string>com.joshbaney.solar-sim</string>
    <key>CFBundleVersion</key>
    <string>1.0.0</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>
    <key>CFBundleExecutable</key>
    <string>solar-sim</string>
    <key>CFBundleIconFile</key>
    <string>icon</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSMinimumSystemVersion</key>
    <string>12.0</string>
</dict>
</plist>
PLIST

# Convert Icon.png to .icns if possible
if [ -f Icon.png ]; then
    ICONSET_DIR=$(mktemp -d)/icon.iconset
    mkdir -p "$ICONSET_DIR"
    sips -z 16 16     Icon.png --out "$ICONSET_DIR/icon_16x16.png"      2>/dev/null || true
    sips -z 32 32     Icon.png --out "$ICONSET_DIR/icon_16x16@2x.png"   2>/dev/null || true
    sips -z 32 32     Icon.png --out "$ICONSET_DIR/icon_32x32.png"      2>/dev/null || true
    sips -z 64 64     Icon.png --out "$ICONSET_DIR/icon_32x32@2x.png"   2>/dev/null || true
    sips -z 128 128   Icon.png --out "$ICONSET_DIR/icon_128x128.png"    2>/dev/null || true
    sips -z 256 256   Icon.png --out "$ICONSET_DIR/icon_128x128@2x.png" 2>/dev/null || true
    sips -z 256 256   Icon.png --out "$ICONSET_DIR/icon_256x256.png"    2>/dev/null || true
    sips -z 512 512   Icon.png --out "$ICONSET_DIR/icon_256x256@2x.png" 2>/dev/null || true
    sips -z 512 512   Icon.png --out "$ICONSET_DIR/icon_512x512.png"    2>/dev/null || true
    sips -z 1024 1024 Icon.png --out "$ICONSET_DIR/icon_512x512@2x.png" 2>/dev/null || true
    iconutil -c icns -o "$STAGING/$BUNDLE_NAME/Contents/Resources/icon.icns" "$ICONSET_DIR" 2>/dev/null || true
    rm -rf "$(dirname "$ICONSET_DIR")"
fi

# Copy headless binary alongside .app
if [ -f bin/solar-sim-headless ]; then
    cp bin/solar-sim-headless "$STAGING/solar-sim-headless"
fi

# Create symlink to Applications for drag-and-drop install
ln -s /Applications "$STAGING/Applications"

# Create DMG
hdiutil create -volname "$APP_NAME" \
    -srcfolder "$STAGING" \
    -ov -format UDZO \
    "$DMG_NAME"

# Clean up staging
rm -rf "$STAGING"

echo "Created: $DMG_NAME"
