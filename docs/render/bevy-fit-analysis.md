# Bevy Fit Analysis

Analysis of how each current rendering feature maps to Bevy 0.15 capabilities.

---

## Planet Spheres

**Current:** CPU path renders textured circular cutouts with Lambertian shading (`lighting.go`). GPU path draws SDF quads with equirectangular UV mapping in a WGSL fragment shader (`pipeline.rs:3-65`).

**Bevy built-in?** Yes. Bevy `Mesh` (sphere primitive) + `StandardMaterial` with `base_color_texture` provides textured spheres with PBR lighting out of the box. Bevy 0.15's `Sphere` shape generates proper UV-mapped meshes.

**Bevy advantage:** Real 3D perspective rendering with depth testing. Current system uses fake oblique projection (`x -= z * 0.5, y -= z * 0.8` in `camera.rs:84-85`). Bevy gives proper perspective/orthographic cameras with depth buffer. Planets will correctly occlude each other.

**Risk/complexity:** Low. This is Bevy's bread and butter. Material properties (albedo, roughness, metallic) can be tuned per planet. Textures can be loaded via `AssetServer`. The current 2048x1024 equirectangular textures work directly as `StandardMaterial::base_color_texture`.

---

## Sun Glow

**Current:** CPU path: radial gradient `*image.RGBA` with `alpha = (1 - distSq) * 0.4` (`lighting.go:147-181`). GPU path: Gaussian falloff fragment shader with additive blending (`pipeline.rs:67-108`).

**Bevy built-in?** Partially. Bevy 0.15 has bloom post-processing (`BloomSettings`) which creates glow around bright objects. This is more physically correct than the current approach.

**Needs custom shader?** Possibly. For exact visual match, a custom billboard quad with additive blend material could replicate the current glow. But Bevy's HDR bloom is likely superior visually.

**Bevy advantage:** HDR bloom produces halo effects automatically based on emissive intensity. No need for manual glow image generation. The sun can use `StandardMaterial { emissive: Color::WHITE * 10.0, .. }` and bloom handles the rest.

**Risk/complexity:** Low. Bloom is a standard Bevy feature. May need tuning of `BloomSettings` threshold and intensity to match desired visual.

---

## Comet Tails

**Current:** 8 gradient line segments away from Sun with decreasing alpha and stroke width (`renderer.go:477-523`). Direction computed as `comet.Position - sun.Position`. Length inversely proportional to distance from Sun.

**Bevy built-in?** No direct equivalent. Bevy has no built-in comet tail or trail effect.

**Needs custom plugin?** Yes. Options:
1. **`bevy_hanabi` particle system** -- emit particles in anti-sunward direction with decreasing alpha. Best visual quality.
2. **Custom line material** -- port the gradient line approach using `bevy_polyline` or Bevy's `Gizmos` API for debug-style lines.
3. **Billboard quad** with alpha-gradient texture -- simplest approach.

**Bevy advantage:** Particle-based tails can respond to solar wind direction, have proper 3D depth, and look significantly better than 8 flat line segments.

**Risk/complexity:** Medium. Particle system integration adds a dependency. The current implementation is very simple (~50 LOC); a particle-based version would be more complex but more visually impressive.

---

## Orbital Trails

**Current:** CPU path: Catmull-Rom interpolated Bresenham lines drawn into pixel buffer, max 200 segments per body, alpha fade from old to new (`trail_buffer.go`). GPU path: line primitives with per-vertex alpha (`ffi.rs:132-210`).

**Bevy built-in?** Not directly. Bevy's `Gizmos` API can draw lines but is intended for debug visualization, not production rendering.

**Needs custom plugin?** Yes. Options:
1. **`bevy_polyline`** -- GPU-accelerated wide polylines with per-vertex color/alpha. Best fit.
2. **Custom mesh** -- generate a tube or ribbon mesh from trail points. More work but allows width variation.
3. **Bevy `Gizmos`** -- functional for prototyping but not ideal for production (no anti-aliasing, limited styling).

**Bevy advantage:** True 3D trails with depth testing. Currently trails are 2D screen-space (Bresenham into a flat image buffer). In Bevy, trails would correctly wrap around in 3D space and be occluded by planets.

**Risk/complexity:** Low-Medium. `bevy_polyline` handles the hard part. Trail data management (ring buffer, downsampling) stays the same. Catmull-Rom interpolation can still be applied to smooth trail points before submitting to Bevy.

---

## Asteroid Belt

**Current:** 1500 visual particles (not N-body). Positions computed from Keplerian elements each frame with 5-iteration Kepler equation solver. Drawn as 1-3px dots into pixel buffer (`belt.go`). Kirkwood gaps modeled.

**Bevy built-in?** Partially. Bevy supports instanced rendering which can draw thousands of small meshes efficiently.

**Needs custom implementation?** Yes. Options:
1. **Instanced meshes** -- spawn 1500 `Mesh` entities with tiny sphere/quad shapes. Bevy handles instancing automatically for identical meshes.
2. **Point sprites** -- custom shader that renders screen-space point particles. More efficient for 1500 points.
3. **GPU compute** -- compute Kepler positions on GPU, render as instanced points. Ideal but complex.

**Bevy advantage:** True 3D belt with perspective, depth sorting, and potential for individual asteroid meshes at high zoom. Current implementation is flat 2D dots.

**Risk/complexity:** Medium. The Kepler solver needs to run each frame for 1500 particles. In Bevy, this could be a system that updates `Transform` components. At 1500 entities, instanced rendering should handle it at 60 FPS easily. The `bevy_hanabi` particle system could also work if particles can have per-particle orbital mechanics.

---

## Spacetime Curvature Grid

**Current:** Gravitational metric perturbation `h_00 = 2GM/(c^2*r)` computed at grid points. Adaptive resolution (40-150 lines). Warped grid with color gradient (purple/red/orange). Cached when camera unchanged. CPU: `spacetime/spacetime.go`. GPU: `crates/render_core/src/spacetime.rs`.

**Bevy built-in?** No. This is a highly specialized physics visualization.

**Needs custom implementation?** Yes. Options:
1. **Dynamic mesh** -- generate a deformed plane mesh each frame based on gravitational potential. Update `Mesh` vertex positions. Color via vertex colors or custom shader.
2. **Custom material** -- shader that displaces grid vertices based on a potential texture computed on CPU/GPU.
3. **Bevy `Gizmos`** -- grid lines via Gizmos API for quick prototyping.

**Bevy advantage:** Proper 3D grid deformation with depth. Current system is 2D screen-space displacement. In Bevy, the spacetime grid could deform in 3D, visible from different camera angles.

**Risk/complexity:** Medium-High. The potential field computation is the bottleneck (80x80 to 120x120 grid, iterating over all bodies). This stays on CPU or moves to a compute shader. Mesh generation/update each frame for up to 120x120 grid is feasible but not trivial. The color gradient mapping needs a custom vertex-color material or a heat-map shader.

---

## Labels / Text Overlays

**Current:** Fyne `canvas.Text` objects positioned near each body. GPU mode uses CPU overlay compositing via `CreateLabelOverlay()` (`renderer.go:417-475`).

**Bevy built-in?** Yes. Bevy has 2D text rendering via `Text2d` component and UI text via `bevy_ui`. For world-space labels, `Text2d` with `Billboard` behavior (always face camera) works.

**Bevy advantage:** Text rendered in GPU pipeline alongside scene. No separate CPU overlay compositing needed. Bevy's text system supports fonts, sizing, colors natively.

**Risk/complexity:** Low. Standard Bevy feature. May want to use a billboard plugin or custom system to keep labels facing the camera and scaled appropriately regardless of zoom.

---

## Skybox

**Current:** CPU path stretches a milky way JPEG to fill the background (`renderer.go:102-113`). GPU path uses a solid dark blue clear color (no skybox).

**Bevy built-in?** Yes. Bevy 0.15 has `Skybox` component that renders a cubemap or equirectangular environment map. Also supports `EnvironmentMapLight` for image-based lighting.

**Bevy advantage:** Proper cubemap skybox with camera rotation. Current implementation stretches a flat image, losing parallax. Bevy's skybox rotates with the camera, providing spatial context.

**Risk/complexity:** Low. Need to convert the current equirectangular milky way image to a cubemap (or use Bevy's equirectangular skybox support). This is a well-documented Bevy feature.

---

## Distance Measurement

**Current:** Yellow line between two selected bodies + text displaying AU, km, light-minutes (`renderer.go:292-319`). GPU path renders line via line pipeline, text via CPU overlay.

**Bevy built-in?** Partially. `Gizmos::line()` for the line segment. `Text2d` for the label. Or use Bevy UI for overlay text.

**Bevy advantage:** Line is properly 3D (connects world-space positions). Text can be a 2D overlay anchored to the midpoint screen position.

**Risk/complexity:** Low. Simple feature in any engine.

---

## Trajectory Overlay

**Current:** Launch trajectory drawn as color-gradient `canvas.Line` segments, progress from green to red (`renderer.go:365-414`). In GPU mode, trajectory is appended as an extra trail.

**Bevy built-in?** No direct equivalent, but same solution as orbital trails.

**Needs custom?** Same as trails -- `bevy_polyline` or custom line mesh with per-vertex color.

**Risk/complexity:** Low. Same approach as orbital trails, just with a different color gradient.

---

## Lighting

**Current:** CPU: Lambertian diffuse (`dot(normal, lightDir)`) with 0.15 ambient, parallelized per-row (`lighting.go:30-143`). GPU raster: no explicit lighting (texture colors passed through). GPU RT: full lighting with shadows, AO (4 samples), glossy reflections (`raytracer.rs:83-213`).

**Bevy built-in?** Yes. Bevy's PBR pipeline includes:
- Directional/point/spot lights with shadow mapping
- Ambient lighting (configurable)
- Physically-based materials (roughness, metallic, emissive)
- Screen-space ambient occlusion (SSAO) via `ScreenSpaceAmbientOcclusion`

**Bevy advantage:** Significantly more advanced than the current Lambertian model. Shadow mapping gives proper hard/soft shadows. PBR materials enable metallic asteroid surfaces, gas giant atmosphere effects, etc. SSAO provides screen-space ambient occlusion without the per-pixel ray tracing cost.

**Risk/complexity:** Low for basic lighting. The Sun would be a `PointLight` with high intensity and range. Each planet gets `StandardMaterial` with configurable roughness/metallic. Shadow mapping is built-in. For the RT-equivalent quality (per-pixel AO, glossy reflections), Bevy's SSR and SSAO plugins provide similar results at lower cost.

---

## Textures

**Current:** CPU: dynamic directory discovery, JPEG/PNG loading, circular cutout masking, nearest-neighbor resize, per-`(name, diameter)` cache (`textures.go`). GPU: texture atlas with 9 layers at 2048x1024, Lanczos3 resize, equirectangular UV mapping in shader (`textures.rs`).

**Bevy built-in?** Yes. Bevy's `AssetServer` handles texture loading with caching. `StandardMaterial::base_color_texture` accepts any `Handle<Image>`.

**Bevy advantage:** Automatic async asset loading with progress tracking. Mipmapping support. GPU-resident textures (no CPU-side circular masking needed -- the sphere mesh handles UV mapping). Hot-reloading in development mode.

**Risk/complexity:** Low. Textures load via `asset_server.load("textures/earth/albedo.jpg")`. No manual circular cutout, resize, or atlas packing needed. Bevy's PBR pipeline can also use normal maps, roughness maps, etc. for enhanced visual quality.

---

## Ray Tracing

**Current:** WGSL compute shader (`raytracer.rs`), Metal compute kernel (`raytracer.metal`), CUDA kernel (`raytracer.cu`), HIP kernel (`raytracer.hip`). All implement identical algorithm: orthographic sphere intersection, shadow rays, 4-sample AO, glossy reflections (material==2), progressive accumulation, sRGB gamma, texture atlas sampling.

**Bevy built-in?** No. Bevy does not have built-in ray tracing.

**Needs custom?** Yes, if RT is desired. Options:
1. **Keep as compute shader plugin** -- Bevy supports custom render nodes and compute shaders. The WGSL RT shader can be integrated as a custom render pass.
2. **Replace with Bevy PBR + post-processing** -- Bevy's PBR with shadow mapping + SSAO + bloom may provide visual quality comparable to the current RT at much lower cost. The current RT is orthographic and limited to 16 spheres.
3. **Future: Bevy RT extensions** -- Bevy has experimental ray tracing support tracking. Not production-ready in 0.15.

**Bevy advantage:** Bevy's standard rendering pipeline (PBR + shadow maps + SSAO + bloom) likely exceeds the visual quality of the current RT implementation, which is limited by:
- Orthographic projection (no perspective)
- Max 16 spheres
- 4 AO samples (noisy, requires accumulation)
- No environment mapping or IBL

**Risk/complexity:** Low if replacing RT with PBR pipeline (delete RT code, rely on Bevy rendering). High if porting RT compute shader to Bevy render node (custom `RenderApp` stage, bind groups, pipeline management).

**Recommendation:** Delete the custom RT system and rely on Bevy's PBR pipeline. If advanced RT is needed later, it can be added as a custom render plugin when Bevy's RT ecosystem matures.

---

## Summary Table

| Feature | Bevy Built-in? | Custom Needed? | Bevy Advantage | Risk |
|---------|---------------|----------------|----------------|------|
| Planet spheres | Yes (`Mesh` + `StandardMaterial`) | No | True 3D, depth, PBR | Low |
| Sun glow | Yes (bloom) | Tuning only | HDR bloom, automatic | Low |
| Comet tails | No | Particle plugin | 3D particles, better visual | Medium |
| Orbital trails | No | `bevy_polyline` or similar | 3D trails with depth | Low-Medium |
| Asteroid belt | Partial (instancing) | Kepler solver system | 3D particles, perspective | Medium |
| Spacetime grid | No | Dynamic mesh or gizmos | 3D deformation | Medium-High |
| Labels | Yes (`Text2d`) | Billboard system | GPU-rendered text | Low |
| Skybox | Yes (`Skybox`) | No | Cubemap rotation | Low |
| Distance line | Yes (`Gizmos`) | No | 3D line | Low |
| Trajectory | No | Same as trails | 3D path | Low |
| Lighting | Yes (PBR) | No | Shadows, SSAO, PBR | Low |
| Textures | Yes (`AssetServer`) | No | Async, mipmaps, hot-reload | Low |
| Ray tracing | No | Delete or custom node | PBR pipeline is superior | Low (delete) / High (port) |
