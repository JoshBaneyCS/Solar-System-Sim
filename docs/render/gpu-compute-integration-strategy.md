# GPU Compute Integration Strategy

Decision document for the future of native GPU backends and compute workloads in the Bevy migration.

---

## Current State

The project maintains **4 GPU rendering backends** plus a CPU fallback:

| Backend | Language | GPU API | Platform | LOC |
|---------|----------|---------|----------|-----|
| Rust wgpu | Rust | wgpu (Vulkan/Metal/DX12) | All | ~2,890 |
| Metal native | Objective-C + MSL | Metal | macOS | ~909 |
| CUDA native | C/C++ + CUDA | CUDA Runtime | Linux/Win (NVIDIA) | ~771 |
| ROCm native | C/C++ + HIP | HIP/ROCm | Linux (AMD) | ~574 |
| CPU fallback | Go | None | All | ~1,550 |

All 3 native backends implement an identical algorithm (sphere ray tracing with shadows, AO, glossy reflections, texture sampling, progressive accumulation). The CUDA and ROCm backends are near-character-identical ports of each other.

---

## Question 1: Should we keep the native Metal/CUDA/ROCm ray tracers?

**Recommendation: No. Delete all three.**

### Rationale

1. **wgpu already covers all platforms.** wgpu compiles to Vulkan (Linux, Windows, Android), Metal (macOS, iOS), and DX12 (Windows). Every platform served by CUDA, ROCm, and Metal is already covered by wgpu -- which is exactly what Bevy uses internally.

2. **The algorithm is identical across all 4 GPU backends.** The WGSL, Metal MSL, CUDA, and HIP ray tracers implement the exact same math (sphere intersection, PCG hash, Lambertian + shadow + AO + glossy, progressive accumulation). There is zero algorithmic advantage to any native backend.

3. **The native backends are feature-incomplete.** Metal/CUDA/ROCm render only ray-traced spheres (when RT is on) or flat solid circles (when RT is off). They lack:
   - Spacetime grid visualization (flag accepted but ignored)
   - SDF anti-aliased circles
   - Glow effect (non-RT mode)
   - Skybox rendering
   - The Rust wgpu backend is strictly more capable.

4. **Maintenance cost is high.** Every change to the ray tracer must be replicated across 4 shader languages (WGSL, MSL, CUDA, HIP). The C API (`native_render.h`) with its identical Go FFI wrappers (`render_metal.go`, `render_cuda.go`, `render_rocm.go`) adds ~840 LOC of boilerplate. Build infrastructure includes 3 separate Makefiles for library compilation.

5. **Bevy eliminates the need for manual GPU backend selection.** Bevy's wgpu integration automatically selects the optimal backend (Vulkan, Metal, or DX12) at runtime. Users never need to choose a build tag.

### Performance argument

The Metal backend uses threadgroup shared memory, and the CUDA backend uses `__shared__` memory for sphere data. This is an optimization over the WGSL compute shader which uses global memory. However:
- The sphere count is at most 16 (768 bytes). L1 cache on any modern GPU handles this trivially.
- The overhead of CGO FFI + pixel readback dominates frame time, not sphere buffer access.
- In the Bevy architecture, ray tracing (if kept) would use wgpu compute without any FFI boundary.

---

## Question 2: Should we consolidate to wgpu compute?

**Recommendation: Yes, but only if custom RT is needed. Otherwise, rely on Bevy's built-in PBR pipeline.**

### Analysis

The current ray tracer provides:
- Orthographic sphere intersection
- Hard shadow rays from a single point light (the Sun)
- 4-sample cosine-weighted ambient occlusion
- Glossy reflections for gas giants (material==2)
- Progressive accumulation for noise reduction
- Texture atlas sampling

Bevy's standard PBR pipeline provides (without any custom shaders):
- **Perspective and orthographic rendering** with proper depth buffer
- **Shadow mapping** with configurable cascade count and quality
- **Screen-space ambient occlusion (SSAO)** via `ScreenSpaceAmbientOcclusion`
- **Bloom** for emissive objects (sun glow)
- **Physically-based materials** with roughness/metallic for varied surface appearances
- **Environment mapping / IBL** for realistic reflections

**The Bevy PBR pipeline exceeds the visual quality of the current RT implementation** while being:
- Dramatically simpler to maintain (zero custom shaders)
- More performant (rasterization + post-processing vs per-pixel ray marching)
- More feature-rich (depth testing, multi-light, environment maps, normal maps)

### If custom RT is desired in the future

Bevy supports custom render nodes and compute passes. The WGSL compute shader from `raytracer.rs` could be adapted to a Bevy render plugin. This would require:
1. Registering a custom `RenderApp` sub-app stage
2. Creating bind group layouts and compute pipeline via Bevy's render resource API
3. Managing GPU buffers through Bevy's `RenderAssets`
4. Integrating into the render graph

Estimated effort: 2-3 weeks for a senior Bevy developer. But this should only be done if PBR proves insufficient for the project's visual goals.

---

## Question 3: Cost of maintaining 3 native backends vs 1 wgpu path

| Cost Factor | 3 Native Backends | 1 wgpu/Bevy Path |
|------------|-------------------|-------------------|
| Shader code | 4 implementations (WGSL + MSL + CUDA + HIP) | 0 custom shaders (PBR) or 1 WGSL (if RT) |
| Host code | 3 host implementations (Metal .m, CUDA .cu, HIP .hip) | 0 (Bevy manages GPU) |
| FFI glue | 4 Go files (render_rust.go, render_metal.go, render_cuda.go, render_rocm.go) | 0 (pure Rust) |
| C API | native_render.h + render_core.h (identical APIs) | 0 |
| Build system | 3 Makefiles + build tags + library linking | `cargo build` (Bevy handles deps) |
| CI | Must test 4 GPU configurations per PR | 1 configuration |
| Bug fixes | 4x effort for any RT algorithm change | 1x effort |
| Platform testing | Need macOS + NVIDIA + AMD hardware | Any wgpu-supported system |
| Total LOC | ~5,144 (Rust wgpu + Metal + CUDA + ROCm) | ~0 (Bevy built-in) or ~500 (custom RT plugin) |

**Savings from deletion: ~5,100 LOC of rendering code, 4 Go FFI files (~840 LOC), 3 Makefiles, 6+ build tag configurations.**

---

## Question 4: Per-workload recommendations

| GPU Workload | Current Backend(s) | Recommended Future State | Rationale |
|-------------|-------------------|--------------------------|-----------|
| **Planet rendering (textured spheres)** | WGSL circle shader + CPU Lambertian | **Bevy built-in** (`Mesh` + `StandardMaterial`) | Standard 3D rendering. PBR is strictly superior. |
| **Sun glow** | WGSL glow shader (additive) + CPU radial gradient | **Bevy built-in** (bloom post-processing) | `BloomSettings` on camera + emissive material. |
| **Ray-traced spheres** | WGSL compute + Metal compute + CUDA kernel + HIP kernel | **Delete** (replace with Bevy PBR + SSAO + shadow maps) | PBR provides equivalent/better quality. RT is orthographic-only and limited to 16 spheres. |
| **Orbital trail lines** | WGSL line shader + CPU Bresenham | **Bevy plugin** (`bevy_polyline`) | GPU-accelerated wide polylines with per-vertex color. |
| **Spacetime grid** | Rust-side CPU compute + WGSL line shader + Go CPU compute | **Custom Bevy system** (dynamic mesh or compute shader) | CPU-side potential field computation + dynamic mesh update. The compute part could move to a wgpu compute shader for performance. |
| **Asteroid belt particles** | CPU-only (Go Kepler solver + pixel buffer) | **Bevy instancing** (1500 instanced quads/spheres) | Bevy handles instanced rendering automatically. Kepler solver runs as a Bevy system updating `Transform` components. |
| **Texture atlas** | Rust `TextureAtlas` (2D array, 9 layers, 2048x1024) | **Bevy `AssetServer`** (individual textures per planet) | No need for manual atlas. Bevy handles GPU texture management. |
| **Camera transforms** | Rust `Camera` + C `camera.c` + Go `viewport.go` | **Bevy `Camera3d`** + `Transform` | Unified camera system with proper perspective/ortho projection. |
| **Hardware detection** | Rust `hardware.rs` (wgpu adapter probe) | **Delete** (Bevy handles adapter selection) | Bevy/wgpu selects the best adapter automatically. Hardware tier can be queried from `RenderDevice` if needed. |
| **Pixel readback** | `MAP_READ` buffer + `poll(Wait)` | **Delete** (no readback needed) | Bevy renders directly to the window surface. No CPU readback of pixel data. This eliminates the biggest GPU performance bottleneck. |

---

## Migration Phases

### Phase A: Delete native backends (immediate, before Bevy migration)

Remove `native_gpu/` entirely:
- `native_gpu/metal/` -- raytracer.metal, renderer.m, Makefile, etc.
- `native_gpu/cuda/` -- raytracer.cu, renderer.cu, hardware.cu, Makefile
- `native_gpu/rocm/` -- raytracer.hip, renderer.hip, hardware.hip, Makefile
- `native_gpu/common/` -- rasterizer.c, camera.c, native_render.h, etc.
- `internal/ffi/render_metal.go`, `render_cuda.go`, `render_rocm.go`
- Build tags: `metal_render`, `cuda_render`, `rocm_render`
- Makefile targets: `build-metal-*`, `build-cuda-*`, `build-rocm-*`

**Savings:** ~2,254 native LOC + ~420 FFI LOC + 3 Makefiles + 6 build configurations.

### Phase B: Delete Rust wgpu renderer (during Bevy migration)

Remove `crates/render_core/` when Bevy renderer replaces it:
- All Rust rendering code (~2,890 LOC)
- `internal/ffi/render_rust.go` (~210 LOC)
- `internal/render/gpu_renderer.go` (~300 LOC)
- Build tag: `rust_render`

### Phase C: Delete Go CPU renderer (during Bevy migration)

Remove when Bevy renderer is feature-complete:
- `internal/render/renderer.go` -- replaced by Bevy ECS render systems
- `internal/render/lighting.go` -- replaced by Bevy PBR
- `internal/render/belt.go` -- replaced by Bevy instanced rendering
- `internal/render/trail_buffer.go` -- replaced by `bevy_polyline`
- `internal/render/textures.go` -- replaced by Bevy `AssetServer`
- `internal/render/cache.go` -- no longer needed (no Fyne canvas objects)
- `internal/spacetime/` -- replaced by custom Bevy system
- `internal/render/memset_cgo.go`, `memset_nocgo.go` -- no longer needed

---

## Decision Summary

| Decision | Answer |
|----------|--------|
| Keep native Metal/CUDA/ROCm ray tracers? | **No. Delete all three.** |
| Consolidate to wgpu compute? | **No. Use Bevy's built-in PBR pipeline instead.** |
| Keep Rust wgpu rasterizer? | **No. Delete when Bevy renderer is ready.** |
| Keep custom ray tracing? | **No. Bevy PBR + SSAO + bloom exceeds current RT quality.** |
| Keep Go CPU renderer? | **No. Delete when Bevy renderer is feature-complete.** |
| Custom compute shaders needed? | **Possibly for spacetime grid potential field computation. Otherwise no.** |
