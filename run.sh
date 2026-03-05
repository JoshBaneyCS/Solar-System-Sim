#!/bin/bash

# Solar System Simulator - Quick Start Script

echo "🌟 Solar System Simulator Setup 🌟"
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or later."
    echo "Visit: https://golang.org/dl/"
    exit 1
fi

echo "✅ Go detected: $(go version)"
echo ""

# Check for dependencies on different platforms
OS="$(uname -s)"
case "$OS" in
    Darwin*)
        echo "📦 Platform: macOS"
        echo "Note: Xcode command line tools should be installed"
        ;;
    Linux*)
        echo "📦 Platform: Linux"
        echo "Checking for required packages..."
        if ! dpkg -l | grep -q libgl1-mesa-dev; then
            echo "⚠️  Missing dependencies. Install with:"
            echo "   sudo apt-get install gcc libgl1-mesa-dev xorg-dev"
        else
            echo "✅ Dependencies found"
        fi
        ;;
    *)
        echo "📦 Platform: $OS"
        ;;
esac

echo ""
echo "📥 Fetching dependencies..."
go mod tidy

if [ $? -eq 0 ]; then
    echo "✅ Dependencies downloaded"
else
    echo "❌ Failed to download dependencies"
    exit 1
fi

echo ""
echo "🔨 Building Solar System Simulator..."
go build -o solar_system_sim solar_system_sim.go

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo ""
    echo "🚀 Launching Solar System Simulator..."
    echo ""
    ./main
else
    echo "❌ Build failed"
    exit 1
fi