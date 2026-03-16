# Open Issues: Bevy Bootstrap

Known gaps, technical issues, and blockers discovered during the Phase B scaffold.

## Gaps vs Go Feature Set

### Visual Features Not Yet Present
| Feature | Go Location | Priority | Notes |
|---------|------------|----------|-------|
| Planet textures | `internal/render/textures.go` | Phase B | Need `AssetServer` texture loading, UV-mapped spheres |
| Sun bloom | `internal/render/renderer.go` | Phase B | Add `BloomSettings` to camera entity |
| Skybox | `internal/render/renderer.go` | Phase B | Load milky way cubemap |
| Orbital trails | `internal/render/renderer.go` | Phase C | Requires `bevy_polyline` or Gizmos |
| Planet labels | `internal/render/renderer.go` | Phase C | `Text2d` billboard children |
| Asteroid belt | `internal/render/belt.go` | Phase C | 1500 instanced particles, Kepler solver |
| Moons (8) | `internal/physics/moons.go` | Phase C | Need `CreateMoonFromElements` port |
| Comets (4) | `internal/physics/comets.go` | Phase C | Need comet tail rendering |
| Asteroids (6) | `internal/physics/asteroids.go` | Phase C | Named asteroids with procedural meshes |
| Spacetime grid | `internal/render/spacetime.go` | Phase C | Dynamic mesh with potential field warping |
| Distance measurement | `internal/ui/app.go` | Phase C | Click-select two bodies, draw line |
| Comet tails | `internal/render/renderer.go` | Phase C | Billboard quad away from Sun |

### UI Features Not Yet Present
| Feature | Go Location | Priority | Notes |
|---------|------------|----------|-------|
| Play/pause | `internal/ui/app.go` | Phase B | Needs egui or native Bevy UI |
| Speed slider | `internal/ui/app.go` | Phase B | 2^(-10) to 2^10 range |
| Follow-body dropdown | `internal/ui/app.go` | Phase B | Camera tracks selected planet |
| Zoom display | `internal/ui/statusbar.go` | Phase B | Show zoom level in status bar |
| Bodies panel | `internal/ui/app.go` | Phase C | Grouped list with distance/velocity |
| Physics panel | `internal/ui/app.go` | Phase C | Live equations, orbital parameters |
| Launch planner UI | `internal/ui/launch_panel.go` | Phase D | Vehicle/destination selection |
| Mission playback | `internal/ui/mission_playback.go` | Phase D | Timeline scrubbing |
| Settings dialog | `internal/ui/app.go` | Phase D | Persist to disk via ron |
| About dialog | `internal/ui/about.go` | Phase D | Version, credits |
| Keyboard shortcuts | `internal/ui/app.go` | Phase B | Space, +/-, Escape, F11 |
| Status bar | `internal/ui/statusbar.go` | Phase B | FPS, sim time, speed |
| Diagnostics | `internal/ui/diagnostics.go` | Phase D | OS, GPU, version info |

## Technical Issues Discovered

### 1. Coordinate System Mapping
The physics engine uses a right-handed coordinate system where XY is the ecliptic plane and Z is the normal. Bevy uses Y-up. The current mapping is:
- Physics X -> Bevy X
- Physics Y -> Bevy Z
- Physics Z -> Bevy Y

This works but means orbital inclinations appear rotated compared to the Go renderer (which uses Z-up in its 3D mode). Verify visual correctness once the app runs.

### 2. Display Scale Tuning
The current scale of 1 AU = 10 Bevy world units puts Neptune at ~300 units and Pluto at ~395 units. The camera starts at 50 units from origin, which shows the inner solar system well but requires significant zoom-out to see outer planets. May need:
- A wider default zoom, or
- An auto-fit camera system that frames all planets on startup

### 3. Planet Sphere Sizes
Planet display radii are set as fixed world-unit sizes (0.05 to 0.20 units). At the inner solar system zoom level these are visible, but when zoomed to see Pluto, inner planet spheres become subpixel. The Go renderer uses zoom-dependent scaling. The Bevy app needs a `update_display_radius` system that scales sphere size based on camera distance (Phase B followup).

### 4. `physics_core` Dual crate-type Build
Adding `lib` alongside `cdylib` in `crate-type` means every `cargo build` produces both a `.dylib`/`.so` and an `.rlib`. This doubles the physics_core build output but does not affect correctness. If build times become a concern, the `cdylib` could be moved behind the `ffi` feature flag. For now, the dual output is acceptable.

### 5. No Verlet Integrator in physics_core
The existing `physics_core` only implements RK4. The Go app supports both RK4 and Verlet. The Bevy app defaults to RK4 which is fine for the bootstrap, but Phase A should add Verlet before Phase C (when the UI needs to offer integrator selection).

### 6. No `current_time` Tracking in physics_core
The `Simulation` struct does not track elapsed simulation time. The Bevy app would need to track this separately in a `SimulationClock` resource. This is acceptable for bootstrap but should be added to `Simulation` during Phase A.

### 7. Body Metadata Not in physics_core
The physics engine only stores mass, position, velocity, and GR flags per body. Body names, types, colors, and radii are stored only in the Bevy plugin's `CelestialBody` component and `PlanetDef` static data. For Phase A, a `BodyInfo` struct should be added to `physics_core` so the catalog is the single source of truth.

## What Blocks Phase C

Phase C (visual feature parity) requires:

1. **Phase A completion**: Body catalog, Verlet integrator, trail management, and belt particle state must exist in `solar_sim_core` (the renamed `physics_core`).
2. **`bevy_polyline` compatibility**: Trail rendering depends on `bevy_polyline 0.10` working with Bevy 0.15. If incompatible, fallback to Bevy `Gizmos` API (lower visual quality).
3. **`bevy_egui` compatibility**: All UI panels depend on `bevy_egui 0.34` working with Bevy 0.15. If incompatible, fallback to Bevy's native `bevy_ui` (more development effort).
4. **Texture assets**: Planet texture files must exist in `assets/textures/<planet>/albedo.jpg`. The Go app's existing textures (if any) need to be copied or new ones sourced.
5. **Skybox asset**: A milky way cubemap or equirectangular texture is needed for the background.

None of these are hard blockers -- each has a fallback strategy documented in the phased migration plan.
