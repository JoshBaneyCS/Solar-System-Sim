# Compute Offload Candidates

Evaluation of each compute-intensive workload for GPU offloading via wgpu compute shaders in the Bevy architecture.

---

## Workload Summary Table

| Workload | Current Impl | Bodies/Particles | Ops/Frame | Est. CPU Time | GPU Compute Worth It? | Bevy Integration |
|----------|-------------|-----------------|-----------|---------------|-----------------------|------------------|
| N-body gravity | Go O(n^2) + goroutines (`simulator.go:272-292`) | ~20 | ~48K FLOPs (4 RK4 evals x 400 pairs x 30 FLOPs) | ~2-5 us | **No** -- GPU dispatch overhead ~100 us > compute time | `solar_sim_core::step()` in `FixedUpdate` system |
| Belt Kepler solving | Go per-particle Kepler (`belt.go:113-147`) | 1,500 | ~67.5K FLOPs (1,500 x 15 trig + 30 arith) | ~100-200 us | **No** at 1,500; **Yes** at 10,000+ | `par_iter_mut()` system updating `Transform` |
| Trail interpolation | Go Catmull-Rom + Bresenham (`trail_buffer.go:26-101`) | ~200 segments x ~20 bodies | ~128K FLOPs (4K segments x 4 sub-segs x ~8 FLOPs) | ~50-100 us | **No** -- output is polyline vertices, not parallel-friendly | `bevy_polyline` with CPU-side point generation |
| Spacetime potential | Go grid computation (`spacetime.go:90-173`) | 80x80 grid, ~20 bodies | ~384K FLOPs (6,400 points x ~20 bodies x 3 ops) | ~200-400 us | **Marginal** at 80x80; **Yes** at 200x200+ | CPU system at default; optional compute shader for high-res |
| Lighting/shading | Go Lambertian (`lighting.go:30-143`) | Per-pixel on planet textures | Variable (100px planet: ~31K pixels) | ~50-200 us | **N/A** -- Bevy PBR handles this automatically | `PointLight` + `StandardMaterial` (zero custom code) |

---

## Detailed Analysis Per Workload

### 1. N-Body Gravity

**Current code:** `internal/physics/simulator.go:220-292`

**Computation breakdown per physics tick:**

```
RK4 integrator: 4 acceleration evaluations per step
Each evaluation: n bodies, each computing force from n-1 others + Sun
Force computation: vector subtract (3 ops), dot product (5 ops), sqrt (1 op),
                   normalize (4 ops), G*M/r^2 (3 ops), scale (3 ops) = ~19 FLOPs
GR correction (Sun only): ~30 FLOPs additional per body
Total per evaluation: n * (n-1) * 19 + n * 30
```

| n | FLOPs per RK4 step | CPU time (est.) | GPU dispatch overhead |
|---|-------------------|-----------------|----------------------|
| 9 | 4 x (72 x 19 + 270) = 6,552 | ~1 us | ~100 us |
| 20 | 4 x (380 x 19 + 600) = 31,280 | ~3 us | ~100 us |
| 27 | 4 x (702 x 19 + 810) = 56,508 | ~5 us | ~100 us |
| 1,000 | 4 x (999K x 19 + 30K) = 76M | ~2 ms | ~100 us |

**GPU compute shader design (for reference, not recommended at current scale):**

```wgsl
// Storage buffers
@group(0) @binding(0) var<storage, read> bodies: array<Body>;     // position (vec3), mass (f32)
@group(0) @binding(1) var<storage, read_write> accels: array<vec4<f32>>;  // output accelerations
@group(0) @binding(2) var<uniform> params: SimParams;              // n_bodies, sun_mass, sun_pos, dt, flags

struct Body {
    position: vec3<f32>,
    mass: f32,
};

@compute @workgroup_size(64)
fn compute_gravity(@builtin(global_invocation_id) id: vec3<u32>) {
    let i = id.x;
    if (i >= params.n_bodies) { return; }

    var accel = vec3<f32>(0.0);
    let pos_i = bodies[i].position;

    // Sun gravity
    let r_sun = params.sun_pos - pos_i;
    let dist_sun = length(r_sun);
    if (dist_sun > 1e-6) {
        accel += normalize(r_sun) * (params.G * params.sun_mass / (dist_sun * dist_sun));
    }

    // Planet-planet gravity
    for (var j = 0u; j < params.n_bodies; j++) {
        if (j == i) { continue; }
        let r = bodies[j].position - pos_i;
        let dist = length(r);
        if (dist > 1e-6) {
            accel += normalize(r) * (params.G * bodies[j].mass / (dist * dist));
        }
    }

    accels[i] = vec4<f32>(accel, 0.0);
}
```

**Data layout:**
- Input buffer: `n * 16` bytes (vec3 position + f32 mass per body)
- Output buffer: `n * 16` bytes (vec4 acceleration per body)
- Uniform: 48 bytes (sun position, mass, G constant, n_bodies, flags)
- Bind group: 3 bindings

**Bevy integration:** Not recommended. If ever needed:
- Custom system in `RenderApp`'s `ExtractSchedule` copies body state to GPU buffer
- Compute pass dispatches `ceil(n/64)` workgroups
- Results read back via `BufferSlice::map_async` in `PrepareSchedule` of next frame (1-frame latency)
- Latency acceptable for rendering but problematic for physics accuracy (stale accelerations)

**Verdict: CPU only.** The 1-frame latency from GPU readback would degrade physics accuracy. CPU compute is faster than GPU overhead at n < 500.

---

### 2. Belt Particle Kepler Solving

**Current code:** `internal/render/belt.go:113-147`

**Computation breakdown per frame:**

```
Per particle (beltParticlePosition):
  sqrt(GM/a^3)           -- 1 sqrt, 2 multiply, 1 divide
  fmod(M, 2*pi)          -- 1 fmod
  5x Newton-Raphson:
    sin(E), cos(E)       -- 2 trig per iteration = 10 trig
    2 subtract, 1 divide -- 3 FLOPs per iteration = 15 FLOPs
  sqrt(1+e), sqrt(1-e)   -- 2 sqrt
  sin(E/2), cos(E/2)     -- 2 trig
  atan2                   -- 1 trig
  cos(nu), sin(nu)        -- 2 trig (implicit in atan2 result reuse)
  cos(E)                  -- 1 trig (radial distance)
  cos(nu), sin(nu)        -- 2 trig (orbital plane)
  cos(inc), sin(inc)      -- 2 trig (inclination)
  Arithmetic              -- ~15 multiply/add operations

Total: ~15 trig + ~4 sqrt + ~30 arithmetic = ~49 "heavy" operations per particle
```

| Particles | Trig Calls | Est. CPU Time (1 core) | Est. CPU Time (rayon, 8 cores) | Est. GPU Time |
|-----------|-----------|----------------------|-------------------------------|---------------|
| 1,500 | 22,500 | ~150 us | ~25 us | ~1 us compute + ~100 us overhead |
| 5,000 | 75,000 | ~500 us | ~80 us | ~2 us compute + ~100 us overhead |
| 10,000 | 150,000 | ~1 ms | ~150 us | ~5 us compute + ~100 us overhead |
| 50,000 | 750,000 | ~5 ms | ~700 us | ~20 us compute + ~100 us overhead |

**GPU compute shader design:**

```wgsl
struct BeltParticle {
    semi_major_axis: f32,  // in AU
    eccentricity: f32,
    inclination: f32,
    initial_anomaly: f32,
};

@group(0) @binding(0) var<storage, read> particles: array<BeltParticle>;
@group(0) @binding(1) var<storage, read_write> positions: array<vec4<f32>>;
@group(0) @binding(2) var<uniform> params: BeltParams;  // sim_time, GM, AU

@compute @workgroup_size(256)
fn solve_kepler(@builtin(global_invocation_id) id: vec3<u32>) {
    let i = id.x;
    if (i >= params.n_particles) { return; }

    let p = particles[i];
    let a = p.semi_major_axis * params.AU;
    let e = p.eccentricity;

    // Mean motion and anomaly
    let n = sqrt(params.GM / (a * a * a));
    var M = p.initial_anomaly + n * params.sim_time;
    M = M % 6.283185;

    // Newton-Raphson Kepler solver (5 iterations)
    var E = M;
    for (var k = 0u; k < 5u; k++) {
        E = E - (E - e * sin(E) - M) / (1.0 - e * cos(E));
    }

    // True anomaly
    let nu = 2.0 * atan2(sqrt(1.0 + e) * sin(E / 2.0), sqrt(1.0 - e) * cos(E / 2.0));

    // Radial distance and position
    let r = a * (1.0 - e * cos(E));
    let x_orb = r * cos(nu);
    let y_orb = r * sin(nu);

    // Apply inclination
    let x = x_orb;
    let y = y_orb * cos(p.inclination);
    let z = y_orb * sin(p.inclination);

    positions[i] = vec4<f32>(x, y, z, 0.0);
}
```

**Data layout:**
- Input buffer: `n * 16` bytes (4 x f32 per particle orbital elements)
- Output buffer: `n * 16` bytes (vec4 position per particle)
- Uniform: 16 bytes (sim_time, GM, AU, n_particles)
- Dispatch: `ceil(n/256)` workgroups

**Bevy integration (GPU path):**

The output buffer can serve directly as the instance transform buffer for rendering, eliminating CPU-GPU data transfer:

```rust
// In RenderApp's Prepare phase:
fn prepare_belt_compute(
    mut commands: Commands,
    pipeline: Res<BeltComputePipeline>,
    device: Res<RenderDevice>,
    queue: Res<RenderQueue>,
    belt_data: Res<ExtractedBeltData>,
) {
    // Upload particle orbital elements (once, or when particles change)
    queue.write_buffer(&pipeline.particle_buffer, 0, &belt_data.elements);

    // Upload time uniform (every frame)
    queue.write_buffer(&pipeline.params_buffer, 0, bytemuck::bytes_of(&belt_data.params));

    // Dispatch compute -- output goes directly to instance buffer
    let mut encoder = device.create_command_encoder(&Default::default());
    {
        let mut pass = encoder.begin_compute_pass(&Default::default());
        pass.set_pipeline(&pipeline.pipeline);
        pass.set_bind_group(0, &pipeline.bind_group, &[]);
        pass.dispatch_workgroups((belt_data.n_particles + 255) / 256, 1, 1);
    }
    queue.submit(std::iter::once(encoder.finish()));
}
```

**Bevy integration (CPU path, recommended at current scale):**

```rust
fn update_belt_positions(
    time: Res<SimulationTime>,
    mut query: Query<(&BeltParticle, &mut Transform)>,
) {
    query.par_iter_mut().for_each(|(particle, mut transform)| {
        let pos = kepler_solve(particle, time.current);
        transform.translation = Vec3::new(pos.x as f32, pos.y as f32, pos.z as f32);
    });
}
```

**Verdict: CPU parallel at 1,500 particles. Prepare GPU compute path for future scaling.**

The rayon-based `par_iter_mut()` handles 1,500 particles in ~25 us. GPU compute is not justified until 10,000+ particles. However, the compute shader design above is clean and could be added in a future phase if the belt expands (e.g., Kuiper belt visualization).

---

### 3. Trail Interpolation

**Current code:** `internal/render/trail_buffer.go:26-101`

**Computation breakdown per frame:**

```
Per body with trail:
  Max 200 segments (downsampled from up to 500 trail points)
  Each segment: 4 Catmull-Rom sub-segments
  Each sub-segment: CatmullRom interpolation (6 multiply + 4 add per component x 3 = 30 FLOPs)
                    + WorldToScreen transform (~20 FLOPs)
                    + Bresenham line drawing (~10-50 pixel writes per sub-segment)

For ~20 bodies with trails:
  20 bodies x 200 segments x 4 sub-segs = 16,000 interpolation points
  16,000 x (30 FLOPs interpolation + 20 FLOPs projection) = ~800K FLOPs
  Line drawing: ~16,000 segments x ~20 pixels avg = ~320K pixel operations
```

| Bodies with Trails | Segments (max) | Sub-segments | FLOPs | Est. CPU Time |
|-------------------|----------------|-------------|-------|---------------|
| 9 (planets only) | 1,800 | 7,200 | ~360K | ~30 us |
| 20 (+ moons) | 4,000 | 16,000 | ~800K | ~70 us |
| 27 (+ comets + asteroids) | 5,400 | 21,600 | ~1.1M | ~90 us |

**Why GPU is not a good fit:**

1. **Output is line geometry, not a buffer.** The trail computation produces polyline vertex positions, which Bevy's `bevy_polyline` renders as GPU line primitives. The computation is generating the vertex data, not rasterizing it.

2. **Variable-length trails.** Each body has a different trail length. GPU workload balancing is inefficient with highly variable per-thread work.

3. **Catmull-Rom requires sequential access.** Each interpolation point needs 4 neighboring trail points (P0, P1, P2, P3). While not strictly sequential, the data access pattern is non-uniform and cache-unfriendly on GPU.

4. **WorldToScreen requires camera state.** The current code projects each point; in Bevy, the trails exist in world space and the camera handles projection automatically.

**Bevy integration:**

```rust
fn update_trail_polylines(
    sim: Res<SimulationState>,
    mut polylines: Query<(&TrailOwner, &mut Polyline)>,
) {
    for (owner, mut polyline) in &mut polylines {
        let trail = &sim.inner.get_trail(owner.body_index);
        // Downsample to max 200 points
        let step = (trail.len() / 200).max(1);
        let points: Vec<Vec3> = trail.iter()
            .step_by(step)
            .map(|p| Vec3::new(p.x as f32, p.y as f32, p.z as f32))
            .collect();
        polyline.vertices = points;
    }
}
```

Note: In Bevy, Catmull-Rom smoothing can be applied on the CPU when generating polyline vertices. The smoothing improves visual quality but adds only ~30 FLOPs per output point -- negligible. Alternatively, `bevy_polyline` may support spline modes natively.

**Verdict: CPU only.** Trail interpolation is fast (~70 us for 20 bodies), produces variable-length geometry data, and integrates naturally as a Bevy CPU system feeding polyline vertex data. No GPU compute benefit.

---

### 4. Spacetime Potential Field

**Current code:** `internal/spacetime/spacetime.go:90-173`

**Computation breakdown per frame (when spacetime enabled):**

```
Grid: gridResolution x gridResolution points (default 80x80 = 6,400)
Per grid point:
  Sun contribution: 2 subtract, 2 multiply, 1 add, 1 sqrt, 1 multiply, 1 divide = ~8 FLOPs
  Per planet: 2 subtract, 2 multiply, 1 add, 1 sqrt, 1 compare, 1 multiply, 1 divide = ~9 FLOPs
  Log normalization: 1 multiply, 1 log1p = ~5 FLOPs

Total per grid point: 8 + 20 * 9 + 5 = ~193 FLOPs (with 20 bodies)
Total per frame: 6,400 * 193 = ~1.24M FLOPs
```

| Grid Resolution | Grid Points | FLOPs (20 bodies) | Est. CPU Time | GPU Worth It? |
|----------------|------------|-------------------|---------------|---------------|
| 40x40 | 1,600 | ~309K | ~50 us | No |
| 80x80 | 6,400 | ~1.24M | ~200 us | Marginal |
| 120x120 | 14,400 | ~2.78M | ~400 us | Marginal |
| 200x200 | 40,000 | ~7.72M | ~1.1 ms | **Yes** |
| 400x400 | 160,000 | ~30.9M | ~4.4 ms | **Yes** |

**Caching mitigates the need:** The current implementation caches the potential field and only recomputes when zoom/pan change by >5% (`spacetime.go:72-87`). During static views, the computation happens zero times per frame. This makes GPU offloading less impactful than the raw compute cost suggests.

**GPU compute shader design:**

```wgsl
struct SpacetimeParams {
    grid_size: u32,
    n_bodies: u32,
    sun_mass: f32,
    sun_x: f32,
    sun_y: f32,
    G_over_c2: f32,        // precomputed G / (c^2)
    world_origin_x: f32,
    world_origin_y: f32,
    grid_spacing_x: f32,
    grid_spacing_y: f32,
    scale_factor: f32,
    _pad: f32,
};

struct BodyMassPos {
    x: f32,
    y: f32,
    mass: f32,
    _pad: f32,
};

@group(0) @binding(0) var<uniform> params: SpacetimeParams;
@group(0) @binding(1) var<storage, read> bodies: array<BodyMassPos>;
@group(0) @binding(2) var<storage, read_write> potentials: array<f32>;

@compute @workgroup_size(16, 16)
fn compute_potential(@builtin(global_invocation_id) id: vec3<u32>) {
    let i = id.x;
    let j = id.y;
    if (i >= params.grid_size || j >= params.grid_size) { return; }

    let world_x = params.world_origin_x + f32(i) * params.grid_spacing_x;
    let world_y = params.world_origin_y + f32(j) * params.grid_spacing_y;

    var curvature: f32 = 0.0;

    // Sun contribution
    let dx_sun = params.sun_x - world_x;
    let dy_sun = params.sun_y - world_y;
    let r_sun = sqrt(dx_sun * dx_sun + dy_sun * dy_sun);
    if (r_sun > 1e6) {
        curvature += 2.0 * params.G_over_c2 * params.sun_mass / r_sun;
    }

    // Planet contributions
    for (var b = 0u; b < params.n_bodies; b++) {
        let body = bodies[b];
        let dx = body.x - world_x;
        let dy = body.y - world_y;
        let r = sqrt(dx * dx + dy * dy);
        if (r > 1e6) {
            curvature += 2.0 * params.G_over_c2 * body.mass / r;
        }
    }

    // Log-scale normalization
    let idx = j * params.grid_size + i;
    potentials[idx] = log(1.0 + curvature * params.scale_factor);
}
```

**Data layout:**
- Uniform: 48 bytes (grid params)
- Body buffer: `n * 16` bytes (position + mass per body, ~320 bytes for 20 bodies)
- Output buffer: `grid_size^2 * 4` bytes (f32 per grid point)
  - 80x80: 25.6 KB
  - 200x200: 160 KB
- Dispatch: `ceil(grid_size/16) x ceil(grid_size/16)` workgroups

**Bevy integration (GPU compute path):**

```rust
// Custom render node in the render graph
fn spacetime_compute_node(world: &World) -> impl Node {
    // Only dispatch when cache is invalidated
    if !world.resource::<SpacetimeCache>().needs_update { return; }

    // Prepare buffers
    let params = world.resource::<SpacetimeParams>();
    queue.write_buffer(&self.params_buffer, 0, bytemuck::bytes_of(params));
    queue.write_buffer(&self.body_buffer, 0, bytemuck::cast_slice(&params.bodies));

    // Dispatch compute
    let mut pass = encoder.begin_compute_pass(&Default::default());
    pass.set_pipeline(&self.pipeline);
    pass.set_bind_group(0, &self.bind_group, &[]);
    let wg = (params.grid_size + 15) / 16;
    pass.dispatch_workgroups(wg, wg, 1);

    // Output buffer feeds directly into the mesh generation system
    // (no CPU readback needed if grid mesh is also GPU-generated)
}
```

However, the output must be used to generate warped grid mesh vertex positions, which typically happens on the CPU. Unless the grid mesh generation is also GPU-accelerated, a readback is required, adding ~50-100 us overhead.

**Bevy integration (CPU path, recommended):**

```rust
fn update_spacetime_grid(
    sim: Res<SimulationState>,
    camera: Query<&Transform, With<Camera3d>>,
    mut grid: ResMut<SpacetimeGrid>,
    mut meshes: ResMut<Assets<Mesh>>,
) {
    if !grid.needs_update(&camera) { return; }

    // Compute potential field on CPU (same algorithm as spacetime.go)
    let potentials = compute_potential_field(&sim, &camera, grid.resolution);
    let normalized = normalize_potentials(&potentials);

    // Generate warped mesh vertices
    let mesh = generate_grid_mesh(&normalized, grid.resolution);
    meshes.insert(&grid.mesh_handle, mesh);

    grid.mark_updated(&camera);
}
```

**Verdict: CPU for default (80x80 grid). Optional GPU compute for high-resolution grids (200x200+).**

The caching strategy means this computation runs infrequently (only when camera moves). At 80x80, CPU completes in ~200 us -- not worth the complexity of a GPU compute pipeline. At 200x200+, GPU compute becomes worthwhile, but this is an enhancement, not a requirement for the initial migration.

---

### 5. Lighting and Shading

**Current code:** `internal/render/lighting.go:30-143`

**Computation:** Per-pixel Lambertian shading on circular planet texture cutouts. For a 100px diameter planet: pi * 50^2 = ~7,854 pixels. Each pixel: sphere normal computation (3 sqrt ops), dot product (3 multiply + 2 add), intensity clamp = ~15 FLOPs.

| Planet Diameter (px) | Pixels | FLOPs | CPU Time |
|---------------------|--------|-------|----------|
| 20 | ~314 | ~4.7K | <1 us |
| 50 | ~1,963 | ~29K | ~5 us |
| 100 | ~7,854 | ~118K | ~20 us |
| 200 | ~31,416 | ~471K | ~80 us |

**Current parallelization:** Images > 100px tall are split across `runtime.NumCPU()` goroutines (`lighting.go:72`).

**Bevy integration:**

This workload is **entirely eliminated** by Bevy's PBR pipeline. There is no custom lighting code to write or optimize:

```rust
// Planet entity setup (done once at initialization)
commands.spawn((
    Mesh3d(meshes.add(Sphere::new(1.0).mesh().uv(32, 18))),
    MeshMaterial3d(materials.add(StandardMaterial {
        base_color_texture: Some(asset_server.load("textures/earth/albedo.jpg")),
        ..default()
    })),
    Transform::from_translation(earth_position),
    CelestialBody { name: "Earth".into(), .. },
));

// Sun light (done once)
commands.spawn((
    PointLight {
        intensity: 1_000_000.0,
        shadows_enabled: true,
        ..default()
    },
    Transform::from_translation(Vec3::ZERO),
));
```

Bevy's fragment shader handles Lambertian diffuse, specular highlights, shadow sampling, and ambient occlusion automatically. The GPU executes this as part of the standard rasterization pipeline -- no separate compute pass needed.

**Verdict: Not applicable. Bevy PBR replaces this entirely with zero custom code.**

---

## GPU Compute Decision Summary

| Workload | GPU Offload? | Threshold | Current Scale | Bevy Strategy |
|----------|-------------|-----------|---------------|---------------|
| N-body gravity | **No** | ~500+ bodies | ~20 bodies | CPU (`solar_sim_core`) |
| Belt Kepler solving | **No** (prepare for future) | ~10,000+ particles | 1,500 particles | CPU `par_iter_mut()` |
| Trail interpolation | **No** | Never (wrong workload type) | ~20 bodies x 200 segs | CPU system -> `bevy_polyline` |
| Spacetime potential | **Optional** | ~200x200+ grid | 80x80 grid (cached) | CPU default; optional compute shader |
| Lighting/shading | **N/A** | N/A | N/A | Bevy PBR (automatic) |

### When GPU Compute Becomes Worthwhile

The project currently operates well below the thresholds where GPU compute provides benefit. GPU compute should be considered if:

1. **Belt particle count increases to 10,000+** (e.g., Kuiper belt visualization, detailed asteroid belt)
2. **Spacetime grid resolution increases to 200x200+** (e.g., high-DPI displays, detailed curvature visualization)
3. **Body count increases to 500+** (e.g., N-body swarm simulations, galaxy merger visualization)

None of these are planned for the initial Bevy migration. The recommended approach is:

1. **Phase B-C (initial migration):** Pure CPU computation for all workloads. Use `rayon` parallelism in `solar_sim_core` and Bevy `par_iter_mut()` for belt particles.
2. **Phase D+ (optimization):** If profiling reveals CPU bottlenecks at higher workload scales, add targeted wgpu compute shaders using the designs documented above. The Bevy render graph makes it straightforward to insert compute passes without restructuring the application.

### Bevy 0.15 Compute Shader API Reference

For future GPU compute work, Bevy 0.15 provides:

- **`RenderDevice`** -- wraps `wgpu::Device`, used to create buffers, bind groups, and pipelines
- **`RenderQueue`** -- wraps `wgpu::Queue`, used for buffer writes and command submission
- **`SpecializedComputePipeline`** -- trait for pipelines with compile-time specialization
- **`ComputePipelineDescriptor`** -- configures compute pipeline (shader, entry point, layout)
- **`BindGroupLayout` / `BindGroup`** -- resource binding configuration
- **`StorageBuffer<T>`** / `UniformBuffer<T>`** -- typed GPU buffer wrappers with automatic size management
- **Render Graph nodes** -- custom `Node` implementations inserted into the render graph for ordered execution
- **`ExtractSchedule`** -- system schedule for copying data from main world to render world (runs in parallel with next frame's game logic)

A compute shader integration follows this pattern:

```rust
// 1. Define pipeline as a resource
#[derive(Resource)]
struct MyComputePipeline {
    pipeline: CachedComputePipelineId,
    bind_group_layout: BindGroupLayout,
}

// 2. Initialize in Plugin::build() for RenderApp
impl FromWorld for MyComputePipeline {
    fn from_world(world: &mut World) -> Self {
        let device = world.resource::<RenderDevice>();
        // Create bind group layout, shader module, pipeline...
    }
}

// 3. Extract data from main world
fn extract_compute_data(mut commands: Commands, data: Extract<Res<MyData>>) {
    commands.insert_resource(ExtractedData(data.clone()));
}

// 4. Implement Node for render graph
impl Node for MyComputeNode {
    fn run(&self, _graph: &mut RenderGraphContext, render_context: &mut RenderContext, world: &World) -> Result<(), NodeRunError> {
        let pipeline = world.resource::<MyComputePipeline>();
        let mut pass = render_context.command_encoder().begin_compute_pass(&Default::default());
        pass.set_pipeline(&pipeline.get_pipeline());
        pass.set_bind_group(0, &pipeline.bind_group, &[]);
        pass.dispatch_workgroups(wg_x, wg_y, 1);
        Ok(())
    }
}
```
