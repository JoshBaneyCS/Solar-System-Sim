# Timestep Model: Current and Target

---

## Current Go Timestep Model

### Physics Loop Architecture

The physics runs on a dedicated goroutine, decoupled from the render thread:

```
StartPhysicsLoop(dt=7200.0)  // dt = BaseTimeStep = 2 hours
    |
    +-> goroutine:
        ticker = time.NewTicker(16ms)  // ~62.5 Hz wall clock
        loop:
            <-ticker.C
            mu.Lock()
            drainCommands()           // process UI mutations
            if IsPlaying:
                effectiveDt = dt * TimeSpeed
                if |effectiveDt| <= MaxSafeDt:
                    Step(effectiveDt)
                else:
                    nSub = ceil(|effectiveDt| / MaxSafeDt)
                    subDt = effectiveDt / nSub
                    for i in 0..nSub: Step(subDt)
            publishSnapshot()         // atomic pointer swap
            mu.Unlock()
```

**Source:** `internal/physics/simulator.go:652-689`

### Key Parameters

| Parameter | Value | Source | Notes |
|-----------|-------|--------|-------|
| `BaseTimeStep` | 7200.0 s (2 hours) | `pkg/constants/constants.go:9` | Sim-seconds per physics tick at 1x speed |
| `MaxSafeDt` | 28800.0 s (8 hours) | `simulator.go:35` | Substep threshold |
| `TimeSpeed` range | 2^(-10) to 2^(10) | `ui/state.go` (feature inventory) | ~0.001x to ~1024x |
| Ticker interval | 16 ms | `simulator.go:661` | ~62.5 Hz wall clock |
| Negative TimeSpeed | Supported | `simulator.go:394` | Time reversal via `effectiveDt = dt * TimeSpeed` where TimeSpeed < 0 |

### Effective Timestep Calculation

```
effectiveDt = BaseTimeStep * TimeSpeed
```

At different speed settings:

| TimeSpeed | effectiveDt | Substeps | Per-substep dt | Sim time/wall second |
|-----------|------------|----------|----------------|---------------------|
| 1x | 7,200 s | 1 | 7,200 s | 7,200 * 62.5 = 450,000 s/s (~5.2 days/s) |
| 4x | 28,800 s | 1 | 28,800 s | 1,800,000 s/s (~20.8 days/s) |
| 8x | 57,600 s | 2 | 28,800 s | 3,600,000 s/s (~41.7 days/s) |
| 1024x | 7,372,800 s | 256 | 28,800 s | 460,800,000 s/s (~14.6 years/s) |
| -1x | -7,200 s | 1 | -7,200 s | Rewind at 5.2 days/s |

### Substep Logic Detail

From `simulator.go:389-406` (also duplicated in `StartPhysicsLoop` at lines 671-683):

```go
func (s *Simulator) Update(dt float64) {
    effectiveDt := dt * s.TimeSpeed
    absDt := math.Abs(effectiveDt)
    if absDt <= MaxSafeDt {
        s.Step(effectiveDt)
    } else {
        nSub := int(math.Ceil(absDt / MaxSafeDt))
        subDt := effectiveDt / float64(nSub)
        for i := 0; i < nSub; i++ {
            s.Step(subDt)
        }
    }
}
```

**Important:** The substep division preserves the sign of `effectiveDt`, so time reversal works correctly through substeps. Each substep calls `Step(subDt)` which calls either `stepVerlet` or the RK4 path, then increments `CurrentTime += dt` (with the signed value).

### RK4 Scratch Pre-allocation

The Go implementation avoids per-frame heap allocation for the RK4 integrator:

```go
type rk4Scratch struct {
    pos0, vel0 []math3d.Vec3  // saved initial state
    k1p, k1v   []math3d.Vec3  // k1 derivatives
    k2p, k2v   []math3d.Vec3  // k2 derivatives
    k3p, k3v   []math3d.Vec3  // k3 derivatives
    k4p, k4v   []math3d.Vec3  // k4 derivatives
    pos2, vel2 []math3d.Vec3  // midpoint state for k2
    pos3, vel3 []math3d.Vec3  // midpoint state for k3
    pos4, vel4 []math3d.Vec3  // endpoint state for k4
    snapshot   []BodyState    // position snapshot for acceleration calculation
    cap        int
}
```

**Total buffers:** 17 arrays (16 Vec3 arrays + 1 BodyState array), each of length `n_bodies`.

**Growth policy:** `ensureSize(n)` checks if `cap >= n`; if not, reallocates all arrays. This happens when bodies are added (moons, comets, asteroids).

**Source:** `simulator.go:41-86`

### Verlet Allocation Pattern

Unlike RK4, the Verlet integrator allocates fresh temporary arrays each step:

```go
func (s *Simulator) stepVerlet(dt float64) {
    states := make([]BodyState, n)    // current state snapshot
    accel := make([]math3d.Vec3, n)   // current accelerations
    halfVel := make([]math3d.Vec3, n) // half-step velocities
    newStates := make([]BodyState, n) // new state for second acceleration
    ...
}
```

**Source:** `internal/physics/verlet.go:15-68`

This is a performance gap: 4 allocations per step, each of `n_bodies * 24` bytes (Vec3 = 3 x f64). For 9 planets, this is trivial. For 28 bodies (with moons, comets, asteroids), still small. But a pre-allocated scratch would be more consistent with the RK4 pattern.

### Trail Management

Trails are managed inline during `Step()`:

```go
// After computing new position:
if s.ShowTrails && s.Planets[i].ShowTrail {
    planet.Trail = append(planet.Trail, planet.Position)
    if len(planet.Trail) > s.maxTrailLen {
        planet.Trail = planet.Trail[1:]
    }
}
```

**Source:** `simulator.go:378-383` (RK4), `verlet.go:59-64` (Verlet), `simulator.go:303-308` (backend)

**Trail parameters:**
- `maxTrailLen = 500` (set in `NewSimulator()` at `simulator.go:144`)
- Go uses slice append + truncation: `Trail[1:]` drops the oldest point
- This creates a memory issue noted in risks: the backing array never shrinks

### Snapshot Publishing

After each physics tick (regardless of substep count), a full deep copy is published atomically:

```go
func (s *Simulator) publishSnapshot() {
    snap := &SimSnapshot{
        Planets: make([]Body, len(s.Planets)),
        Sun:     Body{...},  // copy
        CurrentTime: s.CurrentTime,
        TimeSpeed:   s.TimeSpeed,
        IsPlaying:   s.IsPlaying,
    }
    for i := range s.Planets {
        snap.Planets[i] = Body{...}  // deep copy including trail
        copy(snap.Planets[i].Trail, s.Planets[i].Trail)
    }
    s.latestSnapshot.Store(snap)  // atomic pointer swap
}
```

**Cost:** For 9 planets with 500-point trails, this is `9 * 500 * 24 = 108 KB` of trail data copied per tick, plus body metadata. At 62.5 Hz, that is ~6.75 MB/s of allocation and copying. GC handles the old snapshots.

---

## Target Bevy Timestep Model

### Bevy FixedUpdate Schedule

Bevy's `FixedUpdate` schedule runs at a configurable fixed rate, independent of the render frame rate. The physics system runs inside this schedule.

```rust
// In solar_sim_bevy/src/plugins/physics.rs

pub struct PhysicsPlugin;

impl Plugin for PhysicsPlugin {
    fn build(&self, app: &mut App) {
        app
            .insert_resource(Time::<Fixed>::from_hz(60.0))  // 60 Hz physics
            .insert_resource(SimulationState::new())
            .insert_resource(SimulationConfig::default())
            .insert_resource(SimulationClock::default())
            .add_systems(FixedUpdate, (
                process_sim_commands,
                step_simulation,
                sync_ecs_from_simulation,
                manage_trails,
                update_clock,
            ).chain());
    }
}
```

### TimeSpeed Mapping

The Go model applies `TimeSpeed` as a multiplier to the base timestep. In Bevy, the same approach works without modifying `Time<Fixed>`:

```rust
fn step_simulation(
    config: Res<SimulationConfig>,
    mut sim: ResMut<SimulationState>,
) {
    if !config.is_playing {
        return;
    }

    // Base dt: the sim-seconds per physics tick at 1x speed
    // At 60 Hz FixedUpdate, each tick represents BASE_TIME_STEP sim-seconds
    let effective_dt = config.fixed_dt * config.time_speed;

    let abs_dt = effective_dt.abs();
    if abs_dt <= MAX_SAFE_DT {
        sim.inner.step(effective_dt);
    } else {
        let n_sub = (abs_dt / MAX_SAFE_DT).ceil() as usize;
        let sub_dt = effective_dt / n_sub as f64;
        for _ in 0..n_sub {
            sim.inner.step(sub_dt);
        }
    }

    sim.inner.current_time += effective_dt;
}
```

**Key insight:** We do NOT change `Time<Fixed>` based on `TimeSpeed`. The fixed timestep stays at 60 Hz wall clock. The `TimeSpeed` multiplier only affects how many sim-seconds each tick advances. This preserves Bevy's deterministic fixed update behavior.

### Comparison: Go vs Bevy Tick Rate

| Aspect | Go | Bevy |
|--------|-----|------|
| Wall clock tick rate | ~62.5 Hz (16ms ticker) | 60 Hz (`Time<Fixed>::from_hz(60.0)`) |
| Base sim-seconds per tick | 7,200 s | 7,200 s (configurable via `SimulationConfig.fixed_dt`) |
| TimeSpeed applied to | `dt * TimeSpeed` in `Update()` | `fixed_dt * time_speed` in system |
| Substep threshold | 28,800 s | 28,800 s (same constant) |
| Time reversal | Negative TimeSpeed | Same: negative `time_speed` |
| Threading | Dedicated goroutine | Bevy `FixedUpdate` schedule (main thread or `ComputeTaskPool`) |

### RK4 Scratch in Rust

The existing Rust `RK4Scratch` already pre-allocates:

```rust
struct RK4Scratch {
    pos0: Vec<Vec3>, vel0: Vec<Vec3>,
    k1p: Vec<Vec3>,  k1v: Vec<Vec3>,
    pos2: Vec<Vec3>,  vel2: Vec<Vec3>,
    k2p: Vec<Vec3>,  k2v: Vec<Vec3>,
    pos3: Vec<Vec3>,  vel3: Vec<Vec3>,
    k3p: Vec<Vec3>,  k3v: Vec<Vec3>,
    pos4: Vec<Vec3>,  vel4: Vec<Vec3>,
    k4p: Vec<Vec3>,  k4v: Vec<Vec3>,
}
```

**Source:** `crates/physics_core/src/sim.rs:24-41`

**Differences from Go:**
- Rust has 16 Vec arrays (Go has 16 + 1 `snapshot` array)
- Rust uses `Vec<Vec3>` with `copy_from_slice` (no reallocation)
- Rust takes scratch out of `Option` to avoid borrow conflicts: `self.scratch.take()`
- Growth: `ensure_size()` re-creates all arrays if too small

**Target Verlet scratch:** Add a `VerletScratch` struct to avoid per-step allocation:

```rust
struct VerletScratch {
    accel: Vec<Vec3>,         // current accelerations
    half_vel: Vec<Vec3>,      // half-step velocities
    snapshot_pos: Vec<Vec3>,  // position snapshot for acceleration
}

impl VerletScratch {
    fn new(n: usize) -> Self {
        let z = Vec3::default();
        Self {
            accel: vec![z; n],
            half_vel: vec![z; n],
            snapshot_pos: vec![z; n],
        }
    }

    fn ensure_size(&mut self, n: usize) {
        if self.accel.len() >= n { return; }
        *self = Self::new(n);
    }
}
```

### Trail Management in Bevy

Go uses `[]Vec3` with `append` + slice truncation. The target uses `VecDeque<DVec3>`:

```rust
// In solar_sim_core/src/trail.rs

pub struct TrailManager {
    pub trails: Vec<VecDeque<DVec3>>,
    pub max_len: usize,
}

impl TrailManager {
    pub fn new(n_bodies: usize, max_len: usize) -> Self {
        Self {
            trails: (0..n_bodies)
                .map(|_| VecDeque::with_capacity(max_len))
                .collect(),
            max_len,
        }
    }

    pub fn push(&mut self, body_index: usize, position: DVec3) {
        let trail = &mut self.trails[body_index];
        if trail.len() >= self.max_len {
            trail.pop_front();  // O(1), unlike Go's slice[1:] which is O(1) amortized but leaks
        }
        trail.push_back(position);
    }

    pub fn clear_all(&mut self) {
        for trail in &mut self.trails {
            trail.clear();
        }
    }

    pub fn resize(&mut self, n_bodies: usize) {
        while self.trails.len() < n_bodies {
            self.trails.push(VecDeque::with_capacity(self.max_len));
        }
        self.trails.truncate(n_bodies);
    }
}
```

**Advantages over Go:**
- `VecDeque::pop_front()` is O(1) with no memory leak (Go's `Trail[1:]` keeps the backing array)
- `VecDeque::with_capacity()` pre-allocates once
- Clearing resets length without deallocating capacity
- No deep copy needed for snapshots -- Bevy reads trail data directly from the resource

### ECS Sync

After the physics step, positions are synced to ECS components:

```rust
fn sync_ecs_from_simulation(
    sim: Res<SimulationState>,
    mut query: Query<(&CelestialBody, &mut Transform, &mut Velocity)>,
) {
    for (body, mut transform, mut velocity) in &mut query {
        if body.sim_index >= sim.inner.n_bodies {
            continue;
        }
        let pos = sim.inner.positions[body.sim_index];
        transform.translation = physics_to_render(pos);
        velocity.0 = DVec3::new(
            sim.inner.velocities[body.sim_index].x,
            sim.inner.velocities[body.sim_index].y,
            sim.inner.velocities[body.sim_index].z,
        );
    }
}
```

**Key difference from Go:** No snapshot copy needed. Bevy's system ordering guarantees that `sync_ecs_from_simulation` runs after `step_simulation` in the same `FixedUpdate` tick. No lock contention because Bevy systems have exclusive access through resource mutability.

### Trail Sync to Polylines

```rust
fn manage_trails(
    config: Res<SimulationConfig>,
    sim: Res<SimulationState>,
    mut query: Query<(&CelestialBody, &mut Orbit)>,
) {
    if !config.show_trails {
        return;
    }

    for (body, mut orbit) in &mut query {
        if !orbit.show_trail || body.sim_index >= sim.inner.n_bodies {
            continue;
        }
        let pos = sim.inner.positions[body.sim_index];
        let dvec = DVec3::new(pos.x, pos.y, pos.z);

        if orbit.trail_history.len() >= orbit.max_trail_len {
            orbit.trail_history.pop_front();
        }
        orbit.trail_history.push_back(dvec);
    }
}
```

### Physics<->Bevy Interface

The physics library communicates with Bevy through **direct method calls**, not channels:

```
Bevy FixedUpdate systems (main thread or compute thread)
    |
    +-> sim.inner.step(dt)           // direct call into solar_sim_core
    +-> sim.inner.positions[i]       // direct field read
    +-> sim.inner.trail_manager      // direct access
    |
    No channels, no atomic pointers, no locks
```

This is simpler than the Go architecture because Bevy's ECS system scheduling provides the synchronization guarantees that Go achieves through goroutines + mutexes + atomic pointers.

---

## Migration Path for Timestep Logic

### Phase A: In solar_sim_core

Add to `Simulation`:
1. `current_time: f64` field
2. `integrator: IntegratorType` field
3. `update(dt: f64, time_speed: f64)` method implementing substep logic
4. `step()` dispatches to RK4 or Verlet based on `integrator`
5. `TrailManager` for optional trail management in core

### Phase B: In solar_sim_bevy

The `step_simulation` system calls `sim.inner.update(config.fixed_dt, config.time_speed)`, which handles substep logic internally. The Bevy system only needs to check `config.is_playing` and call the update method.

Alternatively, the substep logic can live in the Bevy system (calling `sim.inner.step()` in a loop), keeping `solar_sim_core` simpler. This is the recommended approach since `is_playing` and `time_speed` are UI concerns:

```rust
// Recommended: substep logic in Bevy system
fn step_simulation(config: Res<SimulationConfig>, mut sim: ResMut<SimulationState>) {
    if !config.is_playing { return; }
    let eff_dt = config.fixed_dt * config.time_speed;
    let abs_dt = eff_dt.abs();
    if abs_dt <= MAX_SAFE_DT {
        sim.inner.step(eff_dt);
    } else {
        let n = (abs_dt / MAX_SAFE_DT).ceil() as usize;
        let sub = eff_dt / n as f64;
        for _ in 0..n { sim.inner.step(sub); }
    }
}
```

The validation harness calls `sim.step(dt)` directly (no substep, no time speed), keeping tests simple and deterministic.
