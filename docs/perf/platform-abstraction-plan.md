# Platform Abstraction Plan

How Bevy/wgpu replaces the current multi-backend GPU architecture, what is lost, what is gained, and how platform-specific optimizations are handled.

---

## How Bevy/wgpu Handles Metal/Vulkan/DX12 Natively

### wgpu Backend Selection

Bevy uses wgpu as its GPU abstraction layer. wgpu compiles to native GPU APIs at runtime:

| Platform | Primary Backend | Fallback | Selection |
|----------|----------------|----------|-----------|
| macOS | Metal | N/A (Metal-only on Apple) | Automatic |
| Linux | Vulkan | OpenGL (via `wgpu::Backends::GL`) | Automatic, prefers Vulkan |
| Windows | DX12 | Vulkan, then DX11/GL | Automatic, prefers DX12 |
| iOS | Metal | N/A | Automatic |
| Android | Vulkan | OpenGL ES | Automatic |
| Web | WebGPU | WebGL2 (via `wgpu::Backends::BROWSER_WEBGPU`) | Automatic |

**No build tags, no compile-time backend selection, no platform-specific libraries.** wgpu probes available backends at startup and selects the best one. Users never need to choose.

### What wgpu Provides vs What Native APIs Provide

| Capability | wgpu | Metal (native) | CUDA (native) | ROCm (native) |
|-----------|------|----------------|---------------|----------------|
| Compute shaders | WGSL compute | MSL compute | CUDA kernels | HIP kernels |
| Workgroup shared memory | `var<workgroup>` | `threadgroup` | `__shared__` | `__shared__` |
| Storage buffers | `var<storage>` | `device` buffer | Global memory | Global memory |
| Texture sampling | `textureSample*` | `texture.sample` | `tex2DLayered` | `tex2DLayered` |
| Texture arrays | `texture_2d_array` | `texture2d_array` | Layered CUDA array | Layered HIP array |
| Synchronization barriers | `workgroupBarrier()` | `threadgroup_barrier` | `__syncthreads()` | `__syncthreads()` |
| RNG (PCG hash) | Manual (same code) | Manual (same code) | Manual (same code) | Manual (same code) |

Every feature used by the current native backends is available in wgpu. The WGSL compute shader in `raytracer.rs:4-213` is functionally identical to the Metal, CUDA, and HIP kernels.

### Bevy's GPU Pipeline Architecture

Bevy 0.15's render pipeline provides:

1. **Render Graph:** Directed acyclic graph of render passes. Standard passes: `MainOpaquePass3dNode`, `MainTransparentPass3dNode`, `BloomNode`, `TonemappingNode`, `FxaaNode`. Custom compute passes can be inserted at any point.

2. **Extract-Prepare-Queue-Render pattern:**
   - `ExtractSchedule`: copies data from main world to render world (runs in parallel with next frame's simulation)
   - `PrepareSchedule`: creates GPU resources (buffers, bind groups)
   - `QueueSchedule`: sorts and batches draw calls
   - `RenderSchedule`: executes GPU commands

3. **Automatic instancing:** Entities with identical `Mesh` + `Material` are automatically batched into instanced draw calls. This is how 1,500 belt particles render efficiently.

4. **Asset pipeline:** `AssetServer` handles async loading, GPU upload, format conversion, and caching for textures, meshes, shaders, and other assets.

---

## What is Lost by Dropping Native Metal/CUDA/ROCm

### Metal-Specific Features Not Available via wgpu

| Feature | Description | Used by Current Code? | Impact |
|---------|-------------|----------------------|--------|
| Metal Performance Shaders (MPS) | Apple's optimized kernel library (matrix multiply, image processing, neural networks) | **No** | None |
| Metal ray tracing (MPSRayIntersector) | Hardware-accelerated BVH traversal on Apple Silicon | **No** | None -- current RT uses brute-force linear search |
| Tile shading / imageblocks | On-chip tile memory for deferred rendering on Apple GPUs | **No** | None |
| Mesh shaders (Metal 3) | Object/mesh shader pipeline on A17+/M3+ | **No** | None |
| `threadgroup` shared memory | Fast on-chip memory within a threadgroup | **Yes** (`raytracer.metal:99`) | Minimal -- 16 spheres (768 bytes) fits in L1 cache without shared memory. wgpu `var<workgroup>` provides equivalent functionality if needed. |

### CUDA-Specific Features Not Available via wgpu

| Feature | Description | Used by Current Code? | Impact |
|---------|-------------|----------------------|--------|
| Tensor cores (WMMA) | Matrix multiply-accumulate for AI/HPC | **No** | None |
| RT cores (OptiX) | Hardware BVH traversal on RTX GPUs | **No** | None -- current RT uses brute-force |
| CUDA Dynamic Parallelism | Kernels launching sub-kernels | **No** | None |
| Cooperative Groups | Flexible thread synchronization | **No** | None |
| `__shared__` memory | On-chip shared memory | **Yes** (`raytracer.cu:114`) | Minimal -- same as Metal analysis above |
| `tex2DLayered` hardware filtering | Hardware-filtered layered texture reads | **Yes** (`raytracer.cu:162`) | None -- wgpu `textureSampleLevel` provides identical hardware filtering on NVIDIA via Vulkan |
| Unified Memory | Managed memory spanning CPU/GPU | **No** | None |
| CUDA Streams | Concurrent kernel execution | **No** (uses `cudaDeviceSynchronize`) | None |

### ROCm-Specific Features Not Available via wgpu

| Feature | Description | Used by Current Code? | Impact |
|---------|-------------|----------------------|--------|
| Matrix cores (MFMA) | AMD's matrix multiply units | **No** | None |
| HIP graphs | GPU work graphs for reduced launch overhead | **No** | None |
| All other CUDA-equivalent features | HIP is a 1:1 port of CUDA API | Same as CUDA analysis | Same |

### Summary of Losses

**Nothing of practical value is lost.** The native backends use only:
1. Basic compute dispatch (wgpu provides this)
2. Shared/threadgroup memory for 768 bytes (fits in L1 cache without it; wgpu `var<workgroup>` available if needed)
3. Hardware-filtered texture sampling (wgpu provides this identically)
4. Standard math functions (sin, cos, sqrt, etc.) (WGSL provides these)

No CUDA tensor cores, no Metal MPS, no hardware ray tracing, no platform-specific memory models, and no advanced synchronization features are used by any backend.

---

## What is Gained

### Single Codebase

| Metric | Before (5 backends) | After (Bevy/wgpu) |
|--------|---------------------|--------------------|
| Shader languages | 4 (WGSL + MSL + CUDA + HIP) | 1 (WGSL, or 0 if using Bevy PBR) |
| Host code implementations | 5 (Rust renderer + Metal .m + CUDA .cu + ROCm .hip + Go CPU) | 1 (Bevy systems in Rust) |
| FFI bridge files | 5 (render_rust.go + render_metal.go + render_cuda.go + render_rocm.go + render_noop.go) | 0 |
| C API headers | 2 (render_core.h + native_render.h) | 0 |
| Build tag configurations | 6 (rust_render, metal_render, cuda_render, rocm_render, no tag, cgo) | 1 (cargo build) |
| Makefiles for GPU libraries | 3 (Metal, CUDA, ROCm) | 0 |
| Total rendering LOC to maintain | ~6,694 (Rust 2,890 + native 2,254 + Go CPU 1,550) | ~500-1,000 (Bevy systems + plugins) |

### Bevy Ecosystem

Bevy provides out-of-the-box capabilities that the current codebase implements manually:

| Capability | Current Implementation | Bevy Equivalent |
|-----------|----------------------|-----------------|
| PBR lighting | 73 LOC Lambertian in Go (`lighting.go:30-143`) | `StandardMaterial` + `PointLight` |
| Sun glow | 34 LOC radial gradient (`lighting.go:147-181`) | `BloomSettings` + emissive material |
| Shadow casting | 213 LOC RT shadow rays (WGSL) | `DirectionalLight { shadows_enabled: true }` |
| Ambient occlusion | 20 LOC in each RT shader (x4 backends) | `ScreenSpaceAmbientOcclusion` |
| Texture loading | 100+ LOC Go async loader (`textures.go`) | `asset_server.load("earth/albedo.jpg")` |
| Texture atlas | 210 LOC Rust atlas builder (`textures.rs`) | Not needed (per-material textures) |
| Camera projection | 80 LOC Go viewport (`viewport.go`) + 125 LOC Rust camera (`camera.rs`) + 76 LOC C camera (`camera.c`) | `Camera3d` + `Transform` |
| Object pooling | 100+ LOC Go cache (`cache.go`) | ECS entities persist across frames |
| Buffer management | 484 LOC CUDA host code (`renderer.cu`) | Bevy render resource system |
| Hardware detection | 127 LOC Rust (`hardware.rs`) + 72 LOC CUDA (`hardware.cu`) + 56 LOC ROCm (`hardware.hip`) | `RenderDevice::features()` / `RenderDevice::limits()` |

### Automatic Platform Targeting

Users currently must:
1. Install platform-specific toolchains (Xcode for Metal, CUDA toolkit for NVIDIA, ROCm for AMD)
2. Select the correct build tag (`metal_render`, `cuda_render`, `rocm_render`)
3. Build the native GPU library (`make build-metal-lib`, etc.)
4. Link the Go binary against the correct library

With Bevy: `cargo run -p solar_sim_bevy`. That is the entire build and run process on every platform. wgpu selects the optimal GPU backend at runtime.

### Eliminated Bottleneck: Pixel Readback

The current GPU renderers must read pixels back to CPU memory every frame because the Fyne GUI framework cannot display GPU-rendered content directly:

```
GPU renders to offscreen texture -> MAP_READ buffer -> poll(Wait) -> copy to image.RGBA -> Fyne canvas.Raster
```

This CPU-GPU sync point costs 1-5 ms per frame at 1080p. Bevy renders directly to the window surface, eliminating this entirely. This single change provides the largest performance improvement in the migration.

---

## Platform Support Matrix

### Current Support

| Platform | CPU Renderer | Rust wgpu | Metal Native | CUDA Native | ROCm Native |
|----------|-------------|-----------|--------------|-------------|-------------|
| macOS (Apple Silicon) | Yes | Yes (Metal) | Yes | No | No |
| macOS (Intel) | Yes | Yes (Metal) | Yes | No | No |
| Linux (NVIDIA) | Yes | Yes (Vulkan) | No | Yes | No |
| Linux (AMD) | Yes | Yes (Vulkan) | No | No | Yes |
| Linux (Intel) | Yes | Yes (Vulkan) | No | No | No |
| Windows (NVIDIA) | Yes | Yes (DX12/Vulkan) | No | Yes* | No |
| Windows (AMD) | Yes | Yes (DX12/Vulkan) | No | No | No |
| Windows (Intel) | Yes | Yes (DX12/Vulkan) | No | No | No |

*CUDA on Windows requires additional build configuration.

### Bevy Support (Post-Migration)

| Platform | GPU Backend | Status | Notes |
|----------|-----------|--------|-------|
| macOS (Apple Silicon) | Metal | Full support | wgpu Metal backend is mature |
| macOS (Intel) | Metal | Full support | Same as above |
| Linux (NVIDIA) | Vulkan | Full support | Vulkan drivers well-maintained by NVIDIA |
| Linux (AMD) | Vulkan | Full support | Mesa RADV driver is excellent |
| Linux (Intel) | Vulkan | Full support | Intel ANV driver is mature |
| Windows (NVIDIA) | DX12 or Vulkan | Full support | wgpu prefers DX12, falls back to Vulkan |
| Windows (AMD) | DX12 or Vulkan | Full support | Same as above |
| Windows (Intel) | DX12 or Vulkan | Full support | Same as above |
| Web (WebGPU) | WebGPU | Possible (future) | Bevy has experimental web support |
| iOS | Metal | Possible (future) | Bevy supports iOS via Metal |
| Android | Vulkan | Possible (future) | Bevy supports Android via Vulkan |

**Platform coverage is identical or better** after migration. Every platform currently supported remains supported. Web, iOS, and Android become theoretically possible.

---

## Handling Platform-Specific Optimizations in Bevy

### Tier-Based Quality Scaling

The current `hardware.rs:105-120` classifies GPUs into High/Medium/Low tiers. This concept translates to Bevy:

```rust
fn configure_rendering_quality(
    render_device: Res<RenderDevice>,
    mut bloom_settings: Query<&mut BloomSettings>,
    mut ssao_settings: Query<&mut ScreenSpaceAmbientOcclusion>,
) {
    let limits = render_device.limits();
    let tier = classify_gpu_tier(&limits);

    match tier {
        GpuTier::High => {
            // Full quality: bloom, SSAO, shadow cascades, MSAA 4x
        }
        GpuTier::Medium => {
            // Reduced: simpler bloom, no SSAO, fewer shadow cascades, MSAA 2x
        }
        GpuTier::Low => {
            // Minimal: no bloom, no SSAO, no shadows, no MSAA
        }
    }
}
```

GPU tier can be inferred from `RenderDevice::limits()`:
- `max_texture_dimension_2d >= 8192` + discrete GPU -> High
- `max_texture_dimension_2d >= 4096` -> Medium
- Otherwise -> Low

### wgpu Feature Detection

If platform-specific compute features are needed in the future, wgpu exposes feature flags:

| wgpu Feature | Metal | Vulkan | DX12 | Use Case |
|-------------|-------|--------|------|----------|
| `PUSH_CONSTANTS` | Yes | Yes | Limited | Small uniform updates without buffer allocation |
| `TEXTURE_COMPRESSION_BC` | No | Yes | Yes | Compressed planet textures (smaller VRAM) |
| `TEXTURE_COMPRESSION_ASTC_LDR` | Yes (Apple) | Some | No | Compressed textures on Apple/ARM |
| `SHADER_F16` | Yes (Apple Silicon) | Some | Some | Half-precision math for belt particles |
| `TIMESTAMP_QUERY` | Yes | Yes | Yes | GPU profiling |

These can be queried at runtime to enable optional optimizations:

```rust
if render_device.features().contains(wgpu::Features::PUSH_CONSTANTS) {
    // Use push constants for per-frame camera uniform (avoids buffer upload)
}
```

### Recommendation

1. **Do not write platform-specific code.** Bevy/wgpu abstracts all GPU differences. Writing Metal-specific or Vulkan-specific code defeats the purpose of the migration.

2. **Use tier-based quality scaling** to adjust rendering quality based on GPU capability. This is the only "platform-specific" code needed, and it operates at the wgpu abstraction level (feature flags and limits), not at the native API level.

3. **Profile before optimizing.** The current native backends' only optimization (threadgroup shared memory for 16 spheres) provides zero measurable benefit. Do not add platform-specific optimizations without profiling data showing a bottleneck.

4. **Leverage Bevy's built-in adaptive quality** where available. Bevy's shadow mapping automatically adjusts cascade count based on light configuration. Bloom quality scales with the number of downsample passes. These are controlled via standard Bevy components, not custom GPU code.
