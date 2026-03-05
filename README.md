<p align="center">
  <img src="media/Image.png" alt="Solar System Simulator" width="600"/>
</p>

<h1 align="center">Solar System Simulator</h1>

<p align="center">
  GPU-Accelerated, Physically Accurate &mdash; Go GUI + Rust Kernels
</p>

---

A cross-platform solar system and mission simulation toolkit with scientifically grounded N-body physics, real-time visualization, launch planning, and a modular architecture designed for GPU acceleration.

## Features

### Physics Engine
- **N-body gravity** with 8 planets + Sun, initialized from real Keplerian orbital elements
- **Two integrators**: Velocity Verlet (symplectic, default) and RK4 (4th-order Runge-Kutta)
- **Substep protection** — automatically subdivides large timesteps (max 28,800s) to prevent orbit collapse at high speeds
- **General Relativity** — 1PN post-Newtonian correction producing Mercury's ~43 arcsec/century perihelion precession
- **Catmull-Rom spline trails** — smooth orbital path rendering via 4-point interpolation

### GUI (Fyne)
- **Three-panel tabbed layout**: Simulation controls, Launch Planner, Bodies manager
- **Full menu bar**: File (screenshot export), View (trails/spacetime/labels toggles), Simulation (play/pause, integrator), Settings, About
- **Settings persistence** across sessions via Fyne Preferences API
- **Bodies panel** with live stats (distance, velocity, orbital period), per-body trail toggles, and follow buttons
- **Interactive controls**: variable time speed (1x–100,000x), zoom, pan, 3D rotation, follow mode
- **Spacetime fabric visualization** — weak-field GR curvature overlay (toggleable)
- **Distance measurement** — select two bodies to measure AU / km / light-minutes

### Launch Planner
- **Hohmann transfer** delta-v calculations with patched-conic trajectory modeling
- **Destinations**: LEO, ISS, GTO, Moon, Mars
- **Vehicles**: Generic, Falcon, Saturn V (with stage-by-stage delta-v budgets)
- **Trajectory propagation** with RK4 and CSV export

### CLI & Headless Mode
- Unified `solar-sim` binary with subcommands: `gui`, `run`, `validate`, `launch`, `assets`
- **Headless simulation** with CSV/JSON export (no graphics dependencies)
- **Physics validation** scenarios (energy, angular momentum, Kepler periods, Mercury precession)

### Rust Acceleration (Optional)
- **physics_core** — Rust N-body engine behind stable C ABI / FFI
- **render_core** — GPU rendering via wgpu (Metal/Vulkan/DX12)
- Conditional compilation via build tags (`rust_physics`, `rust_render`)

### Packaging & CI/CD
- **GitHub Actions** CI on macOS, Linux, Windows
- **Automated releases** — tag `v*` builds `.dmg`, `.tar.gz`, `.zip` for all platforms
- **Local packaging** via `make package-macos|linux|windows`

### Validation Suite
- 30+ unit tests + 4 benchmarks
- Energy conservation (drift < 1e-13 Newtonian)
- Angular momentum conservation (drift < 1e-15)
- Kepler period accuracy (Earth: 365.25d &plusmn;0.002%, Mercury: 87.97d &plusmn;0.04%)
- Mercury GR precession (42.97 arcsec/century, error 0.07%)
- Golden baseline reproducibility

---

## Quick Start

### Pre-built Binaries

Download from [GitHub Releases](https://github.com/joshbaney/solar-system-simulator/releases):

| Platform | File |
|----------|------|
| macOS (Apple Silicon) | `solar-sim-darwin-arm64.dmg` |
| macOS (Intel) | `solar-sim-darwin-amd64.dmg` |
| Linux (x86_64) | `solar-sim-linux-amd64.tar.gz` |
| Windows (x86_64) | `solar-sim-windows-amd64.zip` |

### Build from Source

```bash
# Prerequisites: Go 1.21+, C compiler (Xcode CLT on macOS, gcc on Linux)
git clone https://github.com/joshbaney/solar-system-simulator.git
cd solar-system-simulator

make build-solar-sim    # Build unified CLI with GUI
./bin/solar-sim gui     # Launch GUI
```

See [docs/INSTALL.md](docs/INSTALL.md) for platform-specific details, Rust backend setup, and headless builds.

---

## CLI

```bash
solar-sim gui                                          # Launch GUI
solar-sim run --years 1 --export ephemeris.csv         # Headless simulation -> CSV
solar-sim run --years 5 --format json --export out.json --integrator verlet
solar-sim validate --scenario all --years 10           # Physics validation
solar-sim launch --dest mars --vehicle falcon          # Launch planner
solar-sim launch --list-destinations                   # List available targets
solar-sim assets verify                                # Validate asset directory
```

See [docs/CLI.md](docs/CLI.md) for the full command reference.

---

## GUI

The GUI opens a 1280x800 window with three regions:

- **Left panel** (tabbed):
  - **Simulation** — time speed, play/pause, integrator selector, physics toggles
  - **Launch Planner** — destination/vehicle selection, delta-v results
  - **Bodies** — live stats for all 9 bodies, per-body trail toggles, follow buttons
- **Center** — 3D rendering canvas with orbital trails and spacetime overlay
- **Menu bar** — File, View, Simulation, Settings, About

Settings persist across sessions. See [docs/UI.md](docs/UI.md) for details.

---

## Physics

| Feature | Details |
|---------|---------|
| Gravity | Newtonian N-body + optional planet-planet interactions |
| GR | 1PN post-Newtonian for Mercury (~43 arcsec/century) |
| Integrators | Velocity Verlet (symplectic, default) and RK4 |
| Substeps | Auto-subdivide when dt > 28,800s |
| Precision | Energy drift < 1e-5/year (Verlet), angular momentum < 1e-15 |

See [docs/PHYSICS.md](docs/PHYSICS.md) and [docs/NUMERICS.md](docs/NUMERICS.md) for derivations and analysis.

---

## Project Structure

```
cmd/
  solar-sim/           Unified CLI (gui, run, validate, launch, assets)
  gui/                 Standalone GUI entry point
  meshgen/             Sphere mesh generator (.glb)
  validate-assets/     Asset directory validator
internal/
  physics/             N-body simulator, integrators (RK4, Verlet), GR
  render/              Fyne canvas renderer + GPU renderer (Rust)
  ui/                  Fyne GUI (app, menu, settings, about, bodies, launch)
  launch/              Hohmann transfers, vehicles, destinations, trajectory
  validation/          Physics validation scenarios
  math3d/              Vec3 + Catmull-Rom interpolation
  ffi/                 Go<->Rust FFI bindings
  spacetime/           GR curvature visualization
  viewport/            Camera (zoom, pan, rotation, follow)
  assets/              Asset validation
pkg/
  constants/           Physical constants (G, AU, c)
crates/
  physics_core/        Rust physics engine (cdylib, optional)
  render_core/         Rust GPU renderer via wgpu (cdylib, optional)
assets/
  textures/            Planet albedo maps (8K/4K/2K)
  models/              glTF models
  meshes/              Generated sphere meshes
packaging/             Platform packaging scripts
.github/workflows/     CI and release automation
docs/                  16 documentation files
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | High-level design and module diagram |
| [PHYSICS.md](docs/PHYSICS.md) | N-body model, GR corrections, validation |
| [NUMERICS.md](docs/NUMERICS.md) | Integrator comparison, timestep analysis, precision |
| [CLI.md](docs/CLI.md) | Command-line reference (all subcommands and flags) |
| [UI.md](docs/UI.md) | GUI layout, menu bar, panels, settings persistence |
| [LAUNCH_PLANNER.md](docs/LAUNCH_PLANNER.md) | Hohmann transfers, delta-v, vehicles, destinations |
| [FFI.md](docs/FFI.md) | Go/Rust FFI design (C ABI, opaque handles) |
| [RENDERING.md](docs/RENDERING.md) | Rendering architecture (Fyne + wgpu) |
| [RAY_TRACING.md](docs/RAY_TRACING.md) | Optional ray tracing mode |
| [INSTALL.md](docs/INSTALL.md) | Build from source, pre-built downloads, packaging |
| [TESTING.md](docs/TESTING.md) | Test suites, benchmarks, CI matrix |
| [ASSETS.md](docs/ASSETS.md) | Asset pipeline (textures, models, meshes) |
| [CODE_STYLE.md](docs/CODE_STYLE.md) | Go and Rust style conventions |
| [CONTRIBUTING.md](docs/CONTRIBUTING.md) | Dev workflow, build tags, PR process |
| [SECURITY.md](docs/SECURITY.md) | FFI safety, build integrity, asset handling |
| [CREDITS.md](docs/CREDITS.md) | Licenses and attribution |

---

## Build Targets

| Target | Description |
|--------|-------------|
| `make build` | Build GUI (Fyne) |
| `make build-solar-sim` | Build unified CLI with GUI |
| `make build-solar-sim-headless` | Build headless CLI (no graphics deps) |
| `make test` | Run all Go tests |
| `make bench` | Run physics benchmarks |
| `make lint` | Format check + vet + clippy |
| `make build-rust` | Build with Rust physics backend |
| `make build-gpu` | Build with Rust physics + GPU rendering |
| `make package-macos` | Create macOS `.dmg` |
| `make package-linux` | Create Linux `.tar.gz` |
| `make package-windows` | Create Windows `.zip` |

See [docs/INSTALL.md](docs/INSTALL.md) for full build instructions.

---

## Contributing

See [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) for prerequisites, workflow, build tags, testing guidelines, and the refactoring checklist.

## About

**Author:** Joshua Baney

Credits, references, and asset attribution are listed in [docs/CREDITS.md](docs/CREDITS.md).

## License

MIT
