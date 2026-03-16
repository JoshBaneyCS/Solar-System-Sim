# GPU Acceleration Strategy

Decision matrix for each GPU workload in the Bevy migration. For each workload: current implementation, GPU compute feasibility, expected speedup, Bevy integration approach, and recommendation.

---

## 1. N-Body Gravity (~20 Bodies)

### Current Implementation

- **File:** `internal/physics/simulator.go:220-268` (`CalculateAccelerationWithSnapshot`)
- **Algorithm:** O(n^2) pairwise gravitational force + GR correction for each body against the Sun
- **Parallelization:** Go goroutines when `n >= parallelThreshold` (12 bodies) -- one goroutine per body (`simulator.go:277-292`)
- **Body count:** 9 planets baseline, up to ~27 with moons/comets/asteroids enabled
- **Per-step cost:** 4 acceleration evaluations per RK4 step, each computing n^2 force pairs

### FLOP Analysis

| Bodies (n) | Force Pairs | FLOPs/Pair | FLOPs/Eval | RK4 Evals/Step | FLOPs/Step |
|------------|-------------|------------|------------|----------------|------------|
| 9          | 81          | ~30        | ~2,430     | 4              | ~9,720     |
| 20         | 400         | ~30        | ~12,000    | 4              | ~48,000    |
| 27         | 729         | ~30        | ~21,870    | 4              | ~87,480    |
| 1,000      | 1,000,000   | ~30        | ~30M       | 4              | ~120M      |
| 10,000     | 100,000,000 | ~30        | ~3B        | 4              | ~12B       |

Each force pair requires: 3 subtractions (vector), 3 multiplications + 2 additions (dot product for distance), 1 sqrt, 1 division, 3 multiplies (normalize), 1 division (G*M/r^2), 3 multiplies (scale) = ~30 FLOPs.

### GPU Compute Feasibility

**GPU launch overhead:** A wgpu compute dispatch involves bind group creation, command buffer encoding, queue submission, and device synchronization. Typical overhead is 50-200 us per dispatch on modern hardware.

**Crossover analysis:**

| Bodies | CPU Time (est.) | GPU Compute Time (est.) | GPU Overhead | GPU Total | Winner |
|--------|-----------------|------------------------|--------------|-----------|--------|
| 9      | ~2 us           | <1 us                  | ~100 us      | ~100 us   | **CPU** |
| 20     | ~5 us           | <1 us                  | ~100 us      | ~100 us   | **CPU** |
| 100    | ~50 us          | ~2 us                  | ~100 us      | ~102 us   | **CPU** |
| 1,000  | ~2 ms           | ~10 us                 | ~100 us      | ~110 us   | **GPU** |
| 10,000 | ~200 ms         | ~100 us                | ~100 us      | ~200 us   | **GPU** |

**Crossover point: ~500-1,000 bodies.** Below this, GPU dispatch overhead dominates. The current simulator maxes out at ~27 bodies.

### Bevy Integration Approach (if GPU were needed)

A wgpu compute shader would use:
- Storage buffer: body positions/masses (`vec4<f32>` per body, xyz + mass)
- Storage buffer: output accelerations (`vec4<f32>` per body)
- Workgroup size: `@workgroup_size(64)`, one thread per body
- Each thread computes the sum of forces from all other bodies
- Bevy integration: custom `RenderApp` system in `ExtractSchedule`, write results to a `StorageBuffer`, read back via `BufferSlice::map_async`

### Recommendation

**CPU only. Do not GPU-accelerate N-body gravity.**

At 20-27 bodies, the computation takes single-digit microseconds on CPU. GPU dispatch overhead would make it 20-50x slower. The existing Go goroutine parallelization (and Rust rayon equivalent) is sufficient.

GPU gravity becomes worthwhile only if the simulator ever supports 1,000+ bodies (e.g., full asteroid belt as N-body particles). That would require a fundamentally different simulation architecture (Barnes-Hut octree or fast multipole method) regardless of CPU vs GPU.

---

## 2. Belt Particle Kepler Solving (1,500 Particles)

### Current Implementation

- **File:** `internal/render/belt.go:113-147` (`beltParticlePosition`)
- **Algorithm:** Per-particle Kepler equation solving via 5 Newton-Raphson iterations
- **Per-particle operations:**
  - 1 `sqrt` (mean motion)
  - 1 `fmod` (mean anomaly wrapping)
  - 5 iterations x (`sin`, `cos`, 2 subtracts, 1 divide) = 10 trig calls + 10 FLOPs
  - 1 `sqrt`, 1 `sqrt` (true anomaly)
  - 1 `atan2`, 1 `sin`, 1 `cos` (true anomaly)
  - 1 `cos` (radial distance)
  - 2 `cos`, 2 `sin` (orbital plane position)
  - 1 `cos`, 1 `sin` (inclination rotation)
- **Total per particle:** ~15 trig calls + ~30 arithmetic ops
- **Total per frame:** 1,500 particles x 15 trig = **22,500 trig calls/frame**

### FLOP Analysis

| Metric | Value |
|--------|-------|
| Particles | 1,500 |
| Trig calls/particle | ~15 |
| Arithmetic ops/particle | ~30 |
| Total trig calls/frame | ~22,500 |
| Total FLOPs/frame | ~67,500 |
| Estimated CPU time | ~100-200 us (single core) |

### GPU Compute Feasibility

GPU trig throughput is massive -- a modern GPU can evaluate >1 billion `sin`/`cos` per second. With 1,500 particles and ~15 trig calls each, the GPU computation would take <1 us. However:

- **GPU dispatch overhead:** ~100 us per dispatch
- **Data upload:** 1,500 particles x 32 bytes (a, e, inc, M0) = 48 KB upload
- **Data readback:** 1,500 particles x 12 bytes (xyz) = 18 KB readback

**Total GPU path:** ~100 us (overhead) + ~1 us (compute) + ~20 us (readback) = ~121 us
**Total CPU path:** ~100-200 us (single core), or ~50-100 us with SIMD/parallel

The crossover is marginal at 1,500 particles. At 10,000+ particles, GPU wins decisively.

### Bevy Integration Approach

In the Bevy architecture, belt particles become 1,500 instanced entities with `Transform` components. The Kepler solver runs as a CPU system updating `Transform` for each entity:

```rust
fn update_belt_positions(
    time: Res<SimulationTime>,
    mut query: Query<(&BeltParticle, &mut Transform)>,
) {
    query.par_iter_mut().for_each(|(particle, mut transform)| {
        let pos = solve_kepler(particle, time.current);
        transform.translation = pos;
    });
}
```

Using Bevy's `par_iter_mut()` (backed by rayon) distributes 1,500 particles across all CPU cores. On a 4-core system, that is ~375 particles/core, completing in ~25-50 us.

If GPU compute is desired (e.g., for 10,000+ particles), a wgpu compute shader would:
- Storage buffer input: particle orbital elements (32 bytes/particle)
- Storage buffer output: positions (16 bytes/particle, `vec4<f32>`)
- Uniform: simulation time
- Workgroup size: `@workgroup_size(256)`, ceil(n/256) dispatches
- Write results directly to instance buffer used by Bevy's instanced rendering

### Recommendation

**CPU with parallel iteration. Consider GPU compute only at 10,000+ particles.**

At 1,500 particles, Bevy's `par_iter_mut()` handles this in ~25-50 us across CPU cores. The overhead of GPU dispatch and readback eliminates any GPU advantage. If the particle count grows to 10,000+ (e.g., a more detailed Kuiper belt), GPU compute becomes worthwhile and straightforward to add as a wgpu compute pass that writes directly to the instance buffer.

---

## 3. Ray Tracing (Optional Mode)

### Current Implementation

Four identical ray tracers exist across backends:

| Backend | File | LOC | Shader Language |
|---------|------|-----|-----------------|
| Rust wgpu | `crates/render_core/src/raytracer.rs:4-213` | 213 | WGSL compute |
| Metal native | `native_gpu/metal/raytracer.metal` | 233 | MSL compute |
| CUDA native | `native_gpu/cuda/raytracer.cu` | 287 | CUDA kernel |
| ROCm native | `native_gpu/rocm/raytracer.hip` | 261 | HIP kernel |

**Algorithm (identical across all 4):**
- Orthographic ray cast in +Z direction (no perspective)
- Linear search for nearest sphere intersection (max 16 spheres)
- Lambertian diffuse + hard shadow ray to Sun
- 4-sample cosine-weighted ambient occlusion
- Glossy reflection for gas giants (material == 2)
- Equirectangular texture sampling from atlas
- Progressive accumulation over frames
- sRGB gamma correction
- PCG hash RNG

**Per-pixel cost:** 1 intersection test per sphere (16 max) + 1 shadow ray (16 intersections) + 4 AO rays (16 intersections each) + optional glossy ray (16 intersections) = 16 + 16 + 64 + 16 = **112 sphere-ray intersections per pixel**.

**At 1920x1080:** ~2M pixels x 112 intersections = ~224M intersection tests per frame.

### What the Ray Tracer Adds Beyond PBR

| Feature | Custom RT | Bevy PBR |
|---------|-----------|----------|
| Shadows | Single hard shadow ray | Shadow maps (configurable cascades, soft shadows via PCF) |
| Ambient occlusion | 4-sample stochastic AO | SSAO post-process (full-screen, higher quality) |
| Reflections | Single glossy bounce | Environment maps, SSR, or reflection probes |
| Lighting model | Lambertian only | Full PBR (metallic/roughness workflow, Fresnel, energy conservation) |
| Projection | Orthographic only | Perspective and orthographic |
| Anti-aliasing | None (progressive accumulation converges) | MSAA, TAA, or FXAA |
| Max objects | 16 spheres (hardcoded) | Unlimited (depth-buffered rasterization) |
| Textures | Equirectangular atlas sampling | Per-material textures, normal maps, emission maps |
| Sun glow | Emission material heuristic | Bloom post-process (HDR, physically-based) |

**Bevy PBR strictly exceeds the current RT implementation** in every dimension except one: the RT produces "true" ambient occlusion from actual geometry rather than screen-space approximation. However, with only ~20 spheres in the scene, SSAO achieves equivalent results.

### Recommendation

**Delete all 4 ray tracers. Use Bevy PBR + SSAO + shadow maps + bloom.**

The custom ray tracer was a valuable proof-of-concept but is now redundant. Bevy's PBR pipeline provides:
- Better visual quality (PBR materials, proper Fresnel, HDR bloom)
- Better performance (rasterization + post-processing vs per-pixel ray marching)
- Perspective rendering (the RT is orthographic-only)
- Zero maintenance cost (no custom shaders)
- Unlimited object count (vs 16 sphere cap)

If a custom RT mode is desired in the future for educational purposes, the WGSL shader from `raytracer.rs` can be adapted to a Bevy custom render node. Estimated effort: 2-3 weeks.

---

## 4. Rasterization (Default Mode)

### Current Implementation

| Path | Technology | Features |
|------|-----------|----------|
| CPU (default) | Go + Fyne canvas objects | Full feature set: textured planets, belt, trails, comets, spacetime, labels |
| GPU (Rust wgpu) | Custom WGSL pipelines | Textured planets, trails, spacetime, distance line (no belt/comets) |
| Native (Metal/CUDA/ROCm) | Custom kernels | RT mode or flat circles only (no textures in raster mode) |

### Bevy Built-in vs Custom Pipeline

| Aspect | Custom wgpu Pipelines | Bevy Built-in |
|--------|----------------------|---------------|
| Planet rendering | SDF circle + equirect UV mapping (2D) | 3D sphere mesh + `StandardMaterial` (PBR, normal maps, roughness) |
| Sun glow | Custom Gaussian fragment shader | `BloomSettings` on camera + emissive material |
| Trail lines | `LineList` primitive (1px width) | `bevy_polyline` (GPU-accelerated wide lines, per-vertex color) |
| Depth testing | None (painter's algorithm via draw order) | Hardware depth buffer |
| Anti-aliasing | SDF smoothstep on circles only | MSAA or TAA (full scene) |
| Pipeline management | Manual bind groups, vertex buffers, pipeline creation | Automatic via Bevy's render graph |
| Pixel readback | `MAP_READ` buffer + `poll(Wait)` per frame | None (direct-to-swapchain) |

The biggest performance win from Bevy is **eliminating pixel readback**. The current Rust wgpu renderer reads back every frame to an RGBA buffer for Fyne to display (`renderer.rs` readback path). This GPU-CPU sync point costs 1-5 ms per frame depending on resolution. Bevy renders directly to the window surface.

### Recommendation

**Use Bevy's built-in rendering. Delete all custom rasterization pipelines.**

Bevy's PBR renderer is strictly superior to the current custom pipelines:
- True 3D with depth buffer (current pipelines are 2D orthographic projections)
- PBR materials with metallic/roughness
- Automatic instancing for belt particles
- No pixel readback overhead
- No manual vertex buffer management

---

## 5. Native GPU Backend Disposition

### Decision: Delete All Three (Metal, CUDA, ROCm)

| Backend | Files | LOC | Decision | Rationale |
|---------|-------|-----|----------|-----------|
| **Metal native** | `native_gpu/metal/raytracer.metal`, `renderer.m`, Makefile | ~909 | **Delete** | wgpu targets Metal automatically via Bevy. The Metal RT is algorithmically identical to WGSL RT. Threadgroup shared memory optimization is irrelevant for 16 spheres (fits in L1 cache). |
| **CUDA native** | `native_gpu/cuda/raytracer.cu`, `renderer.cu`, `hardware.cu`, Makefile | ~771 | **Delete** | wgpu targets Vulkan on NVIDIA GPUs. No CUDA-specific feature (tensor cores, RTX hardware RT) is used. The shared memory optimization is irrelevant at 16 spheres. |
| **ROCm native** | `native_gpu/rocm/raytracer.hip`, `renderer.hip`, `hardware.hip`, Makefile | ~574 | **Delete** | wgpu targets Vulkan on AMD GPUs. The HIP kernel is a character-identical port of the CUDA kernel. |
| **Shared C code** | `native_gpu/common/rasterizer.c`, `camera.c`, `native_render.h` | ~139 | **Delete** | CPU Bresenham rasterizer used only as fallback in native backends. Bevy handles all rasterization. |
| **Go FFI bridges** | `internal/ffi/render_metal.go`, `render_cuda.go`, `render_rocm.go` | ~420 | **Delete** | No FFI needed in pure Rust Bevy architecture. |

### What is Lost

| Feature | Metal-Specific | CUDA-Specific |
|---------|---------------|---------------|
| Threadgroup/shared memory for sphere data | wgpu `var<workgroup>` provides the same (if needed) | Same |
| Hardware texture filtering (`tex2DLayered`) | wgpu `textureSampleLevel` provides the same | Same |
| CUDA tensor cores | Not used by current code | Not available via wgpu |
| Metal MPS (Metal Performance Shaders) | Not used by current code | N/A |
| CUDA RT cores (hardware ray tracing) | N/A | Not used by current code |

**Nothing of value is lost.** The native backends use no platform-specific features beyond basic compute dispatch and texture sampling, both of which wgpu provides identically.

### What is Gained

- **Single codebase:** 1 WGSL shader replaces 4 (WGSL + MSL + CUDA + HIP)
- **Zero FFI:** No CGO, no C API, no shared libraries to link
- **Zero build complexity:** No `metal_render`/`cuda_render`/`rocm_render` build tags, no per-platform Makefiles
- **Automatic platform targeting:** Bevy/wgpu selects Metal, Vulkan, or DX12 at runtime
- **Savings:** ~2,254 native LOC + ~420 FFI LOC + 3 Makefiles + 6 build configurations

### Deletion Timeline

| Phase | Action | Savings |
|-------|--------|---------|
| Phase A (pre-migration) | Delete `native_gpu/` entirely + FFI bridges | ~2,674 LOC, 3 Makefiles |
| Phase B (Bevy core) | Delete `crates/render_core/` + Go GPU renderer | ~3,190 LOC |
| Phase C (feature parity) | Delete Go CPU renderer | ~1,550 LOC |

---

## Summary Decision Matrix

| Workload | Bodies/Particles | Current CPU Time | GPU Compute Worth It? | Bevy Approach | Decision |
|----------|-----------------|------------------|-----------------------|---------------|----------|
| N-body gravity | ~20 | ~2-5 us | **No** (overhead 20-50x > compute) | `solar_sim_core::step()` on CPU, sync to ECS | CPU only |
| Belt Kepler solving | 1,500 | ~100-200 us | **No** at 1,500 (marginal); **Yes** at 10,000+ | `par_iter_mut()` system updating `Transform` | CPU parallel |
| Ray tracing | 16 spheres max | N/A (GPU-only) | **No** (Bevy PBR is superior) | Bevy PBR + SSAO + shadow maps + bloom | Delete RT |
| Rasterization | All scene objects | ~8-12 ms (CPU) | Built-in via Bevy | Bevy render graph (mesh, material, camera) | Bevy built-in |
| Metal backend | macOS only | N/A | **No** (wgpu covers Metal) | Bevy/wgpu auto-selects Metal | Delete |
| CUDA backend | NVIDIA only | N/A | **No** (wgpu covers Vulkan) | Bevy/wgpu auto-selects Vulkan | Delete |
| ROCm backend | AMD only | N/A | **No** (wgpu covers Vulkan) | Bevy/wgpu auto-selects Vulkan | Delete |
