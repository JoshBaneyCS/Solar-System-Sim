# Current vs Target Simulation Architecture

Side-by-side comparison of the Go (current) and Bevy (target) simulation architectures.

---

## Architecture Comparison

| Concern | Go (Current) | Bevy (Target) | Migration Notes |
|---------|-------------|---------------|-----------------|
| **State ownership** | `Simulator.mu sync.RWMutex` protects all mutable state. Physics goroutine holds write lock during step; render holds read lock during snapshot. | Bevy ECS system ordering. `SimulationState` is a `ResMut` in physics systems, `Res` in render systems. No explicit locks. | Bevy's scheduler guarantees exclusive access. Systems in the same stage run in parallel only if they access disjoint resources. `FixedUpdate` systems are chained, so physics always completes before ECS sync. |
| **State transfer** | `atomic.Pointer[SimSnapshot]` -- physics publishes a deep-copied snapshot via atomic store; render reads via atomic load. Zero lock contention on the hot path. | Direct ECS component queries. After `sync_ecs_from_simulation` runs, all `Transform`/`Velocity`/`Orbit` components are up-to-date. Render systems read components directly. | **Eliminates 108 KB/tick of trail copying.** Bevy's ECS change detection is more efficient than deep-copying the entire state. Trail data lives in `Orbit.trail_history` components, read directly by the trail rendering system. |
| **Command pattern** | `SimCommand` struct with `Apply func(s *Simulator)` closure. Sent via buffered channel (cap 32). Fallback: direct lock + apply if channel full or physics loop not running. (`simulator.go:29-31, 620-636`) | `Event<SimCommand>` enum. UI systems `send()` events; physics systems `read()` events. No channel, no closure -- pattern matching on enum variants. (`solar_sim_bevy/events.rs`) | Go closures capture arbitrary state, which is flexible but untyped. Bevy events are typed enums, which enables exhaustive matching and better IDE support. The fallback for channel-full is unnecessary because Bevy events are unbounded per frame. |
| **Physics threading** | Dedicated goroutine started by `StartPhysicsLoop()`. 16ms ticker. Runs independently of render. (`simulator.go:652-689`) | Bevy `FixedUpdate` schedule at 60 Hz. Runs on the main thread (or a compute thread if configured). Bevy accumulates wall-clock time and runs 0-N fixed ticks per frame to stay on schedule. | Go's dedicated goroutine decouples physics from render frame rate (physics always runs at ~62.5 Hz regardless of render FPS). Bevy's `FixedUpdate` achieves the same: if render drops to 30 FPS, Bevy runs 2 fixed ticks per frame. If render is 120 FPS, Bevy runs 0 or 1 tick per frame. Both achieve deterministic physics independent of display refresh. |
| **Trail management** | `[]math3d.Vec3` on each `Body`. `append()` + `Trail[1:]` truncation. Max 500 points. Trail data deep-copied into every snapshot. (`simulator.go:378-383`) | `VecDeque<DVec3>` in `Orbit` component. `pop_front()` + `push_back()`. Max 500 points (configurable). No deep copy needed -- Bevy reads component directly. | `VecDeque` is a ring buffer: `pop_front()` is O(1) with no memory leak. Go's `Trail[1:]` re-slices without releasing the oldest element's backing memory. The Go approach also copies all trail data into every atomic snapshot (~108 KB per tick for 9 bodies at 500 points). The Bevy approach eliminates this entirely. |
| **Time control** | `TimeSpeed float64` multiplied into `effectiveDt = BaseTimeStep * TimeSpeed`. Negative values reverse time. Speed range 2^(-10) to 2^(10). (`simulator.go:394`) | `SimulationConfig.time_speed: f64` applied identically: `effective_dt = fixed_dt * time_speed`. Same range. Same substep logic. | Direct 1:1 mapping. No architectural change. |
| **Integrator selection** | `Integrator IntegratorType` field (enum: `IntegratorRK4`, `IntegratorVerlet`). `Step()` dispatches via `if s.Integrator == IntegratorVerlet`. (`simulator.go:314-317`) | `SimulationConfig.integrator: IntegratorType` resource. `step()` dispatches via `match self.integrator`. | Functionally identical. Rust `match` is exhaustive, preventing missing cases. |
| **Body addition/removal** | Slice reallocation. `s.Planets = append(s.Planets, body)` for add. `RemoveBodiesByType()` creates a filtered slice. RK4 scratch `ensureSize()` after growth. (`simulator.go:469-515`) | Bevy entity spawn/despawn. `commands.spawn(PlanetBundle{...})`. `commands.entity(e).despawn()`. `SimulationState.inner` arrays resized via `add_body()`/`remove_bodies_by_type()`. Entity map updated. | **Key difference:** Go's slice append can invalidate existing body pointers (documented race condition with `FollowBody *physics.Body`). Bevy entities are stable handles -- spawning/despawning never invalidates other entity references. The `FollowBody` pointer bug (risks doc item #3) is eliminated by design. |
| **Validation** | Go test suite in `internal/validation/`. 5 scenarios: energy, angular momentum, Kepler (Earth, Mercury), Mercury precession. Run via `go test` or `go run ./cmd/cli validate`. (`internal/validation/`) | Rust test suite in `solar_sim_core/src/validation/`. Same 5 scenarios, same tolerances, same golden values. Run via `cargo test -p solar_sim_core -- validation`. | **Same golden values.** Both use IEEE 754 f64, same G, same C, same orbital elements. The Rust validation must reproduce identical results to within floating-point tolerance. Position tolerance: 0.1 m after 100 steps; velocity tolerance: 1e-6 m/s. |

---

## Detailed Comparisons

### State Ownership and Synchronization

**Go:**
```
Physics Goroutine                    Render Thread
    |                                     |
    mu.Lock()                             |
    drainCommands()                       |
    Step(effectiveDt)                     |
    publishSnapshot() -- deep copy        |
    mu.Unlock()                           |
    |                                     |
    |                    snap = GetSnapshot() -- atomic load
    |                    render(snap)
    |                    // snap is immutable, no lock needed
```

The atomic snapshot pattern ensures the render thread never blocks on physics. The cost is a full deep copy per tick.

**Bevy:**
```
FixedUpdate Schedule (sequential):
    process_sim_commands()     // reads Event<SimCommand>, mutates SimulationConfig
    step_simulation()          // mutates SimulationState
    sync_ecs_from_simulation() // reads SimulationState, mutates Transform/Velocity
    manage_trails()            // reads SimulationState, mutates Orbit

Update Schedule (can overlap with next FixedUpdate):
    render_planets()           // reads Transform, Mesh, Material
    render_trails()            // reads Orbit.trail_history
    update_ui()                // reads SimulationConfig, SimulationClock
```

Bevy's ECS guarantees that systems accessing the same resource/component with `ResMut`/`Mut` never run concurrently. The `chain()` ordering in `FixedUpdate` ensures sequential execution. Render systems in `Update` read the data written in the most recent `FixedUpdate`.

### Command Pattern

**Go SimCommand:**
```go
type SimCommand struct {
    Apply func(s *Simulator)
}

// Usage:
sim.SendCommand(SimCommand{
    Apply: func(s *Simulator) {
        s.TimeSpeed = newSpeed
    },
})
```

- Channel-based (buffered, cap 32)
- Closure captures arbitrary state
- Fallback: direct lock + apply if channel full
- No type safety on what the command does

**Bevy SimCommand:**
```rust
#[derive(Event)]
pub enum SimCommand {
    SetPlaying(bool),
    SetTimeSpeed(f64),
    SetIntegrator(IntegratorType),
    SetSunMass(f64),
    SetPlanetGravity(bool),
    SetRelativity(bool),
    SetShowMoons(bool),
    SetShowComets(bool),
    SetShowAsteroids(bool),
    Reset,
    ClearTrails,
    // ...
}

// Usage:
fn ui_system(mut commands: EventWriter<SimCommand>) {
    commands.send(SimCommand::SetTimeSpeed(2.0));
}
```

- Bevy event buffer (unbounded per frame, auto-cleared)
- Typed enum variants
- Exhaustive matching in handler
- No fallback needed

### Physics Threading Model

**Go:**
```
Main thread (Fyne event loop)
    |
    +-- Render goroutine (16ms ticker)
    |     reads atomic.Pointer[SimSnapshot]
    |
    +-- Physics goroutine (16ms ticker)
          reads channel[SimCommand]
          writes atomic.Pointer[SimSnapshot]
          holds mu.Lock during step
```

Two independent timers. Physics and render can drift relative to each other. The atomic snapshot ensures they stay consistent (render always sees a complete state).

**Bevy:**
```
Main thread
    |
    +-- FixedUpdate (accumulated time, targets 60 Hz)
    |     |
    |     +-- step_simulation (physics)
    |     +-- sync_ecs (ECS update)
    |     +-- manage_trails
    |
    +-- Update (every frame, follows vsync)
          |
          +-- render (reads ECS)
          +-- ui (reads/writes resources)
```

Single-threaded execution with deterministic ordering. Bevy may run 0, 1, or 2+ `FixedUpdate` ticks per frame to maintain the target rate. This is equivalent to Go's independent ticker but with tighter integration.

### Integrator Architecture

**Go:**
```go
// simulator.go:294-317
func (s *Simulator) Step(dt float64) {
    if s.Backend != nil {
        // Rust backend path
        s.Backend.Step(dt)
        ...
        return
    }
    if s.Integrator == IntegratorVerlet {
        s.stepVerlet(dt)
        return
    }
    // RK4 path
    ...
}
```

Three paths: Rust backend, Verlet, RK4. The Rust backend is an optimization path via FFI.

**Rust (target):**
```rust
// sim.rs
impl Simulation {
    pub fn step(&mut self, dt: f64) {
        match self.integrator {
            IntegratorType::RK4 => self.step_rk4(dt),
            IntegratorType::Verlet => self.step_verlet(dt),
        }
        self.current_time += dt;
    }
}
```

Two paths: RK4 and Verlet. No FFI indirection since Rust IS the physics engine.

### Body Addition/Removal

**Go:**
```go
func (s *Simulator) AddMoons() {
    for _, moonData := range MoonData {
        // Find parent by name
        var parent *Body
        for i := range s.Planets {
            if s.Planets[i].Name == moonData.ParentName {
                parent = &s.Planets[i]
                break
            }
        }
        if parent == nil { continue }
        s.Planets = append(s.Planets, s.CreateMoonFromElements(moonData, *parent))
    }
    s.ShowMoons = true
}

func (s *Simulator) RemoveBodiesByType(t BodyType) {
    filtered := make([]Body, 0, len(s.Planets))
    for _, b := range s.Planets {
        if b.Type != t {
            filtered = append(filtered, b)
        }
    }
    s.Planets = filtered
}
```

**Bevy:**
```rust
// In process_sim_commands system:
SimCommand::SetShowMoons(true) => {
    let moon_bodies = sim.inner.add_moons(); // returns Vec<(usize, BodyInfo)>
    for (sim_idx, info) in moon_bodies {
        let entity = commands.spawn(PlanetBundle {
            body: CelestialBody {
                sim_index: sim_idx,
                name: info.name,
                mass: sim.inner.masses[sim_idx],
                body_type: info.body_type,
            },
            // ... mesh, material, transform, etc.
        }).id();
        sim.entity_map.push(entity);
    }
}

SimCommand::SetShowMoons(false) => {
    let removed_indices = sim.inner.remove_bodies_by_type(BodyType::Moon);
    for idx in removed_indices {
        if let Some(entity) = sim.entity_map.get(idx) {
            commands.entity(*entity).despawn_recursive();
        }
    }
    // Rebuild entity_map for remaining bodies
}
```

**Key difference:** Go's `RemoveBodiesByType` creates a new slice, which invalidates any pointers into the old slice (the `FollowBody` bug). Bevy's entity despawn only removes the specific entities -- all other entity handles remain valid.

### Validation Suite Comparison

| Scenario | Go Tolerance | Rust Target Tolerance | Implementation Match? |
|----------|-------------|----------------------|----------------------|
| Energy conservation | `1e-4 * years` relative drift | Same: `1e-4 * years` | Yes. Both use same formula: `|E1 - E0| / |E0|` |
| Angular momentum | `1e-6` relative drift | Same: `1e-6` | Yes. Both use `|L1_mag - L0_mag| / |L0_mag|` |
| Kepler period (Earth) | 1% relative error | Same: 1% | Yes. Both track atan2 cumulative angle crossings |
| Kepler period (Mercury) | 1% relative error | Same: 1% | Yes |
| Mercury precession | 42-44 arcsec/century (tolerance 70%) | Same: 42-44 arcsec/century | Yes. Both use LRL vector, linear regression, GR minus Newton subtraction |

**Golden test values:**
| Test | Steps | Position tolerance | Velocity tolerance |
|------|-------|-------------------|-------------------|
| `golden100` | 100 x 7200s | 0.1 m | 1e-6 m/s |
| `golden1000` | 1000 x 7200s | 0.1 m | 1e-6 m/s |

The golden values from `internal/physics/golden_test.go` must be reproduced exactly by the Rust implementation. Both use:
- RK4 integrator
- `PlanetGravityEnabled = true`
- `RelativisticEffects = true` (GR applied to all bodies in Go, via per-body flags in Rust)
- `BaseTimeStep = 7200.0` seconds
- Same 9 planet orbital elements from `PlanetData`

**Potential divergence point:** Go applies GR to ALL bodies (`s.RelativisticEffects` is a global bool). Rust applies GR only to bodies with `gr_flags[i] = true`. The FFI bridge sets `gr_flags[0] = true` (Mercury only). For golden test parity, the Rust implementation must either:
1. Apply GR to all bodies (matching Go), or
2. Accept slightly different golden values when GR is applied only to Mercury

Recommendation: Apply GR to all bodies by default in `solar_sim_core`, matching Go behavior. The per-body `gr_flags` approach can be kept as an optimization but should default to all-true when `relativistic_effects = true`.

---

## Data Flow Comparison

### Go Data Flow

```
UI Thread                              Physics Goroutine
    |                                        |
    |  SendCommand(SimCommand{              |
    |    Apply: func(s) {                   |
    |      s.TimeSpeed = 2.0               |
    |    },                                 |
    |  })                                   |
    |  ---> channel (cap 32) ------------->  |
    |                                        | drainCommands()
    |                                        |   cmd.Apply(s)  // s.TimeSpeed = 2.0
    |                                        | Step(effectiveDt)
    |                                        |   stepVerlet(dt) or RK4
    |                                        |   trail append
    |                                        | publishSnapshot()
    |                                        |   deep copy all bodies + trails
    |                                        |   atomic.Store(snap)
    |                                        |
    | snap = GetSnapshot()                   |
    |   atomic.Load()                        |
    | render(snap)                           |
    |   for each body in snap.Planets:       |
    |     WorldToScreen(body.Position)       |
    |     draw circle/image                  |
    |   for each trail:                      |
    |     CatmullRom + Bresenham             |
```

### Bevy Data Flow

```
FixedUpdate (60 Hz):
    process_sim_commands:
        for event in sim_commands.read():
            match event:
                SetTimeSpeed(v) => config.time_speed = v
                ...

    step_simulation:
        if config.is_playing:
            sim.inner.step(config.fixed_dt * config.time_speed)

    sync_ecs_from_simulation:
        for (body, transform, velocity) in query:
            transform.translation = physics_to_render(sim.inner.positions[body.sim_index])
            velocity.0 = sim.inner.velocities[body.sim_index]

    manage_trails:
        for (body, orbit) in query:
            orbit.trail_history.push_back(sim.inner.positions[body.sim_index])

Update (every frame):
    render:
        // Bevy's built-in PBR pipeline reads Transform + Mesh + Material
        // No manual WorldToScreen needed -- Bevy's camera handles projection

    trail_render:
        for (orbit,) in query:
            update polyline vertices from orbit.trail_history

    ui_system:
        egui panels read/write SimulationConfig, SimulationClock, LaunchState
```

### Key Architectural Differences Summary

| Aspect | Go Advantage | Bevy Advantage |
|--------|-------------|----------------|
| Lock-free render | Atomic snapshot eliminates lock contention | N/A (Bevy scheduling eliminates the need) |
| Memory efficiency | N/A | No deep copy per tick; direct component reads |
| Flexibility | Closures in commands can do anything | N/A |
| Type safety | N/A | Typed command enum with exhaustive matching |
| Body pointer stability | N/A | Entity handles survive spawn/despawn |
| Trail memory | N/A | VecDeque ring buffer; no memory leak |
| Debugging | goroutine stack traces | Bevy system ordering is declarative and inspectable |
| Render pipeline | Full control over pixel-level rendering | PBR, bloom, shadows out of the box |
| Threading simplicity | Explicit goroutines (familiar to Go devs) | ECS scheduling (declarative, less error-prone) |
