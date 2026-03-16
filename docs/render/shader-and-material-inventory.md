# Shader and Material Inventory

Complete catalog of every shader, compute kernel, and material in the project.

---

## WGSL Raster Shaders (Rust wgpu)

### 1. Circle Shader

**File:** `crates/render_core/src/pipeline.rs:3-65`
**Pipeline:** `circle_pipeline` (TriangleList, alpha blend)

**Vertex attributes (`CircleVertex`, 40 bytes):**

| Location | Attribute | Format | Offset |
|----------|-----------|--------|--------|
| 0 | `position` | `Float32x2` | 0 |
| 1 | `center` | `Float32x2` | 8 |
| 2 | `radius` | `Float32` | 16 |
| 3 | `color` | `Float32x4` | 20 |
| 4 | `texture_index` | `Sint32` | 36 |

**Uniforms:**
- Group 0, Binding 0: `mat4x4<f32>` projection (orthographic)
- Group 1, Binding 0: `texture_2d_array<f32>` texture atlas
- Group 1, Binding 1: `sampler` texture sampler

**What it computes:**
- Vertex: applies orthographic projection, passes center/radius/color/texture_index to fragment
- Fragment: SDF circle discard (`smoothstep` at radius boundary for anti-aliasing). If `texture_index >= 0`, computes equirectangular UV from sphere-surface normal (`atan2`/`asin` mapping) and samples texture atlas layer. Output: `vec4<f32>(base_color.rgb, base_color.a * alpha)`.

**Geometry:** 6 vertices per circle (2 triangles forming a quad with `radius + 2px` margin for AA).

**Bevy migration:** Replace entirely with `Mesh::from(Sphere::new(radius))` + `StandardMaterial`. The SDF approach is unnecessary when using real 3D meshes. The equirectangular UV mapping is handled by Bevy's sphere mesh UV layout. **Delete.**

---

### 2. Glow Shader

**File:** `crates/render_core/src/pipeline.rs:67-108`
**Pipeline:** `glow_pipeline` (TriangleList, **additive** blend)

**Vertex attributes:** Same `CircleVertex` layout as circle shader (texture_index ignored).

**Uniforms:**
- Group 0, Binding 0: `mat4x4<f32>` projection

**What it computes:**
- Vertex: same as circle shader
- Fragment: Gaussian intensity falloff `exp(-(dist^2) / (2 * glow_radius^2))` where `glow_radius = radius * 0.5`. Discard below 0.01 intensity. Output: `vec4<f32>(color.rgb, color.a * intensity * 0.4)`. Additive blending creates the glow effect.

**Geometry:** 6 vertices per glow quad (3x the sphere radius).

**Bevy migration:** Replace with `BloomSettings` on the camera + emissive `StandardMaterial` on the Sun. The bloom post-process produces equivalent (or better) glow. **Delete.**

---

### 3. Line Shader

**File:** `crates/render_core/src/pipeline.rs:110-138`
**Pipeline:** `line_pipeline` (LineList, alpha blend)

**Vertex attributes (`LineVertex`, 24 bytes):**

| Location | Attribute | Format | Offset |
|----------|-----------|--------|--------|
| 0 | `position` | `Float32x2` | 0 |
| 1 | `color` | `Float32x4` | 8 |

**Uniforms:**
- Group 0, Binding 0: `mat4x4<f32>` projection

**What it computes:**
- Vertex: applies orthographic projection
- Fragment: passes through vertex color `vec4<f32>`

**Used for:** Trails (per-vertex alpha fade), spacetime grid (per-segment color), distance measurement line (yellow).

**Bevy migration:** Replace with `bevy_polyline` for trails and `Gizmos` for measurement lines. Spacetime grid lines would use a custom line mesh or `Gizmos`. **Delete.**

---

## WGSL Compute Shader (Rust wgpu)

### 4. Ray Tracer Compute Shader

**File:** `crates/render_core/src/raytracer.rs:4-213`
**Pipeline:** `rt_compute_pipeline` (compute, workgroup_size 8x8)

**Bind group layout:**

| Binding | Type | Description |
|---------|------|-------------|
| 0 | `texture_storage_2d<rgba8unorm, write>` | Output texture |
| 1 | `storage<read> array<Sphere>` | Sphere data (max 16) |
| 2 | `uniform Camera` | Camera/frame parameters |
| 3 | `storage<read_write> array<vec4<f32>>` | Accumulation buffer |
| 4 | `texture_2d_array<f32>` | Texture atlas |
| 5 | `sampler` | Texture sampler |

**Sphere struct (48 bytes):**
```
center: vec3<f32>, radius: f32,
color: vec4<f32>,
material: u32, texture_index: i32, _pad: u32[2]
```

**Camera struct (32 bytes):**
```
width: f32, height: f32, frame_count: u32, num_spheres: u32,
sun_screen_x: f32, sun_screen_y: f32,
samples_per_frame: u32, max_bounces: u32
```

**What it computes:**
1. Orthographic ray cast in +Z direction from pixel position
2. Linear search for nearest sphere intersection
3. If hit emissive (material==1): glow lighting
4. If hit diffuse (material==0): Lambertian + shadow ray to Sun + 4-sample cosine-weighted AO + optional glossy reflection (material==2)
5. Texture sampling via `textureSampleLevel` with equirectangular UV from surface normal
6. sRGB gamma correction: `pow(color, 1/2.2)`
7. Progressive accumulation: weighted running average over frames
8. Store to output texture

**RNG:** PCG hash, seeded by pixel position + frame count.

**Bevy migration:** Replace with Bevy's PBR pipeline (shadow maps + SSAO + bloom). If custom RT is needed, this WGSL shader can be integrated as a custom Bevy render node, but significant plumbing is required (render graph integration, bind group management). **Recommend delete; PBR is superior.**

---

## Metal Shaders (Native macOS)

### 5. Metal Ray Tracer

**File:** `native_gpu/metal/raytracer.metal`
**Kernel:** `raytrace` (compute, 8x8 threadgroups)

**Buffers:**

| Index | Type | Description |
|-------|------|-------------|
| buffer(0) | `device const Sphere*` | Sphere array |
| buffer(1) | `constant Camera&` | Camera uniform |
| buffer(2) | `device float4*` | Accumulation buffer |
| buffer(3) | `device uchar4*` | RGBA8 output |
| texture(0) | `texture2d_array<float>` | Texture atlas |
| sampler(0) | `sampler` | Texture sampler |

**Optimization over WGSL:** Uses `threadgroup Sphere s_spheres[16]` for shared memory access (Metal's `threadgroup_barrier(mem_flags::mem_threadgroup)`).

**Algorithm:** Identical to WGSL RT shader -- orthographic ray, sphere intersection, shadow, AO (4 samples), glossy reflection, texture sampling, progressive accumulation, sRGB gamma.

**Output:** Writes directly to `uchar4` output buffer (RGBA8), not storage texture.

**Bevy migration:** **Delete.** Bevy uses wgpu which supports Metal natively. No need for hand-written MSL.

---

## CUDA Kernels (NVIDIA)

### 6. CUDA Ray Tracer

**File:** `native_gpu/cuda/raytracer.cu`
**Kernel:** `raytrace_kernel` (8x8 thread blocks)

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `spheres` | `const RTSphere*` | Device sphere array |
| `cam` | `RTCameraUniform` | Camera uniform (by value) |
| `accum` | `float4*` | Accumulation buffer |
| `output` | `uint8_t*` | RGBA8 output |
| `atlas` | `cudaTextureObject_t` | Layered texture atlas |
| `tex_width/height` | `uint32_t` | Atlas dimensions |

**Optimization:** `__shared__ RTSphere s_spheres[16]` for shared memory. `tex2DLayered<float4>()` for hardware-filtered texture sampling.

**Algorithm:** Identical to WGSL and Metal RT shaders.

**Launcher:** `launch_raytrace_kernel()` -- sets up `dim3` grid/block and dispatches.

**Bevy migration:** **Delete.** wgpu compute shaders (which Bevy uses) work on NVIDIA via Vulkan. No need for CUDA-specific code.

---

### 7. CUDA Renderer Host Code

**File:** `native_gpu/cuda/renderer.cu` (484 LOC)

Not a shader, but host code that:
- Manages CUDA device memory (sphere buffer, accumulation buffer, output buffer)
- Creates layered CUDA texture atlas from `TextureAtlasData`
- Dispatches `raytrace_kernel` when RT enabled
- Falls back to CPU rasterization (`rasterizer.c`) when RT disabled
- Composites trails and distance lines via CPU Bresenham overlay
- Implements the full `native_render.h` C API

**Bevy migration:** **Delete.** All GPU management is handled by Bevy/wgpu.

---

## HIP/ROCm Kernels (AMD)

### 8. ROCm Ray Tracer

**File:** `native_gpu/rocm/raytracer.hip` (261 LOC)
**Kernel:** `raytrace_kernel` (8x8 blocks, `hipLaunchKernelGGL`)

Near-identical to CUDA kernel with `hip*` API substitutions:
- `__shared__` -> same
- `__syncthreads()` -> same
- `tex2DLayered<float4>()` -> `tex2DLayered<float4>()`
- `cudaTextureObject_t` -> `hipTextureObject_t`

**Bevy migration:** **Delete.** wgpu/Vulkan works on AMD GPUs.

### 9. ROCm Renderer Host Code

**File:** `native_gpu/rocm/renderer.hip` (313 LOC)

Same as CUDA renderer with `hip*` API substitutions. **Delete.**

---

## C Rasterizer (Shared)

### 10. CPU Rasterizer

**File:** `native_gpu/common/rasterizer.c` (63 LOC)

Functions:
- `raster_blend_pixel()` -- alpha-blend a single pixel into RGBA buffer
- `raster_draw_line()` -- Bresenham line drawing with alpha blending
- `raster_draw_circle()` -- filled circle drawing (brute-force distance check)

Used by CUDA and ROCm renderers as CPU fallback when RT is disabled, and for overlay compositing (trails, distance lines).

**Bevy migration:** **Delete.** Bevy handles all rasterization.

### 11. CPU Camera

**File:** `native_gpu/common/camera.c` (76 LOC)

`camera_world_to_screen()` -- port of `Camera::world_to_screen()` from Rust. Used by CUDA/ROCm host code for coordinate transforms.

**Bevy migration:** **Delete.** Bevy's `Camera` + `Transform` handle all projection.

---

## Go CPU "Shaders" (Software Rendering)

### 12. Lambertian Diffuse Shading

**File:** `internal/render/lighting.go:30-143`

Per-pixel Lambertian diffuse shading applied to circular planet textures:
- Input: source image, planet position, Sun position
- Computes sphere normal from pixel (x,y) offset
- `dot = nx*lightDir.X + (-ny)*lightDir.Y + nz*lightDir.Z`
- `intensity = ambient(0.15) + diffScale * dot`, clamped
- Parallelized across CPU cores for images > 100px

**Bevy migration:** **Delete.** Bevy's PBR pipeline handles diffuse lighting automatically via `PointLight` + `StandardMaterial`.

### 13. Sun Glow Generator

**File:** `internal/render/lighting.go:147-181`

Procedural radial gradient: `alpha = (1 - distSq) * 0.4`, yellow-orange color.

**Bevy migration:** **Delete.** Replace with bloom post-processing.

### 14. Bresenham Trail Renderer

**File:** `internal/render/trail_buffer.go:104-169`

Software Bresenham line drawing with alpha-over compositing into pixel buffer.

**Bevy migration:** **Delete.** Replace with `bevy_polyline` or equivalent.

---

## Summary: Shader Migration Decisions

| # | Shader/Kernel | File | Port to Bevy WGSL? | Decision |
|---|---------------|------|--------------------:|----------|
| 1 | Circle shader (WGSL) | `pipeline.rs:3-65` | No | **Delete** -- use `Mesh` + `StandardMaterial` |
| 2 | Glow shader (WGSL) | `pipeline.rs:67-108` | No | **Delete** -- use bloom post-processing |
| 3 | Line shader (WGSL) | `pipeline.rs:110-138` | No | **Delete** -- use `bevy_polyline` / `Gizmos` |
| 4 | RT compute (WGSL) | `raytracer.rs:4-213` | Maybe | **Delete** if PBR suffices; otherwise custom render node |
| 5 | Metal RT | `raytracer.metal` | No | **Delete** -- wgpu covers Metal |
| 6 | CUDA RT | `raytracer.cu` | No | **Delete** -- wgpu covers Vulkan/NVIDIA |
| 7 | CUDA host | `renderer.cu` | No | **Delete** |
| 8 | ROCm RT | `raytracer.hip` | No | **Delete** -- wgpu covers Vulkan/AMD |
| 9 | ROCm host | `renderer.hip` | No | **Delete** |
| 10 | C rasterizer | `rasterizer.c` | No | **Delete** |
| 11 | C camera | `camera.c` | No | **Delete** |
| 12 | Lambertian shading | `lighting.go` | No | **Delete** -- Bevy PBR |
| 13 | Sun glow generator | `lighting.go` | No | **Delete** -- Bevy bloom |
| 14 | Bresenham trails | `trail_buffer.go` | No | **Delete** -- `bevy_polyline` |
