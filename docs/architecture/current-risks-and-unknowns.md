# Current Risks and Unknowns

## Technical Debt (Prioritized)

### High Priority

1. **Duplicated WorldToScreen implementations**
   - `viewport/viewport.go` has TWO `WorldToScreen` methods: one on `Snapshot` (lock-free, precomputed trig, line 217) and one on `ViewPort` (locking, recomputes trig each call, line 252). The locking version is still used by `CreateLabelOverlay()` in GPU mode. Should be consolidated to snapshot-only.

2. **FFI code quadruplication**
   - `ffi/render_rust.go` (208 LOC), `ffi/render_metal.go` (196 LOC), `ffi/render_cuda.go` (170 LOC), `ffi/render_rocm.go` (169 LOC) are near-identical copies. Every API change must be applied to all four files. A code generation approach or shared interface would reduce maintenance burden.

3. **FollowBody is a raw pointer to simulator data**
   - `viewport.FollowBody *physics.Body` points directly into `simulator.Planets[]`. This creates a race condition: the physics goroutine mutates `Body.Position` while the render goroutine reads `FollowBody.Position` via `TakeSnapshot()`. Currently works because the viewport snapshot reads the pointer's position under RLock, but body addition/removal (moons, comets, asteroids) can invalidate the pointer entirely if the slice is reallocated.

4. **Physics panel equation text is misleading**
   - `ui/app.go:160-161` displays `a_GR = (3G^2*M^2)/(c^2*r^3*L) * (L x r)` which is the OLD incorrect formula. The actual code uses the standard 1PN formula from `gr/correction.go`. The display text was never updated after the GR bug fix.

5. **Simulator reset creates new sim without stopping physics loop**
   - `ui/app.go:401-416` and `ui/menu.go:114-119` create a new `Simulator` and rebind it, but the old physics goroutine may still be running. `StopPhysicsLoop()` is never called, and `StartPhysicsLoop()` is not called on the new simulator. The new simulator operates via direct `Update()` calls only, losing the decoupled architecture.

### Medium Priority

6. **Asteroid belt particles not parallelized**
   - `belt.go:61-108` iterates 1500 particles sequentially, each solving Kepler's equation (5 Newton iterations). This is ~7500 trig evaluations per frame that could be parallelized.

7. **Trail memory management**
   - Trails use `append` + slice truncation (`Trail[1:]`), which causes the underlying array to grow but never shrink. After toggling trails off and on, old capacity is preserved. No mechanism to reclaim trail memory.

8. **No error handling for goroutine leaks**
   - Multiple goroutines are launched without cleanup: physics panel updater (`app.go:201`), canvas size monitor (`app.go:563`), render loop (`app.go:584`), body panel updater (`bodies_panel.go:111`), telemetry updater (`launch_panel.go:194`), FPS counter (`statusbar.go:79`). None are cleaned up on window close.

9. **Lighting cache grows unbounded**
   - `Renderer.lightingCache` maps `"name_diameter"` to shaded images. As zoom changes, new diameters generate new cache entries. The only invalidation is Sun movement > 1e9m. At extreme zoom levels, hundreds of cached images could accumulate.

10. **Settings GPUMode/RayTracing/QualityPreset are persisted but not applied**
    - `settings.go` stores these values and the settings dialog lets users change them, but `ApplyFromSettings()` in `state.go` never reads GPUMode, RayTracing, or QualityPreset. They are dead configuration.

### Low Priority

11. **`draw.Over` import kept alive artificially**
    - `textures.go:318`: `var _ = draw.Over` — imports `image/draw` for "potential future compositing" that doesn't exist.

12. **`math.Pi` import kept alive artificially**
    - `renderer.go:532`: `var _ = math.Pi` — comment says "used by lighting angle threshold" but no such threshold exists in this file.

13. **Redundant unused variable in propagator**
    - `planner.go:147`: `_, dv2 := HohmannDeltaV(...)` followed by `_ = dv2` and then `dv1, _ := HohmannDeltaV(...)` — calls the function twice unnecessarily.

---

## Dead or Experimental Code

| Item | Location | Assessment |
|------|----------|------------|
| `ViewPort.WorldToScreen()` (locking version) | `viewport/viewport.go:252-313` | Partially dead. Only used by GPU label overlay. Should migrate to snapshot. |
| `Renderer.SelectedBodies` | `render/renderer.go:33` | Functional but externally managed. No code in `ui/` actually sets this (distance measurement UI is wired but bodies panel doesn't populate it). |
| `Settings.GPUMode`, `Settings.RayTracing`, `Settings.QualityPreset` | `ui/settings.go:11-13` | Dead config. Persisted but never read by any logic. |
| ROCm native source | `native_gpu/rocm/` | Only a Makefile exists. No `.hip` or `.cpp` source files found. The `ffi/render_rocm.go` FFI wrapper exists but has nothing to link against. |
| `cmd/meshgen` | `cmd/meshgen/main.go` | Functional but generates GLB meshes that nothing in the codebase consumes. The Rust renderer may use them, but no Go code loads `.glb` files. |
| `assets/validate.go` earth.glb check | `assets/validate.go:42-49` | Validates a model file that the Go codebase never loads. |

---

## Duplicated Logic Between Go and Rust

| Domain | Go Location | Rust Location | Divergence Risk |
|--------|------------|---------------|-----------------|
| N-body gravity | `physics/simulator.go:220-268` | `crates/physics_core/src/sim.rs` | Medium. Same algorithm but different softening, parallelism. |
| GR correction | `physics/gr/correction.go` | `crates/physics_core/src/gr.rs` | Low. Both use standard 1PN formula now. |
| Vec3 operations | `internal/math3d/vec3.go` | `crates/physics_core/src/vec3.rs` | Low. Simple arithmetic. |
| Camera transform | `viewport/viewport.go` (WorldToScreen) | `crates/render_core/src/camera.rs` | Medium. GPU camera must match CPU viewport for label overlay alignment. |
| Spacetime curvature | `spacetime/spacetime.go` | `crates/render_core/src/spacetime.rs` | Medium. Both compute h_00 but may use different normalization. |
| Raytracer | (none in Go) | `crates/render_core/src/raytracer.rs` | N/A. Rust-only feature. |

---

## FFI Fragility and Maintenance Burden

1. **Unsafe memory sharing**: `RenderFrame()` returns a Go slice backed by Rust-owned memory (`unsafe.Slice`). This is valid only until the next `RenderFrame()` or `Free()` call. A retained reference would cause use-after-free.

2. **ABI stability**: The C header files (`render_core.h`, `physics_core.h`, `native_render.h`) are the contract. Any Rust struct layout change or function signature change breaks Go without compile-time detection.

3. **Platform-specific linker flags**: Each FFI file hardcodes relative library paths (`-L${SRCDIR}/../../crates/.../target/release`). Moving directories or changing build output paths breaks silently.

4. **Quadruplicated wrapper code**: Any new rendering API function must be added to 4 Go files + 3 native implementations + 1 Rust implementation.

---

## Fyne-Specific Coupling That Blocks Migration

1. **Render output is Fyne canvas objects** — The CPU renderer produces `canvas.Circle`, `canvas.Line`, `canvas.Text`, `canvas.Image` objects. Migration to Bevy requires replacing all of these with Bevy sprites/meshes.

2. **`fyne.Container` as scene graph** — The render output is a flat list of Fyne objects wrapped in `container.NewWithoutLayout()`. Bevy uses an ECS with transform hierarchies.

3. **Theme system** — `SpaceTheme` implements `fyne.Theme`. Bevy has its own theming.

4. **Widget-heavy UI** — All controls (sliders, checkboxes, selects, buttons) are Fyne widgets. Bevy uses `egui` or `bevy_ui`.

5. **Input handling** — `InteractiveCanvas` implements Fyne's `Scrollable`, `Draggable`, `Focusable`, `desktop.Mouseable`, `desktop.Hoverable` interfaces.

6. **Settings persistence** — Uses `fyne.Preferences` API.

7. **File dialogs** — Screenshot export uses `dialog.ShowFileSave`.

---

## Platform-Specific Concerns

| Concern | Details |
|---------|---------|
| macOS `-lobjc` duplicate warning | Harmless linker warning from Fyne + CGO. Documented in MEMORY.md. |
| Metal only on macOS | `metal_render` build tag is macOS-only. Metal frameworks required. |
| CUDA only on NVIDIA Linux/Windows | Requires `libcudart` and NVIDIA GPU. |
| ROCm only on AMD Linux | FFI wrapper exists but no native source code found. |
| Apple Silicon detection | `diagnostics.go:30-31` checks `darwin + arm64`. Used for display only. |
| Asset paths | `assets/resolve.go` searches relative to CWD and executable. macOS `.app` bundle support included. |

---

## Architectural Decisions Implied by the Codebase

1. **Atomic snapshot for lock-free rendering** — The `SimSnapshot` pattern via `atomic.Pointer` was a deliberate optimization to eliminate lock contention between physics and render goroutines.

2. **Observer pattern for state synchronization** — `AppState` with listeners and debounced notifications keeps UI widgets, menu items, and settings in sync.

3. **Command pattern for physics mutations** — `SimCommand` with `Apply func(s *Simulator)` provides safe cross-goroutine mutation without exposing the simulator's lock.

4. **Build tags over runtime polymorphism** — Backend selection is compile-time via build tags, not runtime. This eliminates overhead but increases build matrix complexity.

5. **Viewport snapshot for precomputed trig** — `TakeSnapshot()` precomputes sin/cos values once per frame, avoiding thousands of redundant trig calls in `WorldToScreen`.

---

## Migration Blockers (Before Bevy Migration Can Begin)

### Must resolve

1. **Decouple render output from Fyne types** — The renderer must produce data (positions, colors, sizes, alpha) rather than Fyne canvas objects. This is the single largest coupling point.

2. **Fix simulator reset lifecycle** — Reset must properly stop the old physics loop and start a new one. Current code leaks goroutines.

3. **Replace `FollowBody *physics.Body` with body ID** — The raw pointer is unsafe across goroutine boundaries and doesn't survive simulator reset. Use a string name or integer index instead.

4. **Consolidate WorldToScreen** — One implementation, snapshot-based. Remove the locking version.

5. **Extract render-independent state** — `AppState` should not hold a reference to `*App`. The observer pattern is good but the direct app pointer creates bidirectional coupling.

### Should resolve

6. **Address FFI quadruplication** — Either code-generate the wrappers or define a shared Go interface that all backends implement, with a single thin CGO bridge per backend.

7. **Fix physics panel GR formula text** — Cosmetic but confusing for developers.

8. **Clean up dead settings** (GPUMode, RayTracing, QualityPreset) or wire them up.

9. **Add goroutine lifecycle management** — Context cancellation or done channels for all background goroutines.

### Nice to have

10. **Decouple spacetime from Fyne** — `SpacetimeRenderer` returns `[]fyne.CanvasObject`. Should return grid point data instead.

11. **Separate launch trajectory from renderer** — `Renderer` holds `LaunchTrajectory` and `LaunchVehiclePos` as mutable fields set from the UI. Should come through the snapshot or a dedicated overlay data struct.

12. **ROCm native implementation** — Either implement it or remove the dead FFI wrapper and Makefile target.
