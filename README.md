# Solar System Simulator — GPU-Accelerated, Physically Accurate (Go GUI + Rust Kernels)

A cross-platform solar system & mission simulation toolkit:

- **Go-based GUI** (desktop: macOS / Linux / Windows)
- **Optional CLI/headless mode** for simulation, validation, and exports
- **High-fidelity physics**: 3D orbital elements, N-body gravity, and **General Relativity correction** for Mercury
- **Modern rendering**: GPU-accelerated PBR planets + starfield; optional **ray tracing** mode
- **Spacetime fabric visualization** toggle (curvature/density field)
- **Launch Planner**: simulate a launch from **Kennedy Space Center** to selectable destinations (LEO/GEO/Moon/Mars transfer)

> This README is written to support a planned refactor/restructure. It includes a clear roadmap and build/packaging story.

---

## Screenshots

_TODO: Add screenshots once the new renderer & UI panels are integrated._

---

## Features

### Physics (accurate + verifiable)
- 3D orbital mechanics from Keplerian elements
- Full N-body gravitational interactions
- GR correction (post-Newtonian) to reproduce Mercury’s perihelion precession component
- Deterministic integrators:
  - RK4 (accuracy)
  - Symplectic (long-term stability)
- Built-in validation suite:
  - energy conservation checks
  - angular momentum checks
  - perihelion precession measurement

### Rendering (beautiful + fast)
- GPU auto-detect (AMD/NVIDIA/Apple Silicon)
- Rasterized PBR planets (default)
- Optional ray tracing mode (progressive)
- Realistic assets support:
  - Earth: `.glb` model
  - Other planets: image textures mapped onto generated sphere meshes
- Asteroid belt visualization (instancing + LOD)
- Spacetime fabric grid overlay (toggleable)

### UI / UX
- Left panel: bodies manager (mass, radius, orbit, model/texture, toggles)
- Center: main canvas
- Right panel: live calculations & selected-body stats (r/v/a vectors, energy, GR term)
- Settings menu:
  - GPU acceleration: Auto/On/Off
  - Ray tracing: Auto/On/Off
  - Quality presets, integrator selection, dt/substeps, worker count
- About window:
  - author, repo, credits/references, donation links, licenses

### CLI (headless)
- `solar-sim run ...` simulate and export ephemeris
- `solar-sim validate ...` run physics verification scenarios
- `solar-sim launch ...` run the Kennedy Launch Planner and export a trajectory

---

## Project Layout (target)

```text
.
├── cmd/
│   └── solar-sim/                # CLI + GUI entrypoint
├── internal/
│   ├── app/                      # controller/state
│   ├── ui/                       # panels, settings, about
│   ├── sim/                      # Go-facing sim API (wraps Rust or Go fallback)
│   └── validation/               # physics tests + scenarios
├── crates/
│   ├── physics_core/             # Rust physics kernels + integrators
│   ├── render_core/              # Rust wgpu renderer (raster)
│   └── ray_core/                 # optional ray tracing kernels
├── assets/
│   ├── models/                   # .glb
│   ├── textures/                 # planet texture sets
│   └── backgrounds/              # starfields / HDRI
├── tools/
│   ├── meshgen/                  # sphere mesh generator (obj/glb)
│   └── validate_assets/          # asset sanity checks
├── packaging/                    # installers + goreleaser config
└── docs/                         # design docs
```

---

## Quick Start (from source)

### Prerequisites

- Go 1.21+
- Rust stable (for GPU kernels / ray tracing / asset tools)
- Git

#### macOS
- Xcode Command Line Tools:
  ```bash
  xcode-select --install
  ```

#### Linux (Ubuntu/Debian)
- Common build deps:
  ```bash
  sudo apt-get update
  sudo apt-get install -y build-essential pkg-config libssl-dev     libx11-dev libxrandr-dev libxi-dev libxcursor-dev libxinerama-dev
  ```

#### Windows
- Install **Go** and **Rust**.
- Recommended: Visual Studio Build Tools.
- If needed for native libs: MSVC toolchain (Rust will prompt).

---

## Build & Run

### 1) Build Rust crates (if enabled)
```bash
make rust
```

### 2) Build the app
```bash
make build
```

### 3) Run GUI
```bash
./bin/SolarSim
# or:
solar-sim gui
```

### 4) Run headless (CLI)
```bash
solar-sim run --years 10 --export out.csv
solar-sim validate --scenario mercury-precession --years 100
solar-sim launch --dest mars --export launch.csv
```

> See `docs/INSTALL.md` for full details and OS-specific notes.

---

## Settings

- **GPU Acceleration**
  - Auto: choose best backend available
  - On: force GPU
  - Off: CPU fallback
- **Ray Tracing**
  - Auto: enable only if hardware/driver supports it
  - On: progressive RT (may reduce FPS)
  - Off: raster mode only
- **Spacetime Fabric**
  - Toggle visualization of curvature/density field
  - Adjustable grid resolution and scale

---

## Assets

- Earth uses a `.glb` model.
- Other planets use textures mapped to generated spheres:
  - `tools/meshgen` can generate `sphere_mesh.obj` or `sphere.glb` at multiple LODs.

See `docs/ASSETS.md`.

---

## Accuracy Notes

This simulator is intended to be **physically grounded**, but visualization scales are not to scale for readability. All calculations run in SI units; display uses scaled transforms.

---

## Roadmap

1. Refactor current Go code into modular packages (no behavior change).
2. Add test harness + Mercury GR validation.
3. Introduce Rust `physics_core` behind FFI.
4. Introduce Rust `render_core` (wgpu) and integrate into Go GUI.
5. Add optional ray tracing kernels.
6. Add asset pipeline & asteroid belt.
7. Add spacetime fabric overlay.
8. Add Kennedy Launch Planner.
9. Build installers for macOS/Windows/Linux.

---

## Contributing

See `docs/CONTRIBUTING.md`.

---

## About

**Author:** Joshua Baney  
**Repository:** (link here)  
**Donations:** (link here)  

Credits and references are listed in `docs/CREDITS.md`.

---

## License

MIT (or update as desired).
