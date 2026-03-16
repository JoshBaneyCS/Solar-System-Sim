# Phased Migration Plan

Four phases, each producing a runnable binary. The Go app remains functional and unmodified throughout all phases.

---

## Phase A: Physics Crate Extraction

**Goal:** Transform the existing `physics_core` crate into a complete, validated physics library (`solar_sim_core`) that can serve both the Go FFI path and the future Bevy app.

### Deliverables

1. **Rename `physics_core` to `solar_sim_core`**
   - Update Cargo.toml: `name = "solar_sim_core"`
   - Change crate-type from `["cdylib"]` to `["lib"]`
   - Add `[features] ffi = []` with conditional cdylib: when `--features ffi` is passed, also build cdylib
   - Update Go FFI linker paths to reference new crate name

2. **Create workspace Cargo.toml**
   ```toml
   [workspace]
   resolver = "2"
   members = ["crates/solar_sim_core"]
   ```

3. **Add body catalog module** (`bodies/catalog.rs`)
   - Port all orbital elements from Go:
     - `PLANET_DATA`: 9 entries from `physics/planets.go`
     - `MOON_DATA`: 8 entries from `physics/moons.go`
     - `COMET_DATA`: 4 entries from `physics/comets.go`
     - `ASTEROID_DATA`: 6 entries from `physics/asteroids.go`
   - Port `BodyType` enum: Star, Planet, DwarfPlanet, Moon, Comet, Asteroid
   - Port `Planet` struct (orbital elements)
   - Port `Body` struct with all fields

4. **Add orbital element conversion** (`bodies/builder.rs`)
   - Port `CreatePlanetFromElements` from `simulator.go:162-218`
   - Port `CreateMoonFromElements` from `simulator.go:411-465`
   - Port `AddMoons`, `AddComets`, `AddAsteroids`, `RemoveBodiesByType`

5. **Add Velocity Verlet integrator** (`integrators/verlet.rs`)
   - Port `stepVerlet` from `verlet.go`
   - Existing RK4 extracted to `integrators/rk4.rs`
   - `Simulation` gains `integrator: IntegratorType` field and dispatches accordingly

6. **Add trail management** (`trail.rs`)
   - `TrailManager` with `VecDeque<Vec3>` per body, configurable max length
   - Append on step, truncate when full
   - Port from `simulator.go` trail logic in `Step()` and `stepVerlet()`

7. **Add substep logic to `Simulation`**
   - Port substep subdivision from `simulator.go:389-406`
   - `MAX_SAFE_DT = 28800.0` (8 hours)
   - Public `update(dt, time_speed)` method that handles subdivision

8. **Add belt particle state** (`belt.rs`)
   - `BeltParticleSet`: 1500 particles with randomized Keplerian elements
   - Kirkwood gap exclusion at 2.5, 2.82, 2.95 AU
   - `compute_position(sim_time) -> Vec3` per particle (Kepler equation solver)
   - Port from `render/belt.go:25-108`

9. **Add launch planner** (`launch/`)
   - Port all of `internal/launch/`: orbital mechanics, planner, vehicle/destination presets, propagator, rocket equation, CSV export
   - This is ~930 LOC of Go, mostly pure math

10. **Add spacetime computation** (`spacetime.rs`)
    - Port `h_00 = 2GM/(c^2 * r)` potential field computation from `spacetime/spacetime.go`
    - Returns grid data (positions + potential values), not visual objects
    - Adaptive resolution logic

11. **Add validation harness** (`validation/`)
    - Port all 5 scenarios from `internal/validation/`:
      - Energy conservation
      - Angular momentum conservation
      - Earth Kepler period
      - Mercury Kepler period
      - Mercury precession (Laplace-Runge-Lenz vector)
    - Tests assert same tolerances as Go:
      - Energy drift < `1e-4 * years`
      - Mercury precession: 42-44 arcsec/century
      - Kepler period within 0.5% of expected

12. **Update FFI** (`ffi.rs`)
    - Wrap behind `#[cfg(feature = "ffi")]`
    - Add new FFI functions for Verlet integrator selection, body add/remove
    - Update Go side to call new crate name
    - Ensure Go `rust_physics` build tag still works

### Success Criteria

- `cargo test -p solar_sim_core` passes all validation tests
- `cargo test -p solar_sim_core -- validation::energy` drift < 1e-4 per simulated year
- `cargo test -p solar_sim_core -- validation::precession` reports 42-44 arcsec/century
- `cargo build --features ffi -p solar_sim_core` produces cdylib
- `go build -tags rust_physics ./cmd/gui` still works with the renamed crate
- `go run ./cmd/cli validate` still passes (Go validation unaffected)

### Estimated Effort

| Task | LOC (approx) | Time |
|------|-------------|------|
| Rename + workspace setup | 50 | 2 hours |
| Body catalog + builder | 400 | 1 day |
| Verlet integrator | 100 | 4 hours |
| Trail management | 80 | 2 hours |
| Substep logic | 40 | 1 hour |
| Belt particles | 120 | 4 hours |
| Launch planner | 500 | 2 days |
| Spacetime computation | 100 | 4 hours |
| Validation harness | 300 | 1.5 days |
| FFI updates | 60 | 3 hours |
| **Total** | **~1,750** | **~7-8 days** |

### Rollback Strategy

If this phase encounters blockers:
- The original `physics_core` crate is untouched until the rename is confirmed working
- Go app never depends on new features; it only cares that the FFI surface is unchanged
- Can revert to the original crate name by restoring `Cargo.toml` and Go linker paths

---

## Phase B: Bevy Window + Planets + Camera

**Goal:** A minimal but functional Bevy application that renders the Sun and 9 planets as textured spheres with physically accurate orbits and an interactive camera.

### Prerequisites

- Phase A complete (solar_sim_core with body catalog and validation)

### Deliverables

1. **Create `solar_sim_bevy` crate**
   - Add to workspace Cargo.toml
   - Dependencies: `solar_sim_core`, `bevy 0.15`, `bevy_egui`

2. **PhysicsPlugin (minimal)**
   - `spawn_solar_system`: create Sun + 9 planets as entities
   - `step_simulation`: runs in `FixedUpdate` at 60Hz, calls `solar_sim_core::Simulation::update()`
   - `sync_ecs_from_simulation`: copies f64 positions to f32 `Transform`
   - `SimulationConfig` resource with play/pause and time speed

3. **CelestialRenderPlugin (minimal)**
   - `setup_lighting`: `PointLight` at Sun position
   - Sun entity with emissive material + `BloomSettings` on camera
   - Planet entities with `StandardMaterial` + albedo textures loaded via `AssetServer`
   - `update_display_radius`: scale sphere meshes based on `PhysicalRadius` and camera distance
   - `setup_skybox`: load milky way texture

4. **CameraPlugin**
   - `OrbitCamera` resource
   - Mouse wheel zoom, drag to rotate, Shift+drag to pan
   - WASD/QE/RF keyboard controls
   - Follow-body support
   - `spawn_camera`: initial position looking down at ecliptic

5. **UIPlugin (minimal)**
   - egui side panel: Play/Pause button, speed slider, follow-body dropdown, zoom slider
   - egui status bar: FPS, sim time, speed

6. **Asset pipeline**
   - Copy existing `assets/textures/` directory structure
   - Bevy's `AssetServer` loads textures by convention: `textures/<planet>/albedo.jpg`

### Success Criteria

- `cargo run -p solar_sim_bevy` opens a window with the Sun and 9 planets
- Planets orbit at correct periods (visual verification: Earth completes orbit in ~365 sim-days)
- Textures render correctly on planet spheres
- Camera zoom, pan, rotate work with mouse and keyboard
- Follow-body tracks selected planet smoothly
- Sun has visible glow (bloom)
- Skybox visible when camera rotates
- FPS >= 60 on discrete GPU, >= 30 on integrated GPU
- Play/pause and speed controls work

### Estimated Effort

| Task | LOC (approx) | Time |
|------|-------------|------|
| Crate setup + main.rs | 80 | 2 hours |
| PhysicsPlugin | 200 | 1 day |
| CelestialRenderPlugin | 250 | 1.5 days |
| CameraPlugin | 300 | 1.5 days |
| UIPlugin (minimal) | 150 | 1 day |
| Asset pipeline | 30 | 2 hours |
| Integration testing | - | 1 day |
| **Total** | **~1,010** | **~7-8 days** |

### Rollback Strategy

- The Go app is unaffected by this phase
- If Bevy proves unsuitable (performance, API instability), the `solar_sim_core` crate from Phase A is still useful for the Go FFI path
- The Phase B crate can be deleted without affecting anything else

---

## Phase C: Feature Parity -- Visual

**Goal:** The Bevy app matches the Go CPU renderer's visual feature set. All rendering features present; UI panels functional.

### Prerequisites

- Phase B complete (planets orbiting with camera)

### Deliverables

1. **TrailPlugin**
   - Integrate `bevy_polyline`
   - Trail data from `solar_sim_core::TrailManager` -> polyline vertices
   - Catmull-Rom interpolation for smooth curves
   - Per-vertex alpha fade (old = transparent, new = opaque)
   - Per-body trail color matching body color
   - Downsample to max 200 visual segments

2. **BeltPlugin**
   - Spawn 1500 instanced entities with `BeltParticle` component
   - Each frame: compute Keplerian positions via `solar_sim_core::belt`
   - Small gray sphere meshes (1-3 pixel equivalent at default zoom)
   - Kirkwood gaps visible in belt structure

3. **Moons, Comets, Asteroids**
   - Extend `spawn_solar_system` to support moons (8), comets (4), asteroids (6)
   - Dynamic add/remove via `SimCommand::SetShowMoons(bool)` etc.
   - Comet tail rendering: billboard quad or particle emitter, direction away from Sun, length inversely proportional to Sun distance

4. **Labels**
   - `Text2d` billboard labels for each body
   - Positioned below the body sphere
   - Toggle via `SimulationConfig.show_labels`

5. **Distance measurement**
   - Click to select bodies (up to 2)
   - Draw line between selected bodies using `Gizmos`
   - Display distance in AU, km, and light-minutes

6. **SpacetimePlugin**
   - Dynamic mesh from `solar_sim_core::spacetime::compute_potential_field()`
   - Vertex positions warped by gravitational potential
   - Vertex color gradient: purple -> red -> orange
   - Adaptive grid resolution
   - Cache invalidation when camera moves > 5%

7. **UIPlugin (full)**
   - Controls panel: all toggles from Go (trails, spacetime, labels, belt, moons, comets, asteroids, planet gravity, relativity, integrator, sun mass)
   - Bodies panel: grouped list, per-body distance/velocity/period, follow/trail toggles
   - Physics panel: live equations display, Earth orbital parameters
   - Settings: persist to disk via `ron` serialization
   - About dialog: version, credits
   - Keyboard shortcuts: Space, +/-, Escape, F11

8. **Launch trajectory overlay**
   - Polyline with green-to-red color gradient
   - Offset by Earth's current position for Earth-centered trajectories

### Success Criteria

- Side-by-side visual comparison with Go CPU renderer shows equivalent features
- All body types render correctly (planets textured, moons small, comets with tails, asteroids irregular shapes or small spheres)
- Trails draw smooth orbital paths with proper fade
- Asteroid belt shows Kirkwood gap structure
- Spacetime grid deforms correctly around massive bodies
- All UI panels functional and responsive
- Settings persist across app restarts
- FPS >= 60 with all features enabled (on discrete GPU)
- FPS >= 30 with all features enabled (on integrated GPU)

### Estimated Effort

| Task | LOC (approx) | Time |
|------|-------------|------|
| TrailPlugin | 250 | 1.5 days |
| BeltPlugin | 200 | 1 day |
| Moons/Comets/Asteroids | 300 | 1.5 days |
| Labels | 100 | 4 hours |
| Distance measurement | 80 | 3 hours |
| SpacetimePlugin | 300 | 2 days |
| UIPlugin (full) | 800 | 3 days |
| Launch trajectory overlay | 100 | 4 hours |
| Integration + polish | - | 2 days |
| **Total** | **~2,130** | **~13-14 days** |

### Rollback Strategy

- Each feature is an independent plugin; broken features can be disabled by removing the plugin registration
- The Go app remains the fallback for any feature that proves too difficult in Bevy
- Trail/belt/spacetime plugins can be simplified (e.g., Gizmos instead of polylines) if third-party crate issues arise

---

## Phase D: Feature Parity -- Full

**Goal:** Complete feature parity with the Go app. The Bevy app passes all validation tests and supports all features. The Go app can be deprecated.

### Prerequisites

- Phase C complete (visual feature parity)

### Deliverables

1. **LaunchPlugin (full)**
   - Complete launch planner UI: vehicle selection, destination selection, simulate button
   - Results display: delta-v budget breakdown, feasibility status, transfer time
   - Mission playback: play/pause, speed control (1x-64x), timeline scrubbing
   - Telemetry display during playback: altitude, velocity, acceleration, distance
   - Vehicle marker (green sphere) at interpolated trajectory position

2. **Diagnostics panel**
   - OS, architecture, CPU count, GPU info (from Bevy's `RenderDevice`)
   - Rust version, Bevy version

3. **CLI subcommands**
   - `solar-sim-bevy validate` -- runs `solar_sim_core::validation::run_all()`
   - `solar-sim-bevy run` -- headless simulation with CSV/JSON export
   - `solar-sim-bevy launch` -- launch planning with CSV export
   - These use `solar_sim_core` directly, no Bevy needed (separate binary or feature-gated)

4. **Settings persistence**
   - Serialize `SimulationConfig` to RON file on exit
   - Load on startup
   - Settings dialog to modify all options

5. **Screenshot export**
   - Bevy screenshot capture to PNG file
   - File dialog for save location (via `rfd` crate)

6. **Procedural asteroid meshes**
   - Port the 8-lobe radial perturbation from `textures.go:217-293` to generate procedural `Mesh`
   - Deterministic RNG per asteroid name

7. **Validation gate**
   - Run full Rust validation suite:
     - Energy conservation (1-year, 10-year)
     - Angular momentum conservation
     - Earth Kepler period
     - Mercury Kepler period
     - Mercury precession: 42-44 arcsec/century
   - Results must match Go tolerances exactly

8. **Optional: Custom ray tracing render node**
   - Only if PBR pipeline proves insufficient for desired visual quality
   - Port WGSL compute shader from `render_core/raytracer.rs` to a Bevy custom render plugin
   - Estimated 2-3 weeks of additional work
   - Recommendation: skip this unless there is a specific visual goal that PBR cannot achieve

### Success Criteria

- All validation tests pass in Rust with same tolerances as Go
- All features from the Go feature inventory document are present
- Launch planner produces identical delta-v budgets as Go (within floating-point tolerance)
- Mission playback works with all destinations
- Settings persist and restore correctly
- CLI subcommands produce correct output
- Screenshot export works
- No visual regressions compared to Go app (screenshot comparison)
- FPS >= 60 on discrete GPU with all features
- Binary size reasonable (< 100 MB with embedded assets)
- Builds on macOS, Linux, and Windows

### Go App Deprecation Checklist

Before deleting any Go code:

- [ ] Rust validation: energy drift < 1e-4 per year (Verlet, 10-year simulation)
- [ ] Rust validation: Mercury precession 42-44 arcsec/century
- [ ] Rust validation: Earth Kepler period within 0.5%
- [ ] Rust validation: Mercury Kepler period within 0.5%
- [ ] Rust validation: angular momentum conserved < 1e-10 relative drift
- [ ] Visual: all 9 planets render with correct textures
- [ ] Visual: 8 moons orbit correct parent planets
- [ ] Visual: 4 comets with tails
- [ ] Visual: 6 named asteroids
- [ ] Visual: 1500 belt particles with Kirkwood gaps
- [ ] Visual: orbital trails with Catmull-Rom smoothing
- [ ] Visual: spacetime curvature grid
- [ ] Visual: skybox
- [ ] Visual: sun glow (bloom)
- [ ] Visual: labels
- [ ] Visual: distance measurement line
- [ ] UI: all control panel toggles functional
- [ ] UI: bodies panel with live data
- [ ] UI: physics panel with equations
- [ ] UI: launch planner with all destinations
- [ ] UI: mission playback with telemetry
- [ ] UI: status bar with FPS/sim time/speed
- [ ] UI: settings persistence
- [ ] UI: about dialog
- [ ] UI: keyboard shortcuts
- [ ] Camera: zoom, pan, rotate, follow, auto-fit
- [ ] CLI: validate, run, launch subcommands
- [ ] Performance: >= 60 FPS on discrete GPU
- [ ] Build: macOS, Linux, Windows

### Estimated Effort

| Task | LOC (approx) | Time |
|------|-------------|------|
| LaunchPlugin (full) | 500 | 2.5 days |
| Diagnostics | 50 | 2 hours |
| CLI subcommands | 300 | 1.5 days |
| Settings persistence | 80 | 3 hours |
| Screenshot export | 40 | 2 hours |
| Procedural asteroids | 150 | 1 day |
| Validation gate | 100 | 1 day |
| Cross-platform testing | - | 2 days |
| Polish + bug fixes | - | 3 days |
| **Total** | **~1,220** | **~12-13 days** |

---

## Timeline Summary

| Phase | Duration | Cumulative | Runnable Binary |
|-------|----------|------------|-----------------|
| A: Physics Crate Extraction | ~8 days | 8 days | Go app (unchanged) + `cargo test` passes validation |
| B: Bevy Window + Planets + Camera | ~8 days | 16 days | `cargo run -p solar_sim_bevy` shows planets orbiting |
| C: Feature Parity -- Visual | ~14 days | 30 days | Bevy app visually matches Go app |
| D: Feature Parity -- Full | ~13 days | 43 days | Bevy app fully replaces Go app |

**Total estimated effort: ~6 weeks** for a single developer working full-time.

Phases can overlap: Phase B can start before Phase A's launch planner port is complete (it only needs the body catalog and integrators). Phase C can start before Phase B's UI is polished.

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Bevy 0.15 breaking API changes during development | Medium | High | Pin exact Bevy version in Cargo.toml. Do not upgrade mid-phase. |
| `bevy_polyline` incompatible with Bevy 0.15 | Low | Medium | Fallback: Bevy `Gizmos` API for trails (lower visual quality but functional). |
| `bevy_egui` incompatible with Bevy 0.15 | Low | High | Fallback: Bevy's native `bevy_ui` (more work but no external dependency). |
| f32 precision causes visual artifacts at Neptune distance | Low | Low | Already mitigated by AU-based scaling (Neptune = ~3000 Bevy units). |
| Spacetime grid performance too slow | Medium | Low | Reduce grid resolution, compute on background thread via `AsyncComputeTaskPool`. |
| Go FFI breaks during crate rename | Low | Medium | Test Go build immediately after rename. Keep old crate as git history fallback. |
| Validation tests fail to match Go tolerances | Low | High | Debug with identical initial conditions. Diff stepping algorithm line-by-line. Both use IEEE 754 f64. |
| Belt particle rendering too slow (1500 entities) | Low | Low | Bevy auto-instances identical meshes. If still slow, switch to GPU compute + point sprites. |
| Texture loading fails on some platforms | Low | Medium | Test on macOS, Linux, Windows early in Phase B. Use PNG fallback if JPEG decode fails. |
