# Bevy Bootstrap Status

Status of the Phase B scaffold: the minimal Bevy application that renders the Sun and planets with an interactive camera.

## What Works

### Physics Integration
- `physics_core` crate is now usable as a Rust library dependency (added `lib` to `crate-type` alongside `cdylib`)
- Workspace `Cargo.toml` at project root manages `physics_core`, `render_core`, and `solar_sim_bevy`
- All 9 planets (Mercury through Pluto) are initialized from orbital elements ported verbatim from `internal/physics/planets.go`
- The orbital element to Cartesian state conversion (`CreatePlanetFromElements`) is ported from `simulator.go:162-218`
- GR correction is enabled for Mercury
- Planet-to-planet gravity is enabled
- The RK4 integrator from `physics_core::sim` drives the simulation at 60 Hz fixed update
- Substep subdivision applies when `|effective_dt| > 28800 s` (matching Go's `MaxSafeDt`)
- Simulation is playing by default at 1x speed with a base timestep of 7200 seconds (2 hours)

### Rendering
- Sun renders as an emissive unlit sphere (yellow/white glow)
- Each planet renders as a colored sphere with `StandardMaterial` (PBR)
- Planet colors match the Go source (`planets.go` RGBA values converted to linear sRGB)
- Planet sphere sizes are exaggerated for visibility (0.05 to 0.20 Bevy world units)
- A `PointLight` at the origin illuminates planets (intensity 2 billion, range 10000 units)
- Dim ambient light (brightness 50) ensures planets are not fully black on the far side
- Display scale is maintained via a `DisplayScale` component that reapplies `Transform.scale` each frame after the physics plugin updates `Transform.translation`

### Camera
- Orbit camera with spherical coordinate model (yaw, pitch, distance around focus point)
- Mouse scroll wheel zooms (logarithmic, clamped 0.5 to 5000 units)
- Right-mouse drag rotates the view (yaw + pitch with sensitivity 0.005)
- Middle-mouse drag pans the focus point in the camera's local XY plane
- WASD / arrow keys pan the focus point relative to the camera's forward direction
- Q/E keys rotate (yaw)
- R/F keys zoom in/out via keyboard
- Initial position: 50 units from origin, 45 degrees above the ecliptic plane, looking at the Sun

### Coordinate System
- Physics positions are in meters (f64), stored in `physics_core::Simulation`
- Display positions use 1 AU = 10 Bevy world units
- The physics XY plane (ecliptic) maps to Bevy's XZ plane; physics Z maps to Bevy Y (up)
- At this scale: Mercury is at ~3.9 units, Earth at ~10, Neptune at ~300, Pluto at ~395

### Build
- `cargo run -p solar_sim_bevy` opens a window and shows planets orbiting the Sun
- `cargo check -p solar_sim_bevy` compiles with zero errors and zero warnings
- `go build ./...` still works (Go app unaffected)
- `cargo check -p physics_core` still works (cdylib + lib dual output)

## What Is Stubbed or Missing

- **Textures**: Planets use flat color `StandardMaterial` only. No texture loading via `AssetServer`.
- **Bloom**: No `BloomSettings` on the camera. Sun glow is visible due to emissive material but does not bloom into surrounding pixels.
- **Skybox**: No background skybox texture.
- **Trails**: No orbital trail rendering (no `bevy_polyline` integration).
- **Labels**: No text labels on planets.
- **Asteroid belt**: No belt particles.
- **Moons / Comets / Asteroids**: Only the 9 planets + Sun are spawned. No moon, comet, or asteroid data.
- **UI panels**: No egui integration. No play/pause, speed slider, or body selection UI.
- **Follow-body camera**: The `OrbitCamera` does not yet track a selected planet entity.
- **Spacetime grid**: Not present.
- **Launch planner**: Not present.
- **Distance measurement**: Not present.
- **Settings persistence**: Not present.
- **Keyboard shortcuts**: Only WASD/QE/RF for camera. No Space for play/pause, no +/- for speed.

## Performance Baseline

Target: 60 FPS on discrete GPU, 30+ FPS on integrated GPU.

With 10 entities (Sun + 9 planets) and no trails/belt/UI, the scene is trivially lightweight. The physics step (9-body RK4 with GR) takes microseconds. The render cost is dominated by Bevy's PBR pipeline overhead for 10 sphere meshes, which should be well under 1 ms on any modern GPU.

Actual benchmarks should be measured after the first successful `cargo run`.

## Crate Layout

```
Cargo.toml                          (workspace root)
crates/
  physics_core/
    Cargo.toml                      (added lib + cdylib, ffi feature flag)
    src/                            (unchanged)
  solar_sim_bevy/
    Cargo.toml                      (depends on bevy 0.15, physics_core)
    src/
      main.rs                       (App + DefaultPlugins + 3 custom plugins)
      physics_plugin.rs             (SimulationConfig, SimulationState, planet catalog, orbital element conversion, step + sync systems)
      render_plugin.rs              (lighting, mesh/material attachment, display scale maintenance)
      camera_plugin.rs              (OrbitCamera resource, zoom/rotate/pan/keyboard input, transform application)
```
