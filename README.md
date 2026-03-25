<!-- Shields -->
<div align="center">

[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![AGPL License][license-shield]][license-url]

</div>

<!-- Project Logo & Title -->
<br />
<div align="center">
  <a href="https://github.com/JoshBaneyCS/Solar-System-Sim">
    <img src="media/Image.png" alt="Solar System Simulator" width="600">
  </a>

  <h1>Solar System Simulator</h1>
  <h3>v0.2.0</h3>

  <p>
    GPU-Accelerated, Physically Accurate N-Body Solar System Simulator
    <br />
    <strong>Rust + Bevy Engine</strong>
    <br />
    <br />
    <a href="https://github.com/JoshBaneyCS/Solar-System-Sim/releases">Download</a>
    &middot;
    <a href="https://github.com/JoshBaneyCS/Solar-System-Sim/issues/new?labels=bug">Report Bug</a>
    &middot;
    <a href="https://github.com/JoshBaneyCS/Solar-System-Sim/issues/new?labels=enhancement">Request Feature</a>
  </p>
</div>

---

<!-- Table of Contents -->
<details>
  <summary>Table of Contents</summary>
  <ol>
    <li><a href="#about-the-project">About The Project</a></li>
    <li><a href="#built-with">Built With</a></li>
    <li><a href="#features">Features</a></li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#pre-built-downloads">Pre-built Downloads</a></li>
        <li><a href="#build-from-source">Build from Source</a></li>
      </ul>
    </li>
    <li><a href="#usage">Usage</a></li>
    <li><a href="#keyboard-shortcuts">Keyboard Shortcuts</a></li>
    <li><a href="#cli-headless-mode">CLI / Headless Mode</a></li>
    <li><a href="#physics">Physics</a></li>
    <li><a href="#project-structure">Project Structure</a></li>
    <li><a href="#roadmap">Roadmap</a></li>
    <li><a href="#contributing">Contributing</a></li>
    <li><a href="#credits--acknowledgments">Credits & Acknowledgments</a></li>
    <li><a href="#license">License</a></li>
    <li><a href="#contact">Contact</a></li>
    <li><a href="#donate">Donate</a></li>
  </ol>
</details>

---

## About The Project

A cross-platform solar system simulator with scientifically grounded N-body physics, real-time 3D visualization, launch trajectory planning, and a modular architecture designed for GPU acceleration.

All planet textures, the milky way skybox, and mesh data are **embedded directly into the binary** — download, run, and explore the solar system with a single executable.

---

## Built With

| Technology | Role |
|-----------|------|
| [Rust](https://www.rust-lang.org/) | Core language |
| [Bevy Engine](https://bevyengine.org/) | 3D rendering, ECS, asset management |
| [wgpu](https://wgpu.rs/) | GPU abstraction (Vulkan, Metal, DX12, OpenGL) |
| [egui](https://www.egui.rs/) / bevy_egui | Immediate-mode UI panels |
| [Go](https://go.dev/) | Legacy CLI & headless simulation |

---

## Features

### Physics Engine
- **N-body gravity** with Sun + 8 planets + Pluto, initialized from real Keplerian orbital elements
- **Two integrators**: Velocity Verlet (symplectic, default) and RK4 (4th-order Runge-Kutta)
- **Substep protection** — automatically subdivides large timesteps to prevent orbit collapse at high speeds
- **General Relativity** — 1PN post-Newtonian correction producing Mercury's ~43 arcsec/century perihelion precession
- **Dynamic sun mass** — adjust the sun's mass in real-time and watch orbits respond

### 3D Visualization
- **8K planet textures** from NASA/Solar System Scope with proper albedo mapping
- **Milky Way skybox** — immersive starfield background
- **Bloom & HDR** — glowing sun with physically-based tone mapping
- **Saturn rings** — textured annular disc with alpha blending
- **Asteroid belt** — 1,500 Keplerian particles with Kirkwood gap modeling
- **Orbital trails** — color-coded per planet with alpha falloff
- **Body labels** — projected screen-space text labels
- **Spacetime grid** — gravitational potential visualization
- **Axial rotation** — all planets and moons spin at their real rotation rates with correct axial tilts

### Bodies
- **9 planets** (Mercury through Pluto) with full orbital elements
- **8 moons** (Moon, Io, Europa, Ganymede, Callisto, Titan, Phobos, Deimos)
- **4 comets** (Halley, Hale-Bopp, Encke, Swift-Tuttle)
- **6 asteroids** (Ceres, Vesta, Pallas, Hygiea, Apophis, Bennu)
- All body groups toggleable in real-time

### UI Panels
- **Simulation controls** — play/pause, speed (1x–100,000x), integrator select, physics toggles, sun mass slider, camera follow, zoom
- **Bodies panel** — grouped list with live distance (AU) and velocity (km/s), per-body follow buttons
- **Physics panel** — live equations, Earth orbital data, simulation statistics
- **Status bar** — FPS, simulation time, speed, zoom distance
- **About dialog** — version, system info, credits

### Launch Planner
- **Hohmann transfer** delta-v calculations with patched-conic trajectory modeling
- **5 destinations**: LEO, ISS, GTO, Moon, Mars
- **3 vehicles**: Generic, Falcon-like, Saturn V-like (with stage-by-stage delta-v budgets)
- **3D trajectory visualization** with color gradient and vehicle marker
- **Mission playback** — play/pause, speed control, timeline scrubber, live telemetry

---

## Getting Started

### Pre-built Downloads

Download from [GitHub Releases](https://github.com/JoshBaneyCS/Solar-System-Sim/releases):

| Platform | File |
|----------|------|
| macOS (Apple Silicon) | `solar-sim-darwin-arm64.dmg` |
| macOS (Intel) | `solar-sim-darwin-amd64.dmg` |
| Linux (x86_64) | `solar-sim-linux-amd64.tar.gz` |
| Windows (x86_64) | `solar-sim-windows-amd64.zip` |

All assets are embedded — no additional files needed. Just download and run.

### Build from Source

**Prerequisites:** Rust 1.75+ (install via [rustup](https://rustup.rs/))

```bash
git clone https://github.com/JoshBaneyCS/Solar-System-Sim.git
cd solar-system-simulator

# Run in development mode
cargo run -p solar_sim_bevy

# Build optimized release binary (assets are embedded at compile time)
cargo build --release -p solar_sim_bevy
```

**Platform-specific dependencies:**

<details>
<summary>macOS</summary>

No additional dependencies — Metal is used by default.

</details>

<details>
<summary>Linux</summary>

```bash
sudo apt-get install -y \
  pkg-config libasound2-dev libudev-dev libxkbcommon-dev \
  libwayland-dev libx11-dev libxi-dev libxcursor-dev \
  libxrandr-dev libxinerama-dev libgl1-mesa-dev
```

</details>

<details>
<summary>Windows</summary>

No additional dependencies — DirectX 12 is used by default. Visual Studio Build Tools required for compilation.

</details>

---

## Usage

Launch the simulator and you'll see the solar system rendered in 3D with textured planets orbiting the Sun.

**Camera controls:**
- **Right-click drag** — Rotate camera
- **Middle-click drag** — Pan
- **Scroll wheel** — Zoom in/out
- **WASD / Arrow keys** — Pan
- **Q/E** — Rotate
- **R/F** — Zoom in/out

**Left panel** — Simulation controls, display toggles, physics settings, camera follow, launch planner

**Right panels** — Bodies list, physics equations & live data

---

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Space` | Play / Pause |
| `+` / `-` | Increase / Decrease speed |
| `L` | Toggle labels |
| `T` | Toggle trails |
| `G` | Toggle spacetime grid |

---

## CLI / Headless Mode

The legacy Go CLI is available for headless simulation and data export:

```bash
# Build the headless binary (no graphics dependencies)
go build -tags nogui -o bin/solar-sim-headless ./cmd/solar-sim

# Run simulations
solar-sim-headless run --years 1 --export ephemeris.csv
solar-sim-headless run --years 5 --format json --export out.json
solar-sim-headless validate --scenario all --years 10
solar-sim-headless launch --dest mars --vehicle falcon
```

See [docs/CLI.md](docs/CLI.md) for the full command reference.

---

## Physics

| Feature | Details |
|---------|---------|
| Gravity | Newtonian N-body + optional planet-planet interactions |
| GR | 1PN post-Newtonian for Mercury (~43 arcsec/century) |
| Integrators | Velocity Verlet (symplectic, default) and RK4 |
| Substeps | Auto-subdivide when dt > 28,800s |
| Precision | Energy drift < 1e-5/year (Verlet), angular momentum < 1e-15 |
| Sun mass | Adjustable in real-time (0.1x – 5.0x) |

See [docs/PHYSICS.md](docs/PHYSICS.md) and [docs/NUMERICS.md](docs/NUMERICS.md) for derivations and analysis.

---

## Project Structure

```
crates/
  solar_sim_bevy/      Bevy 3D application (primary)
    src/
      main.rs          App entry point with embedded assets
      physics_plugin   N-body simulation, body catalog, dynamic spawning
      render_plugin    Textured meshes, Saturn rings, body spin
      camera_plugin    Orbit camera with egui input guard
      skybox_plugin    Milky Way skybox + bloom
      trail_plugin     Orbital trail ring buffers + gizmo rendering
      label_plugin     Screen-space body labels
      belt_plugin      1500 Keplerian asteroid belt particles
      spacetime_plugin Gravitational potential grid
      follow_plugin    Camera follow + auto-fit
      launch_plugin    3D trajectory visualization + mission playback
      launch_core      Hohmann transfers, vehicles, destinations
      ui_*             egui panels (controls, bodies, physics, about, status)
      body_catalog     Moon, comet, asteroid definitions
  physics_core/        Rust N-body engine (shared library)
  render_core/         Rust GPU renderer via wgpu (optional)
cmd/                   Go CLI entry points
internal/              Go physics, rendering, UI, launch planner
assets/                Planet textures (8K/4K/2K), skybox, meshes
packaging/             Platform packaging scripts (macOS, Linux, Windows)
.github/workflows/     CI and release automation
docs/                  Documentation
```

---

## Roadmap

- [x] N-body physics with GR corrections
- [x] Textured planets with 8K albedo maps
- [x] Milky Way skybox with bloom
- [x] Orbital trails and labels
- [x] Asteroid belt (1,500 particles)
- [x] Moons, comets, asteroids (dynamic toggling)
- [x] Full UI panels (controls, bodies, physics, status)
- [x] Launch planner with 3D trajectory visualization
- [x] Spacetime grid visualization
- [x] Body axial rotation with real periods
- [x] Embedded assets (single-binary distribution)
- [ ] Settings persistence (save/load preferences)
- [ ] Distance measurement tool (click two bodies)
- [ ] Screenshot export
- [ ] Comet tail rendering
- [ ] Ring system for Uranus and Neptune
- [ ] Gravitational wave visualization
- [ ] VR support

See the [open issues](https://github.com/JoshBaneyCS/Solar-System-Sim/issues) for a full list of proposed features and known issues.

---

## Contributing

Contributions are welcome! See [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) for prerequisites, workflow, and guidelines.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

## Credits & Acknowledgments

See [CREDITS.md](CREDITS.md) for full attribution including:

- **NASA / JPL** — Planetary ephemeris, orbital elements, physical constants
- **Solar System Scope** — Planet texture maps (CC BY 4.0)
- **Nvidia** — GPU computing and graphics technology
- **AMD** — Vulkan drivers and open-source GPU stack
- **Apple Metal** — macOS/iOS GPU framework
- **Bevy Engine** — Open-source Rust game engine
- **wgpu** — Cross-platform GPU abstraction

See [assets/CREDITS.md](assets/CREDITS.md) for detailed per-asset licensing.

---

## License

Distributed under the **GNU Affero General Public License v3.0**. See [LICENSE](LICENSE) for the full text.

---

## Contact

**Joshua Baney** — [GitHub](https://github.com/JoshBaneyCS/)

Project Link: [https://github.com/JoshBaneyCS/Solar-System-Sim](https://github.com/JoshBaneyCS/Solar-System-Sim)

---

## Donate

Help fund ongoing development:

**[Donate via PayPal](https://www.paypal.com/donate/?business=HWM2DENMWG4K2&no_recurring=0&item_name=TO+continue+funding+ongoing+development+to+Solar+System+Simulator+-+An+open+source+Physics+application&currency_code=USD)**

---

<!-- Reference-style links -->
[contributors-shield]: https://img.shields.io/github/contributors/JoshBaneyCS/Solar-System-Sim.svg?style=for-the-badge
[contributors-url]: https://github.com/JoshBaneyCS/Solar-System-Sim/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/JoshBaneyCS/Solar-System-Sim.svg?style=for-the-badge
[forks-url]: https://github.com/JoshBaneyCS/Solar-System-Sim/network/members
[stars-shield]: https://img.shields.io/github/stars/JoshBaneyCS/Solar-System-Sim.svg?style=for-the-badge
[stars-url]: https://github.com/JoshBaneyCS/Solar-System-Sim/stargazers
[issues-shield]: https://img.shields.io/github/issues/JoshBaneyCS/Solar-System-Sim.svg?style=for-the-badge
[issues-url]: https://github.com/JoshBaneyCS/Solar-System-Sim/issues
[license-shield]: https://img.shields.io/github/license/JoshBaneyCS/Solar-System-Sim.svg?style=for-the-badge
[license-url]: https://github.com/JoshBaneyCS/Solar-System-Sim/blob/master/LICENSE



Made by Josh Baney - For the love of science! 
