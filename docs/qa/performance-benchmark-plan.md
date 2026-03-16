# Performance Benchmark Plan

Benchmark definitions, Go baselines, Rust targets, and CI integration plan for the migration.

---

## Benchmark Definitions

### Physics Benchmarks

| # | Benchmark | Metric | Go Baseline | Rust Target | Tool | Phase |
|---|-----------|--------|------------|------------|------|-------|
| P1 | Physics step time (9 planets, RK4, N-body + GR) | us/step | Measure via `BenchmarkStep` (`internal/physics/benchmark_test.go:9-17`). Config: `PlanetGravityEnabled=true`, `RelativisticEffects=true`, dt=7200s. | <= Go baseline (expect 2-5x faster due to Rust numeric perf). Target: < 50% of Go time. | Go: `go test -bench BenchmarkStep -benchmem ./internal/physics/`. Rust: `criterion` crate with identical config. | A |
| P2 | Physics step time (9 planets, Newtonian only) | us/step | Measure via `BenchmarkStepNewtonianOnly` (`benchmark_test.go:19-27`). Config: `PlanetGravityEnabled=false`, `RelativisticEffects=false`. | <= Go baseline. | Same tools. | A |
| P3 | Physics step time (9 planets, N-body, no GR) | us/step | Measure via `BenchmarkStepNBody` (`benchmark_test.go:29-37`). Config: `PlanetGravityEnabled=true`, `RelativisticEffects=false`. | <= Go baseline. | Same tools. | A |
| P4 | Physics step time (9 planets + 8 moons, RK4) | us/step | Not currently benchmarked. Must add: create sim with `AddMoons()`, step with N-body + GR. Expected: ~2-3x P1 due to 17 bodies vs 9 (O(n^2) gravity). | <= Go baseline. | Add new Go benchmark `BenchmarkStepWithMoons`. Rust: criterion. | A |
| P5 | Physics step time (27 bodies, RK4) | us/step | Not currently benchmarked. Must add: create sim with moons + comets + asteroids (27 bodies). | <= Go baseline. Rayon parallel threshold (16) should kick in. | Add new Go benchmark `BenchmarkStepAllBodies`. Rust: criterion. | A |
| P6 | Physics step time (9 planets, Verlet) | us/step | Not currently benchmarked. Must add: Verlet integrator, same config as P1 minus GR. Expected: faster than RK4 (1 accel eval vs 4). | <= Go baseline. | Add new Go benchmark `BenchmarkStepVerlet`. Rust: criterion. | A |
| P7 | Single acceleration computation | ns/call | Measure via `BenchmarkCalculateAcceleration` (`benchmark_test.go:39-52`). Single body, with N-body + GR. | <= Go baseline. | Same tools. | A |
| P8 | GR correction computation | ns/call | Not directly benchmarked. Estimated: ~100ns (a few multiplies and adds on f64). Must add: benchmark calling `calculate_gr_correction` in isolation. | <= Go baseline. Expect < 50ns in Rust. | Go: new benchmark in `gr/`. Rust: criterion on `gr::calculate_gr_correction`. | A |

### Rendering Benchmarks

| # | Benchmark | Metric | Go Baseline | Rust Target | Tool | Phase |
|---|-----------|--------|------------|------------|------|-------|
| P9 | Frame time (full scene: planets + trails + belt + labels) | ms/frame | Not benchmarked. Measure: run Go app with all features enabled, capture FPS via status bar. Expected: 16-33ms (30-60 FPS) depending on GPU mode. | <= 16ms (60 FPS) on discrete GPU. <= 33ms (30 FPS) on integrated GPU. | Go: manual FPS capture from status bar. Bevy: `FrameTimeDiagnosticsPlugin` + `LogDiagnosticsPlugin`. | C |
| P10 | Frame time (planets only, no belt/trails/spacetime) | ms/frame | Not benchmarked. Expected: < 5ms (trivially lightweight with 10 spheres). | <= 5ms. | Same tools as P9. | B |
| P11 | Belt particle update (1500 particles) | ms/frame | Not benchmarked. Go solves Kepler equation per particle per frame (5 Newton iterations each). Estimated: 1-3ms sequential. | <= Go baseline. Bevy can parallelize via `par_iter_mut`. Target: < 1ms. | Go: add benchmark in `render/belt_test.go`. Bevy: frame time delta with/without belt enabled. | C |
| P12 | Trail interpolation (200 segments x 9 planets) | ms/frame | Not benchmarked. Go uses Catmull-Rom + Bresenham per trail per frame. Estimated: < 1ms. | <= 1ms. `bevy_polyline` or Gizmos handle this on GPU. | Measure frame time delta with/without trails. | C |

### System Benchmarks

| # | Benchmark | Metric | Go Baseline | Rust Target | Tool | Phase |
|---|-----------|--------|------------|------------|------|-------|
| P13 | Startup time (launch to first frame) | seconds | Not benchmarked. Estimated: 2-5s (Fyne window init, texture loading, physics init). | <= 3s. Bevy asset loading is async; first frame may appear before all textures load. | Go: `time` command + app instrumentation. Bevy: timestamp from `main()` to first `Update` run. | B |
| P14 | Memory usage (RSS after 1 minute running) | MB | Not benchmarked. Estimated: 50-150 MB (depends on texture resolution, GPU mode). | <= Go baseline. Bevy's ECS is memory-efficient. Target: < 200 MB. | Go: `ps -o rss` or Activity Monitor. Bevy: `/proc/self/status` or equivalent. | B |
| P15 | Texture loading time | seconds | Not benchmarked. Go loads textures lazily on first render. Estimated: 0.5-2s for all planet textures. | <= 2s. Bevy `AssetServer` loads asynchronously. | Instrument loading code with timestamps. | B |
| P16 | Binary size | MB | Not measured. Go binary with Fyne is typically 20-40 MB. | < 100 MB with embedded assets (per Phase D criteria). Without assets: < 30 MB. | `ls -lh` on release binary. | D |

---

## How to Collect Go Baselines

### Existing Benchmarks

Run the existing Go benchmarks to establish baselines:

```bash
cd /Users/joshbaney/GolandProjects/solar-system-simulator

# Physics step benchmarks (P1, P2, P3, P7)
go test -bench=. -benchmem -count=5 -benchtime=5s ./internal/physics/ | tee docs/qa/go-benchmark-baseline.txt

# Run with CPU profile for detailed analysis
go test -bench=BenchmarkStep -cpuprofile=cpu.prof -benchtime=10s ./internal/physics/
go tool pprof cpu.prof
```

### Benchmarks to Add (Go side)

Before migration, add these benchmarks to establish complete baselines:

```go
// internal/physics/benchmark_test.go -- add these:

func BenchmarkStepWithMoons(b *testing.B) {
    sim := NewSimulator()
    sim.PlanetGravityEnabled = true
    sim.RelativisticEffects = true
    sim.AddMoons()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        sim.Step(constants.BaseTimeStep)
    }
}

func BenchmarkStepAllBodies(b *testing.B) {
    sim := NewSimulator()
    sim.PlanetGravityEnabled = true
    sim.RelativisticEffects = true
    sim.AddMoons()
    sim.AddComets()
    sim.AddAsteroids()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        sim.Step(constants.BaseTimeStep)
    }
}

func BenchmarkStepVerlet(b *testing.B) {
    sim := NewSimulator()
    sim.PlanetGravityEnabled = true
    sim.RelativisticEffects = false
    sim.Integrator = IntegratorVerlet
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        sim.Step(constants.BaseTimeStep)
    }
}
```

### Benchmarks to Add (Rust side)

Create `crates/physics_core/benches/physics_bench.rs`:

```rust
use criterion::{criterion_group, criterion_main, Criterion};
use physics_core::sim::Simulation;
// ... initialize from body catalog once available

fn bench_step_9_planets_rk4(c: &mut Criterion) {
    let mut sim = create_solar_system_sim(); // from catalog
    c.bench_function("step_9planets_rk4_nbody_gr", |b| {
        b.iter(|| sim.step(7200.0))
    });
}

fn bench_acceleration_single(c: &mut Criterion) {
    let sim = create_solar_system_sim();
    c.bench_function("acceleration_single_body", |b| {
        b.iter(|| {
            sim.calculate_acceleration(
                0,
                sim.positions[0],
                sim.velocities[0],
                &sim.positions,
            )
        })
    });
}

criterion_group!(benches, bench_step_9_planets_rk4, bench_acceleration_single);
criterion_main!(benches);
```

---

## CI Pipeline Specification

### On Every Pull Request

| Check | Command | Threshold | Fail Action |
|-------|---------|-----------|-------------|
| Rust unit tests | `cargo test -p physics_core` | All pass | Block merge |
| Rust validation: energy conservation | `cargo test -p physics_core validation::energy` | Drift < 1e-4 per year | Block merge |
| Rust validation: golden tests | `cargo test -p physics_core golden` | Position within 0.1m, velocity within 1e-6 m/s | Block merge |
| Rust validation: Kepler periods | `cargo test -p physics_core validation::kepler` | Within 1% of expected | Block merge |
| Go unit tests (regression guard) | `go test ./internal/physics/ ./internal/validation/` | All pass | Block merge |
| Clippy lint | `cargo clippy -p physics_core -- -D warnings` | Zero warnings | Block merge |
| Build check (all platforms) | `cargo check -p physics_core --target {x86_64-linux, x86_64-darwin, aarch64-darwin}` | Compiles | Block merge |

**Estimated CI time:** 2-3 minutes.

### Nightly Benchmarks

| Check | Command | Threshold | Fail Action |
|-------|---------|-----------|-------------|
| Rust physics step benchmark | `cargo bench -p physics_core -- step_9planets` | <= 2x stored baseline (see regression detection below) | Open issue, notify team |
| Rust acceleration benchmark | `cargo bench -p physics_core -- acceleration` | <= 2x stored baseline | Open issue |
| Go physics step benchmark | `go test -bench=BenchmarkStep -benchtime=5s ./internal/physics/` | <= 2x stored baseline | Open issue |
| Long-run validation: Mercury precession (10 years) | `cargo test -p physics_core -- mercury_precession --ignored` (mark as `#[ignore]` for CI) | 42-44 arcsec/century | Open issue |
| Long-run validation: energy (10 years) | `cargo test -p physics_core -- energy_10yr --ignored` | Drift < 1e-3 | Open issue |
| N-body stability (100 years) | `cargo test -p physics_core -- stability_100yr --ignored` | No NaN/Infinity | Open issue |
| Memory leak check (Bevy, Phase B+) | Run `solar_sim_bevy` for 5 minutes, check RSS growth | RSS growth < 50 MB over 5 minutes | Open issue |

**Estimated nightly CI time:** 10-15 minutes.

### Weekly (Phase C+)

| Check | Command | Threshold | Fail Action |
|-------|---------|-----------|-------------|
| Full visual regression suite | Capture screenshots at t=0, t=100, t=1000 steps. Compare SSIM against baseline. | SSIM > 0.85 for each screenshot pair | Open issue with diff images |
| Cross-platform build | Build on macOS (arm64, x86_64), Linux (x86_64), Windows (x86_64) | All compile and pass unit tests | Open issue |
| Binary size check | `ls -lh target/release/solar_sim_bevy` | < 100 MB (with assets) | Warning (not blocking) |

---

## Performance Regression Detection

### Baseline Storage

Store benchmark results in `docs/qa/baselines/`:

```
docs/qa/baselines/
  go-physics-step.json        # { "BenchmarkStep": { "ns_per_op": 12345, "date": "2026-03-16" } }
  rust-physics-step.json      # { "step_9planets_rk4": { "ns_per_iter": 6789, "date": "2026-03-16" } }
```

### Regression Thresholds

| Metric | Warning Threshold | Failure Threshold | Rationale |
|--------|-------------------|-------------------|-----------|
| Physics step time | > 1.3x baseline | > 2.0x baseline | 30% variance is normal for benchmarks; 2x indicates a real regression |
| Acceleration computation | > 1.3x baseline | > 2.0x baseline | Same rationale |
| Frame time | > 20ms (50 FPS) | > 33ms (30 FPS) | User-perceptible below 30 FPS |
| Startup time | > 5s | > 10s | User experience threshold |
| Memory RSS | > 200 MB | > 500 MB | Reasonable for a desktop app |

### Benchmark Comparison Script

Recommended approach for CI:

```bash
# Run Rust benchmarks and compare against baseline
cargo bench -p physics_core -- --save-baseline current
critcmp baseline current --threshold 30  # warn if >30% slower
```

Use [`critcmp`](https://github.com/BurntSushi/critcmp) for comparing criterion baselines across runs.

For Go, use [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat):

```bash
go test -bench=. -count=10 ./internal/physics/ > new.txt
benchstat docs/qa/baselines/go-old.txt new.txt
```

---

## Phase-Gated Performance Requirements

### Phase A: Physics Crate

| Requirement | Metric | Target |
|-------------|--------|--------|
| Rust step time | P1 | <= Go baseline |
| Rust acceleration time | P7 | <= Go baseline |
| All validation tests pass | -- | Same tolerances as Go |
| Cargo test completes | -- | < 30 seconds |

### Phase B: Bevy Window + Planets

| Requirement | Metric | Target |
|-------------|--------|--------|
| FPS (10 entities, no effects) | P10 | >= 60 FPS discrete GPU, >= 30 FPS integrated |
| Startup time | P13 | < 5 seconds |
| Memory | P14 | < 200 MB |

### Phase C: Visual Feature Parity

| Requirement | Metric | Target |
|-------------|--------|--------|
| FPS (full scene) | P9 | >= 60 FPS discrete GPU, >= 30 FPS integrated |
| Belt update | P11 | < 2 ms per frame |
| Trail rendering | P12 | < 1 ms per frame |

### Phase D: Full Feature Parity

| Requirement | Metric | Target |
|-------------|--------|--------|
| All Phase A-C requirements | -- | Still met |
| Binary size | P16 | < 100 MB with assets |
| Rust physics step | P1 | >= 2x faster than Go (Rust should shine on numeric code) |

---

## Existing Go Benchmark Functions (Reference)

Source: `internal/physics/benchmark_test.go`

| Function | Config | What It Measures |
|----------|--------|-----------------|
| `BenchmarkStep` | N-body=true, GR=true | Full physics step with all interactions |
| `BenchmarkStepNewtonianOnly` | N-body=false, GR=false | Sun-only gravity, simplest case |
| `BenchmarkStepNBody` | N-body=true, GR=false | Planet-planet gravity without GR |
| `BenchmarkCalculateAcceleration` | N-body=true, GR=true | Single body acceleration computation |

---

## Existing Rust Test Functions (Reference)

Source: `crates/physics_core/src/sim.rs` `#[cfg(test)]`

| Function | What It Tests |
|----------|--------------|
| `test_sun_only_gravity` | Acceleration magnitude matches GM/r^2 |
| `test_inverse_square_law` | 1 AU vs 2 AU gives 4:1 ratio |
| `test_step_changes_state` | Position changes after a step |
| `test_energy_conservation` | 1000 steps, single Earth body, drift < 1e-6 |
| `test_parallel_consistency` | 20 bodies, parallel execution produces valid results |

Source: `crates/physics_core/src/gr.rs` `#[cfg(test)]`

| Function | What It Tests |
|----------|--------------|
| `test_gr_correction_nonzero_for_mercury` | GR correction is non-zero at Mercury's orbit |
| `test_gr_correction_zero_velocity` | Direction is along position vector when v=0 |
| `test_gr_correction_formula_1pn` | Matches manual 1PN computation to 1e-15 |
