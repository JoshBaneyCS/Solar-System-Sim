# Install & Build Guide

## Quick Install (Pre-built Binaries)

Download the latest release from [GitHub Releases](https://github.com/joshbaney/solar-system-simulator/releases).

| Platform | File | Instructions |
|----------|------|-------------|
| macOS (Apple Silicon) | `solar-sim-darwin-arm64.dmg` | Open DMG, drag app to Applications |
| macOS (Intel) | `solar-sim-darwin-amd64.dmg` | Open DMG, drag app to Applications |
| Linux (x86_64) | `solar-sim-linux-amd64.tar.gz` | Extract, run `./solar-sim gui` |
| Windows (x86_64) | `solar-sim-windows-amd64.zip` | Extract, run `solar-sim.exe gui` |

Each archive includes both `solar-sim` (GUI) and `solar-sim-headless` (CLI-only) binaries, plus the `assets/` directory.

**macOS note:** The app is not signed/notarized. On first launch, right-click the app and select "Open", then click "Open" in the dialog.

---

## Build from Source

### Prerequisites

- **Go 1.21+** — [go.dev/dl](https://go.dev/dl/)
- **Make** — GNU Make
- **Rust** (optional) — only needed for `rust_physics` or `rust_render` features. Install via [rustup.rs](https://rustup.rs/)

### macOS

```bash
# Install Xcode command-line tools (provides C compiler for CGO)
xcode-select --install

# Build and run
make build-solar-sim
./bin/solar-sim gui
```

### Linux (Ubuntu/Debian)

```bash
# Install system dependencies (Fyne requires OpenGL and X11 headers)
sudo apt-get update
sudo apt-get install -y build-essential pkg-config libgl1-mesa-dev \
    xorg-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev

# Build and run
make build-solar-sim
./bin/solar-sim gui
```

### Windows

```powershell
# Install MinGW-w64 (provides GCC for CGO, required by Fyne)
choco install mingw -y

# Build and run
make build-solar-sim
.\bin\solar-sim.exe gui
```

### Headless (no GUI dependencies)

Build a CLI-only binary that doesn't require any graphics libraries:

```bash
make build-solar-sim-headless
./bin/solar-sim run --years 1 --export output.csv
./bin/solar-sim validate --scenario all --years 10
./bin/solar-sim launch --dest mars --vehicle falcon
```

### With Rust Physics Backend

```bash
# Requires Rust toolchain
make rust-build
make build-rust
./bin/solar-system-sim
```

### With GPU Rendering (Rust + wgpu)

```bash
# Requires Rust toolchain + GPU drivers
make rust-build
make render-build
make build-gpu
./bin/solar-system-sim
```

---

## CLI Usage

After building or installing:

```bash
solar-sim gui                                    # Launch GUI
solar-sim run --years 1 --export out.csv         # Headless simulation export
solar-sim validate --scenario mercury-precession  # Physics validation
solar-sim launch --dest leo --export launch.csv   # Launch planner
solar-sim assets verify                           # Validate asset directory
```

See [CLI.md](CLI.md) for full command reference.

---

## Packaging Locally

To create platform-specific packages from a local build:

```bash
# First build both binaries
make build-solar-sim
CGO_ENABLED=0 go build -tags nogui -o bin/solar-sim-headless ./cmd/solar-sim

# Then package for your platform
make package-macos    # Creates .dmg
make package-linux    # Creates .tar.gz
make package-windows  # Creates .zip
```

---

## CI/CD

Releases are built automatically via GitHub Actions when a version tag is pushed:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This triggers the [release workflow](../.github/workflows/release.yml) which builds and packages for all platforms, then creates a GitHub Release with the artifacts attached.
