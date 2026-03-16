# Rendering Migration Table

Every rendering component, shader, texture system, lighting model, and visual feature.

**Keep** = use as-is in Bevy project | **Wrap** = thin adapter around existing code | **Rewrite** = new implementation in Bevy | **Delete** = remove entirely

---

| Current Component | Language/Backend | Purpose | Decision | Bevy Migration Notes | Effort | Phase |
|---|---|---|---|---|---|---|
| **Go CPU Renderer** | | | | | | |
| `internal/render/renderer.go` | Go / Fyne | Main CPU render loop: composites skybox, grid, belt, trails, sun, planets, labels, trajectory, distance line into `fyne.Container` | **Delete** | Replace with Bevy ECS systems: `render_planets_system`, `render_trails_system`, `render_belt_system`, etc. Each visual feature becomes a separate system operating on components. | Large | 3 |
| `internal/render/cache.go` | Go / Fyne | Object pool for `canvas.Circle/Line/Text/Image` | **Delete** | Unnecessary in Bevy. ECS entities persist across frames. Bevy handles GPU resource pooling internally. | Trivial | 3 |
| `internal/render/lighting.go` (Lambertian) | Go | Per-pixel Lambertian diffuse shading on circular planet textures. Parallelized across CPU cores. | **Delete** | Bevy PBR pipeline handles diffuse lighting via `PointLight` + `StandardMaterial`. No custom lighting code needed. | None | 3 |
| `internal/render/lighting.go` (Sun glow) | Go | Radial gradient image for sun glow, cached by diameter. | **Delete** | Replace with `BloomSettings` on camera + `StandardMaterial { emissive }` on Sun entity. Bloom handles glow automatically. | Trivial | 3 |
| `internal/render/belt.go` | Go | 1500 Kepler-solved asteroid belt particles drawn as 1-3px dots into image buffer. | **Rewrite** | Bevy system that updates `Transform` for 1500 instanced entities. Kepler solver logic (`beltParticlePosition()`) ports directly. Use `Mesh::from(Sphere)` or billboard quads with instancing. | Medium | 3 |
| `internal/render/trail_buffer.go` | Go | Orbital trails: Catmull-Rom interpolation + Bresenham line drawing into pixel buffer. | **Rewrite** | Use `bevy_polyline` plugin. Trail data management (ring buffer, downsampling) stays similar. Catmull-Rom interpolation can still smooth trail points. Lines rendered as GPU polylines with per-vertex alpha. | Medium | 3 |
| `internal/render/textures.go` (LoadAll) | Go | Dynamic texture directory discovery, JPEG/PNG loading, per-planet caching. | **Rewrite** | Bevy `AssetServer::load()` handles async texture loading with caching. Directory discovery logic can remain as asset path resolution. No manual image decoding needed. | Small | 3 |
| `internal/render/textures.go` (circular mask) | Go | Nearest-neighbor resize + circular mask for planet textures. | **Delete** | Unnecessary. Bevy sphere meshes have proper UV mapping; textures are applied directly as `StandardMaterial::base_color_texture`. No circular cutout needed. | None | 3 |
| `internal/render/textures.go` (irregular asteroid) | Go | Procedural 8-lobe potato shape with deterministic RNG for asteroids. | **Rewrite** | Port to a Bevy system that generates procedural `Mesh` with perturbed sphere vertices, or use a pre-baked low-poly asteroid mesh asset. Alternatively, apply a procedural noise shader to sphere meshes. | Medium | 4 |
| `internal/render/textures.go` (skybox) | Go | Loads milky_way.jpg for background. | **Rewrite** | Bevy `Skybox` component with equirectangular or cubemap image. May need to convert current equirectangular image to cubemap format. | Small | 2 |
| `internal/render/memset_cgo.go` / `memset_nocgo.go` | Go / C | Fast buffer clearing for trail/belt image buffers. | **Delete** | No pixel buffers to clear. Bevy manages GPU framebuffers. | None | 3 |
| `internal/render/renderer.go` (comet tails) | Go / Fyne | 8 gradient line segments away from Sun with decreasing alpha/width. | **Rewrite** | Particle effect via `bevy_hanabi` (anti-sunward particle emission) or custom billboard material with gradient texture. | Medium | 4 |
| `internal/render/renderer.go` (trajectory) | Go / Fyne | Color-gradient line segments for launch trajectory overlay. | **Rewrite** | Same approach as orbital trails: `bevy_polyline` with per-vertex color gradient (green to red by progress). | Small | 3 |
| `internal/render/renderer.go` (distance line) | Go / Fyne | Yellow line between 2 bodies + AU/km/light-min text. | **Rewrite** | `Gizmos::line_2d()` or `bevy_polyline` for the line. Bevy UI `Text` for the measurement overlay, positioned at screen midpoint. | Small | 3 |
| `internal/render/renderer.go` (labels) | Go / Fyne | `canvas.Text` per body, positioned at body screen coords. | **Rewrite** | Bevy `Text2d` components with billboard behavior (face camera). Or Bevy UI overlay with world-to-screen coordinate mapping. | Small | 2 |
| `internal/render/renderer.go` (vehicle marker) | Go / Fyne | Green dot at launch vehicle position. | **Rewrite** | Bevy entity with small sphere `Mesh` + green `StandardMaterial`. Position updated by playback system. | Trivial | 3 |
| **Go GPU Renderer** | | | | | | |
| `internal/render/gpu_renderer.go` | Go / Fyne | GPU render path: marshals scene data, calls FFI, wraps result in `canvas.Raster`. | **Delete** | Bevy manages the full render pipeline. No FFI, no manual marshaling, no pixel readback. Scene data flows through ECS components. | None | 2 |
| `internal/render/gpu_renderer_noop.go` | Go | Stub `GPURenderer` when no GPU tag is set. | **Delete** | No build-tag-based renderer selection in Bevy. Single unified renderer. | None | 2 |
| **Go FFI Bridges** | | | | | | |
| `internal/ffi/render_rust.go` | Go / CGO | CGO wrapper calling `librender_core` Rust library. | **Delete** | No FFI boundary. Bevy renderer is pure Rust. | None | 2 |
| `internal/ffi/render_metal.go` | Go / CGO | CGO wrapper calling `libnative_render_metal`. | **Delete** | wgpu handles Metal automatically. | None | 1 |
| `internal/ffi/render_cuda.go` | Go / CGO | CGO wrapper calling `libnative_render_cuda`. | **Delete** | wgpu handles Vulkan on NVIDIA. | None | 1 |
| `internal/ffi/render_rocm.go` | Go / CGO | CGO wrapper calling `libnative_render_rocm`. | **Delete** | wgpu handles Vulkan on AMD. | None | 1 |
| `internal/ffi/render_noop.go` | Go | Stub when no render tag set. | **Delete** | No build tags for rendering. | None | 2 |
| **Rust wgpu Renderer** | | | | | | |
| `crates/render_core/src/renderer.rs` | Rust | Main orchestrator: offscreen render, readback, RT dispatch. | **Delete** | Bevy's render graph replaces this entirely. No offscreen rendering or pixel readback needed. | None | 2 |
| `crates/render_core/src/pipeline.rs` (circle shader) | Rust / WGSL | SDF textured circle rendering with orthographic projection. | **Delete** | Bevy `Mesh` + `StandardMaterial` replaces this. True 3D spheres with PBR. | None | 2 |
| `crates/render_core/src/pipeline.rs` (glow shader) | Rust / WGSL | Gaussian glow with additive blending. | **Delete** | Bevy bloom post-processing. | None | 2 |
| `crates/render_core/src/pipeline.rs` (line shader) | Rust / WGSL | Colored line primitives for trails/grid/distance. | **Delete** | `bevy_polyline` / `Gizmos`. | None | 2 |
| `crates/render_core/src/raytracer.rs` | Rust / WGSL compute | Sphere ray tracer with shadows, AO, glossy, progressive accumulation. | **Delete** | Bevy PBR + SSAO + shadow maps + bloom provides equivalent or better quality. | None | 2 |
| `crates/render_core/src/shapes.rs` | Rust | `CircleVertex` / `LineVertex` structs, `make_circle_vertices()`. | **Delete** | Bevy mesh primitives replace this. | None | 2 |
| `crates/render_core/src/textures.rs` | Rust | Texture atlas: 9-layer 2D array, Lanczos3 resize, fallback white. | **Delete** | Bevy `AssetServer` handles per-planet textures individually. No atlas needed. | None | 2 |
| `crates/render_core/src/camera.rs` | Rust | Camera: world-to-screen projection, orthographic matrix. | **Delete** | Bevy `Camera3d` / `Camera2d` + `Transform` + projection components. | None | 2 |
| `crates/render_core/src/hardware.rs` | Rust | GPU detection: adapter probe, vendor ID mapping, tier classification. | **Delete** | Bevy/wgpu handles adapter selection. Tier classification can query `RenderDevice` if needed. | None | 2 |
| `crates/render_core/src/spacetime.rs` | Rust | Gravitational potential field computation + warped grid line generation. | **Rewrite** | Port to a Bevy system. Potential field computation stays as CPU math (or moves to compute shader). Grid rendered as dynamic `Mesh` with vertex colors. | Medium | 3 |
| `crates/render_core/src/ffi.rs` | Rust | C FFI exports for Go interop. | **Delete** | No FFI. Pure Rust Bevy plugin. | None | 2 |
| `crates/render_core/Cargo.toml` | Rust | Build config: cdylib, wgpu 24, bytemuck, pollster, image. | **Delete** | Replaced by Bevy workspace `Cargo.toml` with Bevy dependencies. | None | 2 |
| **Native GPU Backends** | | | | | | |
| `native_gpu/metal/raytracer.metal` | Metal MSL | Metal compute kernel: sphere RT with threadgroup memory. | **Delete** | wgpu/Bevy handles Metal natively. No hand-written MSL needed. | None | 1 |
| `native_gpu/metal/renderer.m` (+ other .m files) | Objective-C | Metal host code: device setup, buffer management, kernel dispatch. | **Delete** | Bevy manages all GPU resources. | None | 1 |
| `native_gpu/cuda/raytracer.cu` | CUDA | CUDA kernel: sphere RT with shared memory. | **Delete** | wgpu/Vulkan works on NVIDIA GPUs. | None | 1 |
| `native_gpu/cuda/renderer.cu` | CUDA/C | CUDA host: device memory, texture objects, kernel dispatch, readback. | **Delete** | Bevy manages all GPU resources. | None | 1 |
| `native_gpu/cuda/hardware.cu` | CUDA | NVIDIA GPU detection. | **Delete** | Bevy/wgpu handles detection. | None | 1 |
| `native_gpu/rocm/raytracer.hip` | HIP | HIP kernel: sphere RT (CUDA port). | **Delete** | wgpu/Vulkan works on AMD GPUs. | None | 1 |
| `native_gpu/rocm/renderer.hip` | HIP/C | HIP host: device memory, texture objects, kernel dispatch. | **Delete** | Bevy manages all GPU resources. | None | 1 |
| `native_gpu/rocm/hardware.hip` | HIP | AMD GPU detection. | **Delete** | Bevy/wgpu handles detection. | None | 1 |
| `native_gpu/common/rasterizer.c` | C | CPU Bresenham line/circle drawing with alpha blend. | **Delete** | Bevy handles all rasterization. | None | 1 |
| `native_gpu/common/camera.c` | C | World-to-screen projection (port of Rust camera). | **Delete** | Bevy `Camera` + `Transform`. | None | 1 |
| `native_gpu/common/native_render.h` | C | Unified C API for all native backends. | **Delete** | No C API needed. | None | 1 |
| **Spacetime Visualization** | | | | | | |
| `internal/spacetime/spacetime.go` | Go / Fyne | CPU spacetime grid: h_00 potential, adaptive resolution, color gradient, warped lines. | **Rewrite** | Bevy system: compute potential field each frame (CPU or compute shader), generate dynamic `Mesh` with warped vertex positions and vertex colors. Render with unlit material or custom shader for heat-map coloring. | Medium | 3 |
| **Viewport / Camera** | | | | | | |
| `internal/viewport/viewport.go` | Go | Camera state: zoom, pan, rotation, follow body, WorldToScreen projection, auto-fit. | **Rewrite** | Bevy `Camera3d` + custom camera controller system. Zoom = camera Z distance or orthographic scale. Pan = camera XY translation. Rotation = camera transform rotation. Follow = parent entity or lerp to target. Auto-fit = compute bounding sphere of all bodies, set camera to fit. | Medium | 2 |

---

## Phase Definitions

| Phase | Name | Description |
|-------|------|-------------|
| 1 | **Pre-migration cleanup** | Delete native GPU backends (Metal/CUDA/ROCm) and their FFI glue. These are dead weight with zero impact on the default build. |
| 2 | **Core Bevy renderer** | Implement basic Bevy scene: camera, planet spheres with textures, sun with bloom, labels, skybox. Delete Rust wgpu renderer and Go GPU renderer. |
| 3 | **Feature parity** | Implement remaining visual features in Bevy: orbital trails, asteroid belt, spacetime grid, distance measurement, launch trajectory, vehicle marker. Delete Go CPU renderer. |
| 4 | **Visual enhancements** | Polish features that benefit from Bevy's capabilities: particle-based comet tails, procedural asteroid meshes, PBR material tuning, environment mapping. |

---

## Effort Key

| Effort | Description |
|--------|-------------|
| None | Just delete the file/component |
| Trivial | < 1 hour, straightforward mapping |
| Small | 1-4 hours, well-understood Bevy pattern |
| Medium | 1-3 days, requires design decisions or custom systems |
| Large | 1-2 weeks, complex system with multiple interacting parts |
