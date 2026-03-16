# Target Architecture

## Rust Workspace Structure

```
solar-system-simulator/
├── Cargo.toml                      # Workspace root
├── crates/
│   ├── solar_sim_core/             # Pure physics library (no Bevy dependency)
│   │   ├── Cargo.toml
│   │   └── src/
│   │       ├── lib.rs
│   │       ├── vec3.rs             # Kept from physics_core
│   │       ├── constants.rs        # Kept from physics_core
│   │       ├── gr.rs               # Kept from physics_core
│   │       ├── sim.rs              # Extended from physics_core
│   │       ├── integrators/
│   │       │   ├── mod.rs
│   │       │   ├── rk4.rs          # RK4 integrator (extracted from sim.rs)
│   │       │   └── verlet.rs       # Velocity Verlet (ported from Go)
│   │       ├── bodies/
│   │       │   ├── mod.rs
│   │       │   ├── catalog.rs      # PlanetData, MoonData, CometData, AsteroidData
│   │       │   ├── body.rs         # Body, BodyType, OrbitalElements
│   │       │   └── builder.rs      # CreatePlanetFromElements, CreateMoonFromElements
│   │       ├── trail.rs            # Trail ring buffer management
│   │       ├── belt.rs             # Asteroid belt Keplerian particle state
│   │       ├── launch/
│   │       │   ├── mod.rs
│   │       │   ├── orbital.rs      # Hohmann, vis-viva, plane change, hyperbolic excess
│   │       │   ├── planner.rs      # LaunchPlan, DeltaVBudget
│   │       │   ├── vehicle.rs      # Vehicle presets
│   │       │   ├── destination.rs  # Destination presets
│   │       │   ├── propagator.rs   # RK4 2-body trajectory propagation
│   │       │   └── rocket.rs       # Tsiolkovsky equation
│   │       ├── spacetime.rs        # Gravitational potential field computation
│   │       ├── validation/
│   │       │   ├── mod.rs
│   │       │   ├── energy.rs       # Energy conservation test
│   │       │   ├── angular_momentum.rs
│   │       │   ├── kepler.rs       # Orbital period measurement
│   │       │   └── precession.rs   # Mercury precession (Laplace-Runge-Lenz)
│   │       └── ffi.rs              # C FFI (behind feature flag)
│   │
│   └── solar_sim_bevy/             # Bevy application
│       ├── Cargo.toml
│       └── src/
│           ├── main.rs
│           ├── plugins/
│           │   ├── mod.rs
│           │   ├── physics.rs      # PhysicsPlugin
│           │   ├── celestial.rs    # CelestialRenderPlugin
│           │   ├── trail.rs        # TrailPlugin
│           │   ├── belt.rs         # BeltPlugin
│           │   ├── camera.rs       # CameraPlugin
│           │   ├── ui.rs           # UIPlugin
│           │   ├── spacetime.rs    # SpacetimePlugin
│           │   └── launch.rs       # LaunchPlugin
│           ├── components.rs       # All ECS components
│           ├── resources.rs        # All ECS resources
│           ├── events.rs           # All ECS events
│           └── bundles.rs          # Entity bundles
│
├── assets/                         # Shared assets (textures, skybox, meshes)
│   └── textures/
│       ├── earth/albedo.jpg
│       ├── skybox/milky_way.jpg
│       └── ...
│
├── internal/                       # Go source (stays functional during migration)
├── cmd/                            # Go entrypoints (stays functional during migration)
└── go.mod
```

## Workspace Cargo.toml

```toml
[workspace]
resolver = "2"
members = [
    "crates/solar_sim_core",
    "crates/solar_sim_bevy",
]

[workspace.dependencies]
bevy = "0.15"
bevy_egui = "0.34"
bevy_polyline = "0.10"
rayon = "1.10"
```

## Crate: solar_sim_core

**Purpose:** Pure physics library with zero graphics dependencies. This is the single source of truth for all simulation math. Both the Bevy app and (during migration) the Go app via FFI consume it.

**Cargo.toml:**
```toml
[package]
name = "solar_sim_core"
version = "0.2.0"
edition = "2021"

[features]
default = []
ffi = []  # Enables C FFI exports for Go interop

[dependencies]
rayon = { workspace = true }

[lib]
# Library by default; cdylib only when FFI feature is active
crate-type = ["lib"]

# When building for Go FFI, add cdylib:
# cargo build --features ffi -p solar_sim_core
```

**What it contains:**
- Vec3 math (from existing `physics_core/vec3.rs`)
- Physical constants G, C, AU (from existing `physics_core/constants.rs`)
- GR correction (from existing `physics_core/gr.rs`)
- N-body simulation engine with RK4 and Verlet integrators
- Complete body catalog: all 9 planets, 8 moons, 4 comets, 6 asteroids with orbital elements
- Orbital element to Cartesian state conversion (port of Go `CreatePlanetFromElements`)
- Trail ring buffer management (VecDeque with max length)
- Asteroid belt particle state and Kepler solver
- Launch planner (Hohmann, Tsiolkovsky, trajectory propagation)
- Spacetime potential field computation
- Validation harness (energy, angular momentum, Kepler, precession)
- C FFI surface behind `ffi` feature flag (for continued Go interop)

**What it does NOT contain:**
- Any Bevy types or dependencies
- Any rendering logic
- Any GUI types

**Key design principle:** The `Simulation` struct owns all physics state. Bevy systems call methods on it, but the struct itself has no knowledge of ECS. This keeps simulation logic testable in isolation and allows the validation suite to run without Bevy.

## Crate: solar_sim_bevy

**Purpose:** Bevy application that renders the simulation using `solar_sim_core` as a library.

**Cargo.toml:**
```toml
[package]
name = "solar_sim_bevy"
version = "0.1.0"
edition = "2021"

[dependencies]
solar_sim_core = { path = "../solar_sim_core" }
bevy = { workspace = true }
bevy_egui = { workspace = true }
bevy_polyline = { workspace = true }
```

**What it contains:**
- Bevy `App` with plugin-based architecture
- ECS components, resources, events, bundles
- All rendering (via Bevy's PBR pipeline + plugins)
- UI panels via `bevy_egui`
- Camera controller
- Input handling

## Dependency Graph

```
solar_sim_bevy
    ├── solar_sim_core   (physics, bodies, launch, validation)
    ├── bevy 0.15        (rendering, ECS, windowing, input)
    ├── bevy_egui        (UI panels)
    └── bevy_polyline    (orbital trails, trajectory overlay)

solar_sim_core
    └── rayon            (parallel acceleration computation)

Go app (during migration)
    └── solar_sim_core   (via FFI feature flag, cdylib)
```

## What Goes in Bevy ECS vs External Modules

### IN Bevy ECS (solar_sim_bevy)

| Concern | Mechanism | Rationale |
|---------|-----------|-----------|
| Per-body rendering state | Components (`Transform`, `Mesh`, `Material`) | Bevy manages GPU resources per entity |
| Camera state | `Camera3d` + `Transform` + custom `OrbitCamera` resource | Bevy camera system |
| UI state | `SimulationConfig` resource, egui panels read/write it | Centralized config, replaces Go `AppState` |
| Trails visual | `Polyline` entities from `bevy_polyline` | GPU-accelerated line rendering |
| Belt particles | 1500 instanced entities with `Transform` | Bevy auto-instances identical meshes |
| Input handling | Bevy `Input<MouseButton>`, `Input<KeyCode>`, `EventReader<MouseWheel>` | Standard Bevy input |
| Events | `SimCommand`, `BodySelected`, `LaunchComputed` | Bevy event system replaces Go channels |

### OUTSIDE Bevy ECS (solar_sim_core)

| Concern | Mechanism | Rationale |
|---------|-----------|-----------|
| N-body integration | `Simulation::step(dt)` method | Must be testable without Bevy |
| Acceleration computation | `Simulation::calculate_acceleration()` | Performance-critical, pure math |
| GR correction | `gr::calculate_gr_correction()` | Pure physics formula |
| Orbital element conversion | `builder::from_elements()` | Initialization logic |
| Launch planning | `Planner::plan()`, `Propagator::propagate()` | Algorithmic, no rendering |
| Validation | `validation::run_all()` | Must run headless |
| Body catalog | Static data arrays (`PLANET_DATA`, `MOON_DATA`, etc.) | Data, not behavior |
| Spacetime potential | `spacetime::compute_potential_field()` | CPU math, returns grid data |

### The Bridge: PhysicsPlugin Systems

The `PhysicsPlugin` in `solar_sim_bevy` bridges these two worlds:

```rust
// In solar_sim_bevy/src/plugins/physics.rs

/// System that runs in FixedUpdate. Calls solar_sim_core's step function,
/// then syncs results back to ECS components.
fn step_simulation(
    config: Res<SimulationConfig>,
    mut sim: ResMut<SimulationState>,
    mut query: Query<(&CelestialBody, &mut Transform, &mut Velocity, &mut Orbit)>,
) {
    if !config.is_playing {
        return;
    }

    let effective_dt = config.fixed_dt * config.time_speed;
    sim.inner.step(effective_dt);  // Call into solar_sim_core

    // Sync positions back to ECS
    for (body, mut transform, mut velocity, mut orbit) in &mut query {
        let state = sim.inner.get_body_state(body.sim_index);
        transform.translation = Vec3::new(
            state.position.x as f32,
            state.position.y as f32,
            state.position.z as f32,
        );
        velocity.0 = state.velocity;
        // Trail management also in solar_sim_core
    }
}
```

## How Go and Bevy Coexist During Migration

### Phase A (Physics Crate Extraction)

```
Go app ──(CGO/FFI)──> solar_sim_core (cdylib, --features ffi)
                      ^
                      Renamed from physics_core, extended with new features
```

The Go app continues to use the Rust physics backend exactly as before. The `ffi.rs` module is preserved behind a feature flag. Building `cargo build --features ffi` produces the shared library. Building `cargo build` (default) produces a regular Rust library for Bevy.

The Go app still owns all rendering and UI. No Bevy binary exists yet.

### Phase B (Bevy Window + Planets + Camera)

```
Go app ──(CGO/FFI)──> solar_sim_core (cdylib, --features ffi)

solar_sim_bevy ──────> solar_sim_core (lib, default features)
```

Two independent binaries coexist:
- `go build ./cmd/gui` produces the Go app (unchanged)
- `cargo run -p solar_sim_bevy` produces the Bevy app

Both use the same physics library. The Bevy app starts minimal (planets + camera only). The Go app remains the "production" binary.

### Phase C-D (Feature Parity)

As the Bevy app gains features, the Go app stays buildable. The Go app is only deprecated when the Bevy app passes the full validation suite and achieves visual feature parity.

**No Go code is deleted until the Bevy equivalent is proven.** The gate for deprecation:

1. Rust validation tests pass with same tolerances as Go (`1e-4 * years` energy drift, 42-44 arcsec/century Mercury precession)
2. All visual features present (trails, belt, spacetime, labels, comet tails, textures, lighting)
3. All UI panels functional (controls, bodies, launch planner, physics, status bar)
4. Launch planner produces same delta-v budgets as Go
5. Screenshot comparison shows no visual regressions

### Build Commands During Migration

```bash
# Go app (unchanged throughout)
go build -o bin/solar-system-sim ./cmd/gui

# Go app with Rust physics backend (Phase A onwards)
cargo build --features ffi -p solar_sim_core
go build -tags rust_physics -o bin/solar-system-sim ./cmd/gui

# Bevy app (Phase B onwards)
cargo run -p solar_sim_bevy

# Run Rust validation (Phase A onwards)
cargo test -p solar_sim_core -- validation

# Run Go validation (unchanged)
go run ./cmd/cli validate
```
