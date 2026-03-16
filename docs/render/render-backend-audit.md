# Render Backend Audit

## Overview

The simulator has 5 rendering backends, selected at build time via Go build tags. All backends share the same physics snapshot mechanism and viewport transform. GPU backends share an identical C API (`native_render.h`).

---

## 1. CPU Backend (Go/Fyne) -- Default

**Build tag:** none (default)
**Entry point:** `internal/render/renderer.go:92` -- `CreateCanvasFromSnapshot()`

### What it renders

| Feature | Implementation | File:Line |
|---------|---------------|-----------|
| Skybox background | Stretched `canvas.Image` from `milky_way.jpg` | `renderer.go:102-113` |
| Spacetime curvature grid | 40-150 warped grid lines via `spacetime.SpacetimeRenderer` | `renderer.go:115-118` |
| Asteroid belt (1500 particles) | Kepler-solved positions drawn as 1-3px dots into `*image.RGBA` | `belt.go:41-109` |
| Orbital trails | Catmull-Rom interpolated Bresenham lines into `*image.RGBA` | `trail_buffer.go:26-101` |
| Sun glow | Radial gradient `*image.RGBA`, cached by diameter | `lighting.go:147-181` |
| Sun body | Textured circular cutout or solid-color `canvas.Circle` | `renderer.go:187-199` |
| Planet/Moon bodies | Textured + Lambertian-shaded `*image.RGBA`, or solid circle | `renderer.go:211-261` |
| Asteroid shapes | Procedural 8-lobe potato shape `*image.RGBA` | `textures.go:248-293` |
| Comet tails | 8 gradient `canvas.Line` segments away from Sun | `renderer.go:477-523` |
| Text labels | `canvas.Text` objects per body | `renderer.go:267-272` |
| Launch trajectory | Color-gradient `canvas.Line` segments | `renderer.go:365-414` |
| Launch vehicle marker | Green `canvas.Circle` (8px) | `renderer.go:282-290` |
| Distance measurement | Yellow `canvas.Line` + AU/km/light-min text | `renderer.go:292-319` |

### Shaders/Kernels

None. Pure CPU rendering into Fyne canvas objects and `*image.RGBA` pixel buffers.

### Data received

- `[]physics.Body` snapshot (positions, velocities, colors, radii, trails, names, types)
- `physics.Body` sun snapshot
- `viewport.Snapshot` (precomputed camera state: zoom, pan, rotation trig, canvas dimensions)
- `simTime float64` (for belt Kepler computation)
- Display flags: `showTrails`, `showSpacetime`, `ShowLabels`, `ShowBelt`

### Performance characteristics

- Belt and trail rendering parallelized via goroutines (write to independent image buffers)
- Lambertian shading parallelized across CPU cores for images > 100px tall
- Object pool (`RenderCache`) eliminates per-frame allocation for circles, lines, text, images
- Lighting cache keyed on `"name_diameter"`, invalidated when Sun moves > 1e9 m
- Trail downsampled to max 200 segments per body
- Belt: 1500 particles, each solving 5-iteration Kepler equation per frame
- Buffer clearing uses C `memset` when CGO available
- Frame pacing: 60 FPS target, drops to ~4 FPS when paused

### Platform availability

All platforms (macOS, Linux, Windows). No CGO required (memset optimization is optional).

### Maturity

**Production-ready.** Full feature coverage. Default path used by all users without GPU build tags.

---

## 2. GPU Backend (Rust wgpu)

**Build tag:** `rust_render`
**Entry point:** `internal/render/gpu_renderer.go:122` -- `generateImage()`
**Rust code:** `crates/render_core/` (~2,890 LOC)

### What it renders

| Feature | Implementation | File:Line |
|---------|---------------|-----------|
| Background | Clear to `(5, 5, 15)` dark blue | `renderer.rs:472-477` |
| Spacetime curvature grid | Rust-side `generate_grid()` -> line vertices -> line pipeline | `spacetime.rs:68-205`, `renderer.rs:369-380` |
| Orbital trails | Line vertices with per-segment alpha fade -> line pipeline | `ffi.rs:132-210`, `renderer.rs:382-389` |
| Sun glow | Quad with Gaussian falloff fragment shader (additive blend) | `pipeline.rs:67-108`, `renderer.rs:396-414` |
| Sun body | Textured circle via SDF quad + texture atlas sampling | `pipeline.rs:3-65`, `renderer.rs:404-412` |
| Planet bodies | Textured circle via SDF quad + equirectangular UV mapping | `pipeline.rs:3-65`, `renderer.rs:416-426` |
| Distance line | Yellow line via line pipeline | `renderer.rs:429-440` |
| Text labels | **Delegated to CPU** -- `CreateLabelOverlay()` composites Fyne text on top | `renderer.go:417-475` |
| Ray tracing | Compute shader: sphere intersection, shadows, AO, glossy, texture sampling | `raytracer.rs:4-213` |

**Not rendered by GPU backend (CPU-only features):**
- Asteroid belt particles (no belt rendering in GPU path)
- Comet tails (no tail rendering in GPU path)
- Irregular asteroid shapes
- Launch trajectory (passed as extra trail instead)
- Launch vehicle marker (not present)

### Shaders/Kernels

| Shader | Type | File |
|--------|------|------|
| Circle shader (WGSL) | Vertex + Fragment | `pipeline.rs:3-65` |
| Glow shader (WGSL) | Vertex + Fragment | `pipeline.rs:67-108` |
| Line shader (WGSL) | Vertex + Fragment | `pipeline.rs:110-138` |
| Ray tracer (WGSL compute) | Compute (@workgroup_size 8,8) | `raytracer.rs:4-213` |

### Data received (via FFI)

From Go -> Rust via `render_core.h` C API:
- Camera: zoom, pan_x/y, rotation_x/y/z, use_3d, follow_x/y/z
- Bodies: positions `[n*3]f64`, colors `[n*4]f64`, radii `[n]f64`, sun separate
- Trails: trail_lengths `[n]u32`, trail_positions `[flat]f64`, trail_colors `[n*4]f64`
- Spacetime: masses `[n]f64`, positions `[n*3]f64`
- Distance line: 6x f64 world coords
- RT mode: enabled flag, samples_per_frame, max_bounces

All world-space coordinates are converted to screen-space by `Camera::world_to_screen()` on the Rust side.

### Performance characteristics

- Offscreen rendering to `Rgba8UnormSrgb` texture
- Readback via `MAP_READ` buffer (CPU-GPU sync per frame -- bottleneck)
- Persistent GPU vertex buffers with 1.5x over-allocation (reduces realloc)
- Texture atlas: 9 layers (8 planets + sun) at 2048x1024 each, Lanczos3 resize at load
- RT progressive accumulation: converges over multiple frames, resets on camera/body movement
- RT workgroup size: 8x8 (64 threads)
- Max 16 RT spheres (hardcoded buffer)

### Platform availability

Cross-platform via wgpu (Vulkan on Linux/Windows, Metal on macOS, DX12 on Windows). Requires Rust toolchain + CGO for build.

### Maturity

**Production-ready** for rasterization path. RT mode is functional but limited:
- Orthographic projection only (no perspective RT)
- No belt, comet tail, or irregular asteroid rendering
- Labels require CPU overlay compositing

---

## 3. Metal Native Backend

**Build tag:** `metal_render`
**Entry point:** `internal/ffi/render_metal.go` -> `native_gpu/metal/`
**Code:** ~676 LOC (.m host) + 233 LOC (.metal shader)

### What it renders

| Feature | Implementation |
|---------|---------------|
| Ray-traced spheres | `raytracer.metal` compute kernel -- identical algorithm to WGSL RT |
| CPU fallback raster | Solid-color circles via `rasterizer.c` when RT disabled |
| Trails | CPU Bresenham lines via `rasterizer.c` overlay |
| Distance line | CPU Bresenham line via `rasterizer.c` overlay |
| Spacetime grid | **Not implemented** (flag accepted but ignored) |

### Shaders/Kernels

| Shader | Type | File |
|--------|------|------|
| `raytrace` | Metal compute kernel (8x8 threadgroups) | `native_gpu/metal/raytracer.metal` |

Features: threadgroup memory for sphere data (16 spheres), PCG hash RNG, Lambertian + shadow + AO (4 samples) + glossy reflection, texture atlas sampling via `texture2d_array`, progressive accumulation, sRGB gamma.

### Data received

Same C API as Rust backend (`native_render.h`). Camera, bodies, trails, spacetime, distance line, RT settings.

### Performance characteristics

- Threadgroup shared memory for sphere data (768 bytes for 16 spheres)
- Direct Metal compute dispatch, no intermediate abstraction
- CPU-side overlay compositing for trails and distance lines (no GPU line rendering)
- Pixel readback via `cudaMemcpy`-equivalent (Metal blit)

### Platform availability

**macOS only.** Requires Metal framework, Foundation, CoreGraphics.

### Maturity

**Functional but incomplete.** RT works. Raster fallback is minimal (solid circles only -- no textures, no glow, no SDF anti-aliasing). Spacetime grid not implemented. Trails/distance rendered on CPU.

---

## 4. CUDA Native Backend

**Build tag:** `cuda_render`
**Entry point:** `internal/ffi/render_cuda.go` -> `native_gpu/cuda/`
**Code:** `renderer.cu` (484 LOC) + `raytracer.cu` (287 LOC) + `hardware.cu`

### What it renders

Identical feature set to Metal native backend:

| Feature | Implementation |
|---------|---------------|
| Ray-traced spheres | `raytracer.cu` CUDA kernel (8x8 blocks) |
| CPU fallback raster | Solid-color circles via `rasterizer.c` |
| Trails | CPU Bresenham lines via `rasterizer.c` |
| Distance line | CPU Bresenham line via `rasterizer.c` |
| Spacetime grid | **Not implemented** (flag ignored) |

### Shaders/Kernels

| Kernel | Type | File |
|--------|------|------|
| `raytrace_kernel` | CUDA global kernel (8x8 blocks) | `native_gpu/cuda/raytracer.cu` |

Features: `__shared__` memory for sphere data, `tex2DLayered` for atlas sampling, same lighting model as WGSL/Metal (shadow, AO, glossy), progressive accumulation, sRGB gamma.

### Data received

Same C API (`native_render.h`).

### Performance characteristics

- Shared memory for sphere data
- Layered CUDA texture object for atlas (hardware-filtered sampling)
- `cudaDeviceSynchronize()` after kernel dispatch (full sync)
- Host-to-device memcpy for sphere data each frame
- Device-to-host memcpy for pixel readback each frame

### Platform availability

**Linux/Windows with NVIDIA GPU.** Requires CUDA runtime (`libcudart`).

### Maturity

**Functional but incomplete.** Same gaps as Metal: no spacetime grid, CPU-only trail/distance rendering, no SDF circle raster, no glow effect in non-RT mode.

---

## 5. ROCm/HIP Native Backend

**Build tag:** `rocm_render`
**Entry point:** `internal/ffi/render_rocm.go` -> `native_gpu/rocm/`
**Code:** `renderer.hip` (313 LOC) + `raytracer.hip` (261 LOC) + `hardware.hip`

### What it renders

Identical feature set to CUDA backend (HIP is a near-identical port with `hip*` API substitutions).

### Shaders/Kernels

| Kernel | Type | File |
|--------|------|------|
| `raytrace_kernel` | HIP global kernel (8x8 blocks) | `native_gpu/rocm/raytracer.hip` |

### Platform availability

**Linux with AMD GPU.** Requires ROCm runtime.

### Maturity

**Functional but incomplete.** Direct port of CUDA backend. Same feature gaps. The feature inventory doc notes "Partial" -- no native ROCm source was initially found, though files now exist.

---

## Backend Comparison Matrix

| Feature | CPU (Go) | GPU (Rust wgpu) | Metal Native | CUDA Native | ROCm Native |
|---------|----------|-----------------|--------------|-------------|-------------|
| Textured planets | Yes (circular cutout + Lambertian) | Yes (SDF quad + atlas + equirect UV) | RT only (atlas sampling) | RT only (atlas sampling) | RT only (atlas sampling) |
| Sun glow | Yes (radial gradient image) | Yes (Gaussian fragment shader, additive) | RT glow only | RT glow only | RT glow only |
| Orbital trails | Yes (Catmull-Rom + Bresenham) | Yes (GPU line primitives) | CPU Bresenham overlay | CPU Bresenham overlay | CPU Bresenham overlay |
| Asteroid belt | Yes (1500 Kepler particles) | No | No | No | No |
| Comet tails | Yes (8-segment gradient) | No | No | No | No |
| Irregular asteroids | Yes (procedural potato) | No | No | No | No |
| Spacetime grid | Yes (adaptive warped grid) | Yes (Rust-computed, line pipeline) | No (stubbed) | No (stubbed) | No (stubbed) |
| Distance line | Yes (Fyne line + text) | Yes (GPU line + CPU text overlay) | CPU line overlay | CPU line overlay | CPU line overlay |
| Labels | Yes (Fyne text) | CPU overlay | CPU overlay | CPU overlay | CPU overlay |
| Launch trajectory | Yes (Fyne lines) | Yes (as extra trail) | As extra trail | As extra trail | As extra trail |
| Ray tracing | No | Yes (WGSL compute) | Yes (Metal compute) | Yes (CUDA kernel) | Yes (HIP kernel) |
| Skybox | Yes (image stretch) | No (solid background) | No | No | No |
| Anti-aliasing | None | SDF smoothstep on circles | None (RT only) | None (RT only) | None (RT only) |
| Platform | All | All (via wgpu) | macOS | Linux/Win (NVIDIA) | Linux (AMD) |
| CGO required | No* | Yes | Yes | Yes | Yes |
| Maturity | Production | Production (raster), Good (RT) | Functional, Incomplete | Functional, Incomplete | Functional, Incomplete |

*CGO used for memset optimization but gracefully falls back.
