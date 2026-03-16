# Current Rendering Pipeline

## Overview

The simulator supports three rendering paths, selected at build time via tags:

1. **CPU rendering (default)** — Go-only, outputs Fyne canvas objects
2. **GPU rendering (Rust wgpu)** — `rust_render` tag, outputs pixel buffer via `canvas.Raster`
3. **Native GPU rendering** — `metal_render`/`cuda_render`/`rocm_render` tags, same FFI interface as Rust wgpu

All paths share the same physics snapshot mechanism and viewport transform.

---

## CPU Rendering Path

**Entry:** `Renderer.CreateCanvasFromSnapshot()` in `internal/render/renderer.go:92`

### Frame production sequence

```
1. Cache.Reset()                    — Reset object pool indices
2. Viewport.TakeSnapshot()          — Single RLock, precompute trig
3. Skybox or background rectangle
4. Spacetime grid (if enabled)      — SpacetimeRenderer.RenderGrid()
5. [PARALLEL] Belt image            — BeltRenderer.RenderToImage()
6. [PARALLEL] Trail image           — TrailBuffer.Render()
7. Wait for belt + trail goroutines
8. Sun glow image + Sun body (textured or circle)
9. For each planet:
   a. Compute display radius (max of fixed pixels, physical radius at zoom)
   b. Asteroids: irregular procedural image
   c. Planets/Moons: textured + Lambertian shading (cached)
   d. Fallback: solid color circle
   e. Comet tail (gradient line segments away from Sun)
   f. Label text
10. Launch trajectory (colored line segments)
11. Launch vehicle marker (green dot)
12. Distance measurement line + text (if 2 bodies selected)
13. Wrap all objects in fyne.Container (reused across frames)
```

### Object pooling (`internal/render/cache.go`)

`RenderCache` pools `canvas.Circle`, `canvas.Line`, `canvas.Text`, and `canvas.Image` objects. Each frame calls `Reset()` to rewind indices, then `GetCircle/GetLine/GetText/GetImage` recycles existing objects or allocates new ones. No mutex needed — single render goroutine.

Initial pool sizes: 100 circles, 5000 lines, 50 texts, 20 images.

### Viewport snapshot mechanism (`internal/viewport/viewport.go:167-248`)

`ViewPort.TakeSnapshot()` acquires a single RLock and copies all camera state into an immutable `Snapshot` struct. Precomputed values:
- `CosX, SinX, CosY, SinY, CosZ, SinZ` — rotation trig
- `CenterX, CenterY` — follow body offset
- `DisplayScale` — `DefaultDisplayScale * Zoom`

`Snapshot.WorldToScreen(pos)` performs 3D rotation (if Use3D) then projects:
```
x = (worldX - centerX) / AU * displayScale - panX * displayScale + canvasWidth/2
y = (worldY - centerY) / AU * displayScale - panY * displayScale + canvasHeight/2
```
For 3D mode, Z contributes an oblique projection offset: `x -= z/AU * scale * 0.5`, `y -= z/AU * scale * 0.8`.

**Note:** `ViewPort` also has a locking `WorldToScreen()` method (viewport.go:252) that recomputes trig each call. This is the legacy path, still used by `CreateLabelOverlay()` for GPU mode.

### Lighting model (`internal/render/lighting.go`)

`LightingModel.ApplyDiffuseShading()` implements Lambertian diffuse:
- Light direction: `planet -> sun` (normalized)
- For each pixel in the circular planet image, compute sphere normal from (x,y) offset
- `dot = nx*lightDir.X + (-ny)*lightDir.Y + nz*lightDir.Z`
- `intensity = ambient + diffScale * dot`, clamped to [ambient, 1.0]
- Ambient level: 0.15 (hardcoded)
- Parallelized: rows split across `runtime.NumCPU()` goroutines for images > 100px tall
- Direct `Pix[]` access for both source and destination (RGBA/NRGBA fast path)

**Lighting cache:** `Renderer.lightingCache` maps `"name_diameter"` to shaded `*image.RGBA`. Invalidated when Sun moves > 1e9 meters (~0.007 AU).

**Sun glow:** `SunGlowImage()` generates a radial gradient: alpha = `(1 - distSq) * 0.4`. Cached by diameter.

### Texture pipeline (`internal/render/textures.go`)

```
1. LoadAll() — scans assets/textures/<planet>/albedo.{jpg,png}
   - Dynamically discovers texture directories (no hardcoded planet list)
   - Async: launched in a goroutine from NewRenderer()
2. LoadSkybox() — loads assets/textures/skybox/milky_way.{jpg,png}
3. GetCircleImage(name, diameter) — returns cached circular cutout
   - makeCircularImage(): nearest-neighbor resize + circular mask
   - Cached per (name, diameter) pair
4. GetIrregularImage(name, diameter, seed) — procedural asteroid shape
   - 8-lobe radial perturbation with deterministic RNG
   - Cached per (name_irreg_seed, diameter)
```

### Trail rendering (`internal/render/trail_buffer.go`)

- Renders all trails into a single `*image.RGBA` buffer
- Downsamples to max 200 segments per planet
- **Catmull-Rom interpolation** (`math3d.CatmullRom`) with 4 sub-segments per trail segment
- **Bresenham line drawing** directly into pixel buffer
- Alpha blending with "over" compositing
- Buffer cleared each frame via `clearPixels()` (C memset when CGO available)

### Belt rendering (`internal/render/belt.go`)

- 1500 visual particles (not N-body simulated)
- Positions computed from Keplerian elements each frame:
  - Mean anomaly from `n*simTime + initialAnomaly`
  - Kepler equation solved by 5 Newton iterations
  - True anomaly -> radial distance -> orbital plane position
- Draws 1-3px dots directly into pixel buffer
- Kirkwood gaps at 2.5, 2.82, 2.95 AU are excluded during particle generation

### Comet tails (`internal/render/renderer.go:477-523`)

- Direction: away from Sun (`comet.Position - sun.Position`, normalized)
- Length: `80 / (distAU + 0.5)` pixels, clamped to [10, 200]
- 8 gradient line segments with decreasing alpha and stroke width
- Color: `RGBA{180, 210, 255, alpha}` (blue-white)

### Spacetime fabric (`internal/spacetime/spacetime.go`)

- Computes gravitational metric perturbation `h_00 = 2GM/(c^2 * r)` at grid points
- Adaptive grid resolution: 40-150 lines based on zoom level
- Potential normalized with gamma correction (exponent 0.4) to reveal planetary contributions
- Rendered as horizontal + vertical warped grid lines with color gradient (purple -> red -> orange)
- Caching: only recomputes when zoom/pan change by > 5%

---

## GPU Rendering Path (Rust wgpu)

**Entry:** `GPURenderer.generateImage()` in `internal/render/gpu_renderer.go:122`

### Data flow

```
1. Read planet snapshot + sun snapshot (with Go locks)
2. Set camera: zoom, pan, rotation, follow body -> rust SetCamera()
3. Marshal body data into flat float64 arrays:
   - positions[n*3], colors[n*4], radii[n]
   - sunPos[3], sunColor[4], sunRadius
4. Marshal trail data: trailLengths[n], trailPositions[flat xyz], trailColors[n*4]
   - Launch trajectory appended as extra trail
5. Marshal spacetime data: masses[n+1], positions[(n+1)*3]
6. Set distance line (if two bodies selected)
7. Call render_frame() -> returns RGBA pixel buffer
8. Wrap pixels as image.RGBA -> return to Fyne canvas.Raster
```

**Label overlay:** GPU mode still uses the CPU `Renderer.CreateLabelOverlay()` for text labels, composited on top of the GPU raster.

### FFI boundary (`internal/ffi/render_rust.go`)

The Go side calls C functions declared in `render_core.h`:
- `render_create` / `render_create_with_textures`
- `render_set_camera`, `render_set_bodies`, `render_set_trails`, `render_set_spacetime`
- `render_set_distance_line`, `render_set_rt_mode`, `render_set_rt_quality`
- `render_frame` -> returns `*uint8` pixel buffer (Rust-owned memory)
- `render_resize`, `render_free`
- `render_get_hardware_info` -> GPU detection

The Rust renderer (`crates/render_core/`) implements:
- Software raytracer (`raytracer.rs`, 498 LOC)
- wgpu pipeline renderer (`pipeline.rs`, 334 LOC)
- Spacetime visualization (`spacetime.rs`, 252 LOC)
- Texture loading (`textures.rs`, 210 LOC)
- Camera transforms (`camera.rs`, 125 LOC)
- Main renderer orchestration (`renderer.rs`, 864 LOC)

---

## Native GPU Paths

Metal, CUDA, and ROCm share the same C function interface defined in `native_gpu/common/native_render.h`. Each FFI Go file (`render_metal.go`, `render_cuda.go`, `render_rocm.go`) is a near-identical copy of the same CGO wrapper with different build tags and library paths.

| Backend | Build tag | Library | Platform |
|---------|-----------|---------|----------|
| Metal | `metal_render` | `libnative_render_metal` | macOS |
| CUDA | `cuda_render` | `libnative_render_cuda` | Linux/Windows (NVIDIA) |
| ROCm | `rocm_render` | `libnative_render_rocm` | Linux (AMD) |

Native code sizes: Metal renderer 676 LOC (`.m`) + 233 LOC (`.metal`), CUDA renderer 484 LOC (`.cu`) + 287 LOC raytracer.

---

## Frame Pacing (`internal/ui/app.go:584-655`)

```
Render loop (goroutine):
  ticker: 16ms (60 FPS target)

  if lastFrameOver: skip this tick, reset flag

  snap = simulator.GetSnapshot()  // atomic, never blocks

  if !snap.IsPlaying:
    pausedFrameSkip++
    if < 4: continue   // ~4 FPS when paused (every 4th tick)

  frameStart = time.Now()

  // ... render frame ...

  if time.Since(frameStart) > 16ms:
    lastFrameOver = true  // skip next tick
```

Physics loop runs independently at ~60Hz (`simulator.go:652-689`):
- 16ms ticker
- Drains command channel
- Steps simulation with substep subdivision if `|effectiveDt| > MaxSafeDt` (28800s = 8 hours)
- Publishes snapshot atomically after each tick
