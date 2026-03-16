# Simulation Port Plan: Go to Rust

File-by-file mapping from Go physics implementation to the target `solar_sim_core` Rust crate.

---

## GR Formula Verification

**Go** (`internal/physics/gr/correction.go:23-29`):
```go
coeff := GM / (c * c * r * r * r)
term1 := bodyPos.Mul(4*GM/r - v2)
term2 := bodyVel.Mul(4 * rdotv)
return term1.Add(term2).Mul(coeff)
```

**Rust** (`crates/physics_core/src/gr.rs:22-28`):
```rust
let coeff = gm / (C * C * r * r * r);
let term1 = body_pos.mul(4.0 * gm / r - v2);
let term2 = body_vel.mul(4.0 * rdotv);
term1.add(term2).mul(coeff)
```

**Verdict: MATCH.** Both implement the standard 1PN formula `a_GR = (GM/(c^2*r^3)) * [(4GM/r - v^2)r + 4(r.v)v]`. The Rust code was updated to match the corrected Go formula. No discrepancy.

---

## File-by-File Mapping

### 1. simulator.go -> sim.rs

| Go Source | Go Location | Rust Target | Rust Location | Status | Gap |
|-----------|------------|-------------|---------------|--------|-----|
| `SimSnapshot` struct | `simulator.go:18-24` | N/A (Bevy ECS replaces this) | `solar_sim_bevy/resources.rs` | **Not needed in core** | Bevy's `SimulationState` resource + ECS component queries replace the atomic snapshot pattern. The core crate exposes state via accessor methods instead. |
| `SimCommand` struct | `simulator.go:29-31` | N/A (Bevy events replace this) | `solar_sim_bevy/events.rs` | **Not needed in core** | Bevy's `Event` system replaces the Go channel+closure pattern. |
| `MaxSafeDt` constant | `simulator.go:35` | `pub const MAX_SAFE_DT: f64 = 28800.0` | `sim.rs` | **Missing** | Must add to `Simulation` or as a module-level constant. |
| `parallelThreshold` constant | `simulator.go:38` | `PARALLEL_THRESHOLD = 16` | `sim.rs:8` | **Exists** | Values differ: Go=12, Rust=16. Harmonize to 12 for consistency. |
| `rk4Scratch` struct | `simulator.go:41-52` | `RK4Scratch` struct | `sim.rs:24-41` | **Exists, partial** | Rust has 16 buffers (pos0..k4v). Go has 16 buffers + `snapshot` (BodyState vec). Rust uses position slice directly as snapshot. **No gap** -- Rust approach is equivalent since it only needs positions for the snapshot. |
| `newRK4Scratch()` / `ensureSize()` | `simulator.go:54-86` | `RK4Scratch::new()` / `ensure_size()` | `sim.rs:43-72` | **Exists** | Equivalent logic. |
| `Simulator` struct | `simulator.go:89-113` | `Simulation` struct | `sim.rs:10-21` | **Exists, partial** | **Missing fields:** `Integrator` (enum), `softening_length`, `current_time`, `time_speed`, `is_playing`, `show_trails`, `max_trail_len`, body names/types/colors, trail storage. The Rust struct is a minimal physics engine. See detailed gap below. |
| `NewSimulator()` | `simulator.go:127-160` | `Simulation::new()` | `sim.rs:75-94` | **Exists, partial** | Go version creates Sun, iterates `PlanetData`, creates bodies from orbital elements. Rust version takes raw arrays from FFI. **Missing:** body catalog initialization, `CreatePlanetFromElements`, default config. |
| `CreatePlanetFromElements()` | `simulator.go:162-218` | N/A | `bodies/builder.rs` | **Missing** | Must port: orbital element to Cartesian conversion using perifocal frame, mu/h velocity formula. ~56 lines of trig. |
| `CalculateAccelerationWithSnapshot()` | `simulator.go:220-268` | `calculate_acceleration()` | `sim.rs:97-146` | **Exists** | Nearly identical. **Differences:** (1) Go uses `s.SofteningLength` in denominator, Rust does not. (2) Go applies GR based on `s.RelativisticEffects` bool, Rust uses per-body `gr_flags[i]`. (3) Go passes `bodyMass` and `bodyName` params (unused in calculation). |
| `computeAccelerations()` | `simulator.go:272-292` | `compute_accelerations()` | `sim.rs:149-175` | **Exists** | Go uses goroutines + WaitGroup. Rust uses rayon `into_par_iter()`. Functionally equivalent. |
| `Step()` (RK4 path) | `simulator.go:294-387` | `step()` | `sim.rs:177-258` | **Exists** | RK4 logic matches. **Missing in Rust:** (1) Backend dispatch, (2) Verlet fallback, (3) trail management after step, (4) `CurrentTime += dt`. |
| `Step()` (Backend path) | `simulator.go:295-312` | N/A | N/A | **Not needed** | Backend dispatch is a Go-specific FFI concern. In the target architecture, the Rust crate IS the physics engine. |
| `Update()` (substep logic) | `simulator.go:389-406` | N/A | `sim.rs` | **Missing** | Must port: `effectiveDt = dt * TimeSpeed`, substep when `|effectiveDt| > MaxSafeDt`. Key method for Bevy `FixedUpdate` integration. |
| `CreateMoonFromElements()` | `simulator.go:411-465` | N/A | `bodies/builder.rs` | **Missing** | Same orbital element conversion as `CreatePlanetFromElements` but adds parent's heliocentric state. Uses parent body's mass as GM source. |
| `AddMoons()` | `simulator.go:469-484` | N/A | `bodies/builder.rs` | **Missing** | Iterates MoonData, finds parent by name, calls `CreateMoonFromElements`. |
| `AddComets()` | `simulator.go:487-494` | N/A | `bodies/builder.rs` | **Missing** | Iterates CometData, creates via `CreatePlanetFromElements`, sets type. |
| `AddAsteroids()` | `simulator.go:497-504` | N/A | `bodies/builder.rs` | **Missing** | Iterates AsteroidData, creates via `CreatePlanetFromElements`. |
| `RemoveBodiesByType()` | `simulator.go:507-515` | N/A | `sim.rs` or `bodies/builder.rs` | **Missing** | Filters body arrays by type. In Rust, this means removing from positions/velocities/masses/gr_flags arrays by index. |
| `SetSunMass()` | `simulator.go:517-520` | Direct field mutation | `sim.rs` | **Exists** (trivial) | `sim.sun_mass = mass;` |
| `ClearTrails()` | `simulator.go:529-536` | N/A | `trail.rs` | **Missing** | Clears all trail buffers. |
| `publishSnapshot()` | `simulator.go:578-609` | N/A | N/A | **Not needed in core** | Bevy's ECS sync system replaces this. |
| `GetSnapshot()` | `simulator.go:613-615` | N/A | N/A | **Not needed in core** | Bevy queries replace atomic snapshot reads. |
| `SendCommand()` / `drainCommands()` | `simulator.go:620-648` | N/A | N/A | **Not needed in core** | Bevy events replace channels. |
| `StartPhysicsLoop()` | `simulator.go:652-689` | N/A | N/A | **Not needed in core** | Bevy `FixedUpdate` schedule replaces the dedicated goroutine. |
| `StopPhysicsLoop()` | `simulator.go:692-696` | N/A | N/A | **Not needed in core** | No background thread to stop. |

#### Detailed Gaps in `Simulation` Struct

Fields present in Go `Simulator` but missing from Rust `Simulation`:

| Go Field | Type | Needed in Rust Core? | Notes |
|----------|------|---------------------|-------|
| `Sun` | `Body` | No | Sun is at origin in Rust. Position/velocity can be added if needed. |
| `TimeSpeed` | `float64` | Yes | Needed for `update()` substep logic. |
| `IsPlaying` | `bool` | No | UI state, belongs in Bevy resource. |
| `ShowTrails` | `bool` | No | Display state, belongs in Bevy resource. |
| `CurrentTime` | `float64` | Yes | Simulation clock. Essential. |
| `SunMass` | `float64` | Already exists as `sun_mass`. | -- |
| `DefaultMass` | `float64` | Yes | For sun mass reset. |
| `maxTrailLen` | `int` | Yes | If trails are managed in core. |
| `PlanetGravityEnabled` | `bool` | Already exists as `planet_gravity`. | -- |
| `RelativisticEffects` | `bool` | Partially | Rust uses per-body `gr_flags`. Need global toggle that sets all flags. |
| `Integrator` | `IntegratorType` | Yes | Must add Verlet option. |
| `SofteningLength` | `float64` | Optional | Go default is 0 (unused). Can skip. |
| `Backend` | `PhysicsBackend` | No | N/A in pure Rust. |
| `ShowMoons/Comets/Asteroids` | `bool` | No | UI state, belongs in Bevy resource. |
| Body metadata (name, color, type, radius) | On each `Body` | Yes | Rust currently has only mass/pos/vel. Need `BodyInfo` struct. |

---

### 2. integrator.go -> integrators/mod.rs

| Go Source | Go Location | Rust Target | Status | Gap |
|-----------|------------|-------------|--------|-----|
| `IntegratorType` enum | `integrator.go:4-11` | `integrators/mod.rs` | **Missing** | Add `enum IntegratorType { RK4, Verlet }` |
| `IntegratorRK4` | `integrator.go:8` | N/A | **Missing** | Enum variant |
| `IntegratorVerlet` | `integrator.go:10` | N/A | **Missing** | Enum variant |

Function mapping:
- Go `IntegratorType` -> Rust `pub enum IntegratorType { RK4, Verlet }`
- The integrator dispatch is in `Simulator.Step()` -> should become `Simulation::step()` with match on integrator type.

---

### 3. verlet.go -> integrators/verlet.rs

| Go Source | Go Location | Rust Target | Status | Gap |
|-----------|------------|-------------|--------|-----|
| `stepVerlet()` | `verlet.go:15-68` | `integrators/verlet.rs` | **Missing entirely** | Must port velocity Verlet algorithm. |

**Go function signature:**
```go
func (s *Simulator) stepVerlet(dt float64)
```

**Target Rust signature:**
```rust
pub fn step_verlet(sim: &mut Simulation, dt: f64)
// or as a method:
impl Simulation { pub fn step_verlet(&mut self, dt: f64) }
```

**Key implementation details to port:**
1. Build `BodyState` snapshot of current positions/velocities
2. Compute current accelerations for all bodies
3. Half-step velocity: `v_half = v + a * dt/2`
4. Full-step position: `x_new = x + v_half * dt`
5. Build new snapshot with new positions and half velocities
6. Compute new accelerations at new positions
7. Complete velocity: `v_new = v_half + a_new * dt/2`
8. Trail management (if in core)
9. `CurrentTime += dt`

**Note:** Go's `stepVerlet` allocates 3 temporary Vec arrays per call (`states`, `accel`, `halfVel`, `newStates`). Rust should pre-allocate these in a `VerletScratch` struct, mirroring the RK4 scratch pattern.

---

### 4. body.go -> bodies/body.rs

| Go Source | Go Location | Rust Target | Status | Gap |
|-----------|------------|-------------|--------|-----|
| `BodyType` enum | `body.go:10-19` | `bodies/body.rs` | **Missing** | Port 6 variants: Star, Planet, DwarfPlanet, Moon, Comet, Asteroid |
| `Body` struct | `body.go:22-33` | `bodies/body.rs` | **Missing** | Rust only has flat arrays (masses, positions, velocities). Need `BodyInfo` for metadata. |
| `Planet` struct | `body.go:36-52` | `bodies/body.rs` | **Missing** | Orbital element data structure for catalog entries. |
| `BodyState` struct | `body.go:55-58` | N/A | **Not needed** | Rust uses position slices directly. |

**Go `Body` struct -> Rust mapping:**
```go
type Body struct {
    Name           string       // -> BodyInfo.name: String
    Mass           float64      // -> Simulation.masses[i]
    Position       math3d.Vec3  // -> Simulation.positions[i]
    Velocity       math3d.Vec3  // -> Simulation.velocities[i]
    Color          color.Color  // -> BodyInfo.color (rendering only, maybe Bevy-side)
    Radius         float64      // -> BodyInfo.display_radius
    Trail          []math3d.Vec3 // -> TrailManager.trails[i]
    ShowTrail      bool         // -> Bevy component
    Type           BodyType     // -> BodyInfo.body_type
    PhysicalRadius float64      // -> BodyInfo.physical_radius
}
```

**Target Rust structs:**
```rust
pub enum BodyType { Star, Planet, DwarfPlanet, Moon, Comet, Asteroid }

pub struct BodyInfo {
    pub name: String,
    pub body_type: BodyType,
    pub display_radius: f64,
    pub physical_radius: f64,
    pub color: [u8; 4],  // RGBA
}

pub struct OrbitalElements {
    pub semi_major_axis: f64,    // AU
    pub eccentricity: f64,
    pub inclination: f64,        // degrees
    pub long_ascending_node: f64,// degrees
    pub arg_perihelion: f64,     // degrees
    pub initial_anomaly: f64,    // radians
    pub orbital_period: f64,     // days
    pub mass: f64,               // kg
    pub parent_name: Option<String>,
    pub parent_mass: Option<f64>,
    pub physical_radius: f64,    // meters
}
```

---

### 5. planets.go + moons.go + comets.go + asteroids.go -> bodies/catalog.rs

| Go Source | Go Location | Rust Target | Status | Gap |
|-----------|------------|-------------|--------|-----|
| `PlanetData` (9 entries) | `planets.go:9-145` | `bodies/catalog.rs` | **Missing** | Port all 9 planet orbital elements. |
| `MoonData` (8 entries) | `moons.go:10-154` | `bodies/catalog.rs` | **Missing** | Port 8 moon entries with parent references. |
| `CometData` (4 entries) | `comets.go:10-71` | `bodies/catalog.rs` | **Missing** | Port 4 comet entries. |
| `AsteroidData` (6 entries) | `asteroids.go:11-105` | `bodies/catalog.rs` | **Missing** | Port 6 named asteroids. |
| `BeltParticle` struct | `asteroids.go:108-113` | `belt.rs` | **Missing** | Simple struct: SMA, eccentricity, inclination, initial anomaly. |
| `GenerateBeltParticles()` | `asteroids.go:116-134` | `belt.rs` | **Missing** | Deterministic RNG (seed 42), Kirkwood gap exclusion at 2.5, 2.82, 2.95 AU. |

**Data volume:** 9 + 8 + 4 + 6 = 27 catalog entries. Each entry has ~12 fields. This is ~300 lines of static data in Rust.

**Key conversion notes:**
- Moon `SemiMajorAxis` is stored as `km_value / 1.496e11` (km converted to AU) in Go. The Rust catalog should store the same.
- Initial anomalies use Go's `math.Pi` expressions (e.g., `math.Pi / 4`). Rust uses `std::f64::consts::PI / 4.0` or `FRAC_PI_4`.
- Colors are `color.RGBA{r, g, b, 255}` in Go -> `[r, g, b, 255]` in Rust.

---

### 6. gr/correction.go -> gr.rs

| Go Source | Go Location | Rust Target | Status | Gap |
|-----------|------------|-------------|--------|-----|
| `CalculateGRCorrection()` | `correction.go:16-30` | `calculate_gr_correction()` | `gr.rs:11-28` | **Exists, matches** | Verified above. Both use identical 1PN formula. |

**No work needed.** The Rust GR module is complete and correct.

---

### 7. validation/ -> validation/ (Rust test module)

| Go Source | Go Location | Rust Target | Status | Gap |
|-----------|------------|-------------|--------|-----|
| `Result` struct | `validation.go:9-17` | `validation/mod.rs` | **Missing** | Port result struct with pass/fail, measured, expected, tolerance, units. |
| `AllScenarios()` | `validation.go:34-42` | `validation/mod.rs` | **Missing** | Return scenario names. |
| `RunScenario()` | `validation.go:45-60` | `validation/mod.rs` | **Missing** | Dispatch to scenario functions. |
| `RunAll()` | `validation.go:63-70` | `validation/mod.rs` | **Missing** | Run all scenarios. |
| `ValidateEnergyConservation()` | `energy.go:11-52` | `validation/energy.rs` | **Missing** | Compute total KE+PE, run N steps, measure relative drift. Tolerance: `1e-4 * years`. |
| `ValidateAngularMomentumConservation()` | `angular_momentum.go:12-51` | `validation/angular_momentum.rs` | **Missing** | Compute total L = sum(r x v * m), measure relative magnitude drift. Tolerance: `1e-6`. Uses `PlanetGravityEnabled = false`. |
| `ValidateKeplerPeriod()` | `kepler.go:12-109` | `validation/kepler.rs` | **Missing** | Track cumulative angle via atan2, detect orbit completions, measure period. Tolerance: 1% relative. |
| `ValidateMercuryPrecession()` | `mercury_precession.go:18-131` | `validation/precession.rs` | **Missing** | Run with and without GR, compute Laplace-Runge-Lenz vector angle drift via linear regression, subtract Newton-only rate. Expected: 42-44 arcsec/century. |

**Key implementation notes:**
- All validation functions create a fresh `Simulation` (equivalent to `NewSimulator()`)
- Energy test uses `RelativisticEffects = false`
- Angular momentum test uses `PlanetGravityEnabled = false` (Sun-only, central force -> L conserved exactly)
- Precession test uses `PlanetGravityEnabled = true` (N-body perturbations present)
- LRL vector: `A = (v x L) / GM - r_hat` where `L = r x v`
- Linear regression for precession rate extraction (manual implementation, no library needed)
- `BaseTimeStep = 7200.0` seconds (2 hours)

---

### 8. golden_test.go -> Rust golden tests

| Go Source | Go Location | Rust Target | Status | Gap |
|-----------|------------|-------------|--------|-----|
| `golden100` data | `golden_test.go:17-27` | `tests/golden.rs` or `sim.rs` `#[cfg(test)]` | **Missing** | Port 9 planet position/velocity vectors after 100 RK4 steps. |
| `golden1000` data | `golden_test.go:29-39` | Same | **Missing** | Port 9 planet vectors after 1000 steps. |
| `testGoldenBaseline()` | `golden_test.go:41-67` | Same | **Missing** | Create sim, step N times with `BaseTimeStep`, compare against golden values. |
| `TestGenerateGoldenBaseline()` | `golden_test.go:77-103` | Same | **Missing** | Utility to regenerate golden values. |

**Golden test requirements:**
- Uses RK4 integrator (not Verlet)
- `PlanetGravityEnabled = true`, `RelativisticEffects = true`
- `ShowTrails = false` (no trail allocation overhead)
- Position tolerance: `1e-1` meters
- Velocity tolerance: `1e-6` m/s
- Uses `constants.BaseTimeStep = 7200.0` seconds

**Critical:** The golden values depend on identical initial conditions (same orbital elements, same G, same C). The Rust catalog must reproduce the exact same initial positions/velocities as Go's `NewSimulator()` + `CreatePlanetFromElements()`.

---

### 9. launch/ -> launch/ (new module or separate crate)

| Go Source | Go Location | Rust Target | Status | Gap |
|-----------|------------|-------------|--------|-----|
| `CircularVelocity()` | `orbital.go:7-9` | `launch/orbital.rs` | **Missing** | `sqrt(mu/r)` |
| `EscapeVelocity()` | `orbital.go:13-15` | `launch/orbital.rs` | **Missing** | `sqrt(2*mu/r)` |
| `HohmannDeltaV()` | `orbital.go:19-31` | `launch/orbital.rs` | **Missing** | Compute dv1, dv2 for Hohmann transfer. |
| `HohmannTransferTime()` | `orbital.go:35-38` | `launch/orbital.rs` | **Missing** | `pi * sqrt(a^3/mu)` |
| `PlaneChangeDV()` | `orbital.go:42-44` | `launch/orbital.rs` | **Missing** | `2*v*sin(di/2)` |
| `HyperbolicExcessDV()` | `orbital.go:49-53` | `launch/orbital.rs` | **Missing** | Hyperbolic departure. |
| `VisViva()` | `orbital.go:57-59` | `launch/orbital.rs` | **Missing** | Vis-viva equation. |
| `DeltaVBudget` struct | `planner.go:11-17` | `launch/planner.rs` | **Missing** | 4 components + total. |
| `LaunchPlan` struct | `planner.go:20-29` | `launch/planner.rs` | **Missing** | Full plan result. |
| `Planner` + `Plan()` | `planner.go:32-92` | `launch/planner.rs` | **Missing** | Main planning logic with Earth/lunar/interplanetary branches. |
| `planElliptical()` | `planner.go:95-101` | `launch/planner.rs` | **Missing** | GTO path. |
| `planLunar()` | `planner.go:104-117` | `launch/planner.rs` | **Missing** | TLI + lunar orbit insertion. |
| `planInterplanetary()` | `planner.go:120-134` | `launch/planner.rs` | **Missing** | Heliocentric Hohmann + hyperbolic excess. |
| `PropagateTrajectory()` | `planner.go:137-182` | `launch/planner.rs` | **Missing** | Trajectory propagation dispatch. |
| `Summary()` | `planner.go:185-230` | `launch/planner.rs` | **Missing** | Human-readable output. |
| `Vehicle` struct + catalog | `vehicle.go` | `launch/vehicle.rs` | **Missing** | 3 vehicles: Generic, Falcon-like, Saturn V-like. |
| `Destination` struct + catalog | `destination.go` | `launch/destination.rs` | **Missing** | 5 destinations: LEO, ISS, GTO, Moon, Mars. |
| `RocketDeltaV()` | `rocket.go:7-12` | `launch/rocket.rs` | **Missing** | Tsiolkovsky: `Isp * g0 * ln(m0/mf)` |
| `StageDeltaV()` | `rocket.go:16-20` | `launch/rocket.rs` | **Missing** | Stage dv with payload. |
| `TotalVehicleDeltaV()` | `rocket.go:24-35` | `launch/rocket.rs` | **Missing** | Multi-stage chaining. |
| `PropagatorConfig` struct | `propagator.go:8-13` | `launch/propagator.rs` | **Missing** | Mu, timestep, duration, max steps. |
| `Propagate()` | `propagator.go:17-61` | `launch/propagator.rs` | **Missing** | RK4 2-body trajectory propagation with ~1000 output points. |
| `rk4Step()` | `propagator.go:64-103` | `launch/propagator.rs` | **Missing** | Single RK4 step for 2-body. |
| `TrajectoryPoint` struct | `trajectory.go:7-11` | `launch/mod.rs` | **Missing** | Time + position + velocity. |
| `Trajectory` struct | `trajectory.go:14-17` | `launch/mod.rs` | **Missing** | Points + frame. |
| `ToHeliocentric()` / `ToEarthCentered()` | `trajectory.go:20-56` | `launch/mod.rs` | **Missing** | Frame conversion by adding/subtracting Earth position. |
| Constants (MuEarth, MuSun, etc.) | `constants.go` | `launch/mod.rs` or `constants.rs` | **Missing** | 12 physical constants for launch calculations. |
| `WriteCSV()` | `csv.go` | `launch/csv.rs` | **Missing** | CSV export of trajectory. |

**Total launch module: ~930 LOC in Go, estimated ~500-600 LOC in Rust** (less boilerplate, pattern matching).

---

## Summary: What Exists vs What's Missing

### Already Exists in Rust (physics_core)

| Component | LOC | Quality |
|-----------|-----|---------|
| `vec3.rs` -- Vec3 math | 73 | Complete, well-tested |
| `constants.rs` -- G, C | 2 | Complete (missing AU, BaseTimeStep) |
| `gr.rs` -- 1PN GR correction | 28 | Complete, correct formula, tested |
| `sim.rs` -- RK4 N-body engine | 258 | Core RK4 works, missing Verlet/substep/trails |
| `ffi.rs` -- C FFI surface | 112 | Complete for current Go interop |
| **Total existing** | **~473** | |

### Must Be Created

| Component | Target File | Estimated LOC | Priority |
|-----------|-----------|---------------|----------|
| `IntegratorType` enum + dispatch | `integrators/mod.rs` | 30 | Phase A |
| Velocity Verlet integrator | `integrators/verlet.rs` | 80 | Phase A |
| `BodyType` enum + `BodyInfo` struct | `bodies/body.rs` | 50 | Phase A |
| `OrbitalElements` struct | `bodies/body.rs` | 30 | Phase A |
| `from_elements()` + `from_moon_elements()` | `bodies/builder.rs` | 120 | Phase A |
| Body catalog (27 entries) | `bodies/catalog.rs` | 300 | Phase A |
| Trail ring buffer | `trail.rs` | 60 | Phase A |
| Belt particles + generator | `belt.rs` | 80 | Phase A |
| Substep logic (`update()`) | `sim.rs` extension | 30 | Phase A |
| `add_moons/comets/asteroids`, `remove_by_type` | `sim.rs` extension | 60 | Phase A |
| `current_time` tracking | `sim.rs` extension | 10 | Phase A |
| Validation harness (5 scenarios) | `validation/` | 300 | Phase A |
| Golden test data + assertions | `tests/golden.rs` | 100 | Phase A |
| Launch orbital mechanics | `launch/orbital.rs` | 50 | Phase A |
| Launch planner | `launch/planner.rs` | 150 | Phase A |
| Vehicles + destinations | `launch/vehicle.rs`, `launch/destination.rs` | 80 | Phase A |
| Rocket equation | `launch/rocket.rs` | 30 | Phase A |
| Trajectory propagation | `launch/propagator.rs` | 80 | Phase A |
| Trajectory types + CSV | `launch/mod.rs`, `launch/csv.rs` | 70 | Phase A |
| Launch constants | `launch/mod.rs` or `constants.rs` | 20 | Phase A |
| FFI updates (feature-gated) | `ffi.rs` | 40 | Phase A |
| **Total new code** | | **~1,770** | |

### Requires Modification

| Component | Change | Estimated LOC |
|-----------|--------|---------------|
| `sim.rs` `Simulation` struct | Add fields: `integrator`, `current_time`, `default_sun_mass`, body_info vec | 20 |
| `sim.rs` `calculate_acceleration()` | Add softening length (optional), global GR toggle | 10 |
| `sim.rs` `PARALLEL_THRESHOLD` | Change from 16 to 12 to match Go | 1 |
| `constants.rs` | Add `AU`, `BASE_TIME_STEP` | 2 |
| `Cargo.toml` | Rename to `solar_sim_core`, add `[features] ffi = []`, change crate-type | 10 |
| `lib.rs` | Add new module declarations | 10 |

---

## Function-Level Mapping Reference

| Go Function | Rust Function | File |
|-------------|--------------|------|
| `NewSimulator()` | `Simulation::new_solar_system()` | `sim.rs` |
| `CreatePlanetFromElements(p Planet)` | `Body::from_orbital_elements(elem, sun_mass)` | `bodies/builder.rs` |
| `CreateMoonFromElements(moon, parent)` | `Body::from_moon_elements(elem, parent_state)` | `bodies/builder.rs` |
| `CalculateAccelerationWithSnapshot(...)` | `Simulation::calculate_acceleration(...)` | `sim.rs` |
| `computeAccelerations(...)` | `Simulation::compute_accelerations(...)` | `sim.rs` |
| `Step(dt)` | `Simulation::step(dt)` | `sim.rs` (dispatches to RK4 or Verlet) |
| `stepVerlet(dt)` | `step_verlet(sim, dt)` | `integrators/verlet.rs` |
| `Update(dt)` | `Simulation::update(dt, time_speed)` | `sim.rs` |
| `AddMoons()` | `Simulation::add_moons()` | `sim.rs` or `bodies/builder.rs` |
| `AddComets()` | `Simulation::add_comets()` | `sim.rs` or `bodies/builder.rs` |
| `AddAsteroids()` | `Simulation::add_asteroids()` | `sim.rs` or `bodies/builder.rs` |
| `RemoveBodiesByType(t)` | `Simulation::remove_bodies_by_type(t)` | `sim.rs` |
| `SetSunMass(mult)` | `Simulation::set_sun_mass(mult)` | `sim.rs` |
| `ClearTrails()` | `TrailManager::clear_all()` | `trail.rs` |
| `GenerateBeltParticles(n)` | `generate_belt_particles(n)` | `belt.rs` |
| `CalculateGRCorrection(...)` | `gr::calculate_gr_correction(...)` | `gr.rs` (already done) |
| `ValidateEnergyConservation(y)` | `validation::energy::validate(y)` | `validation/energy.rs` |
| `ValidateAngularMomentumConservation(y)` | `validation::angular_momentum::validate(y)` | `validation/angular_momentum.rs` |
| `ValidateKeplerPeriod(name, y)` | `validation::kepler::validate(name, y)` | `validation/kepler.rs` |
| `ValidateMercuryPrecession(y)` | `validation::precession::validate(y)` | `validation/precession.rs` |
| `HohmannDeltaV(mu, r1, r2)` | `launch::orbital::hohmann_delta_v(mu, r1, r2)` | `launch/orbital.rs` |
| `Planner.Plan(v, d)` | `launch::planner::plan(v, d)` | `launch/planner.rs` |
| `Propagate(pos, vel, cfg)` | `launch::propagator::propagate(pos, vel, cfg)` | `launch/propagator.rs` |
| `RocketDeltaV(isp, m0, mf)` | `launch::rocket::delta_v(isp, m0, mf)` | `launch/rocket.rs` |
| `TotalVehicleDeltaV(v)` | `launch::rocket::total_vehicle_delta_v(v)` | `launch/rocket.rs` |

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Golden values diverge due to float ordering | High | Ensure identical computation order in RK4. Both Go and Rust use IEEE 754 f64. Verify by stepping 1 body and comparing. |
| Moon heliocentric conversion differs | Medium | Unit test `CreateMoonFromElements` with known parent state, compare Go vs Rust output. |
| Belt particle RNG differs | Low | Both should use seed 42 with the same algorithm. Go uses `math/rand` (LCG), Rust should use the same or accept visual-only differences. |
| Verlet energy conservation differs | Medium | Verlet is simpler than RK4 -- less room for divergence. Test with single-body orbit. |
| Launch planner constants differ | Low | Copy exact constants from Go `launch/constants.go` to Rust. |
