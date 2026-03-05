# Install & Build Guide (macOS / Linux / Windows)

This guide explains how to build and run the Solar System Simulator from source **and** how packaged installers are produced.

## 1. From Source

### Toolchains
- Go 1.21+
- Rust stable (`rustup`)

### macOS
1. Install Xcode CLT:
   ```bash
   xcode-select --install
   ```
2. Build:
   ```bash
   make rust
   make build
   ```
3. Run:
   ```bash
   ./bin/SolarSim
   # or:
   solar-sim gui
   ```

### Linux (Ubuntu/Debian)
1. Install dependencies:
   ```bash
   sudo apt-get update
   sudo apt-get install -y build-essential pkg-config libssl-dev      libx11-dev libxrandr-dev libxi-dev libxcursor-dev libxinerama-dev
   ```
2. Build & run:
   ```bash
   make rust
   make build
   ./bin/SolarSim
   ```

### Windows
1. Install Go and Rust (MSVC toolchain recommended).
2. Build:
   ```powershell
   make rust
   make build
   ```
3. Run:
   ```powershell
   .\bin\SolarSim.exe
   ```

## 2. CLI / Headless
After building:
```bash
solar-sim --help
solar-sim run --years 1 --export out.csv
solar-sim validate --scenario mercury-precession --years 100
solar-sim launch --dest leo --export launch.csv
```

## 3. Installers (Release Artifacts)

### macOS
- `.dmg` (drag & drop) and optional `.pkg`
- Optional codesigning/notarization steps documented in `packaging/macos/`.

### Windows
- `.msi` (recommended) or `.exe` installer
- Installs GUI + CLI and places `solar-sim` on PATH (optional).

### Linux
- `.AppImage` (recommended)
- Optional `.deb` packages.

## 4. Dependency Bundling Policy

- Prefer **bundling** Rust dynamic libraries and GPU runtime libs required by the app.
- If a system package is required (Linux graphics libs), installer should detect and either:
  - prompt user, or
  - install via package manager (documented).
