# Current System Overview

## Project Purpose

Solar System Simulator is a real-time N-body gravitational simulator with a Fyne-based GUI. It renders the solar system with physically accurate orbital mechanics, supports general relativistic corrections (Mercury precession), and includes a KSC launch planner. Optional Rust and native GPU backends provide hardware-accelerated physics and rendering.

## Technology Stack

| Layer | Technology | Notes |
|-------|-----------|-------|
| Language (primary) | Go 1.21 | All simulation, UI, and CPU rendering |
| GUI toolkit | Fyne v2.4.5 | Cross-platform desktop GUI |
| Language (optional) | Rust | Physics backend (`physics_core`) and GPU renderer (`render_core`) |
| GPU rendering | wgpu (via Rust) | Cross-platform GPU, ray tracing support |
| Native GPU | Metal (macOS), CUDA (NVIDIA), ROCm (AMD) | Native ray tracers via CGO |
| Math | Custom `math3d.Vec3` | No external math library |
| Build system | Makefile + `go build` + Cargo | Build tags control backend selection |

## Lines of Code (approximate)

| Module | Go LOC | Rust LOC | Native LOC | Notes |
|--------|--------|----------|------------|-------|
| `internal/physics/` | ~1,760 | - | - | Includes tests (~870 LOC) |
| `internal/render/` | ~1,550 | - | - | CPU renderer, GPU renderer, trails, belt, textures, lighting |
| `internal/ui/` | ~2,590 | - | - | App, panels, state, input, menus, settings, theme |
| `internal/viewport/` | ~450 | - | - | Camera, coordinate transforms |
| `internal/spacetime/` | ~330 | - | - | Gravitational field visualization |
| `internal/launch/` | ~930 | - | - | Launch planner + tests |
| `internal/ffi/` | ~840 | - | - | FFI bridges (render + physics) |
| `internal/validation/` | ~460 | - | - | Physics validation harness |
| `internal/math3d/` | ~170 | - | - | Vec3 + Catmull-Rom + tests |
| `internal/assets/` | ~230 | - | - | Asset resolution + validation |
| `cmd/` | ~850 | - | - | 5 entrypoints |
| `pkg/constants/` | 12 | - | - | Physical constants |
| `crates/physics_core/` | - | ~780 | - | Rust N-body simulator |
| `crates/render_core/` | - | ~2,890 | - | Rust wgpu renderer |
| `native_gpu/` | - | - | ~1,820 | Metal (.m/.metal), CUDA (.cu) |
| **Total** | **~11,380** | **~3,670** | **~1,820** | **~16,870 total** |

## Module Dependency Graph

```
cmd/gui/main.go ──> internal/ui
cmd/cli/main.go ──> internal/launch, internal/validation
cmd/solar-sim/  ──> internal/ui, internal/launch, internal/validation, internal/assets

internal/ui ──> internal/physics, internal/render, internal/viewport, internal/launch, internal/ffi, pkg/constants
internal/render ──> internal/physics, internal/viewport, internal/spacetime, internal/launch, internal/math3d, internal/ffi, internal/assets, pkg/constants
internal/physics ──> internal/math3d, internal/physics/gr, internal/ffi, pkg/constants
internal/viewport ──> internal/math3d, internal/physics, pkg/constants
internal/spacetime ──> internal/physics, internal/viewport, pkg/constants (+ fyne)
internal/launch ──> internal/math3d
internal/validation ──> internal/physics, internal/math3d, pkg/constants
internal/ffi ──> (CGO to Rust/Metal/CUDA/ROCm libraries)
internal/assets ──> (standard lib only)
internal/math3d ──> (standard lib only)
pkg/constants ──> (standard lib only)
```

## Data Flow

The system uses a decoupled producer-consumer architecture:

```
UI Thread (Fyne)                      Physics Goroutine (~60Hz)
    |                                        |
    |-- SimCommand (via channel) ----------->|
    |   (SetTimeSpeed, SetPlaying, etc.)     |
    |                                        |-- drainCommands()
    |                                        |-- Step(dt) [RK4 or Verlet]
    |                                        |-- publishSnapshot()
    |                                        |     atomic.Pointer[SimSnapshot]
    |                                        |
    |<--- GetSnapshot() (atomic load) -------|
    |
    |-- Render loop (~60Hz, 16ms ticker)
    |   |-- Read SimSnapshot (lock-free)
    |   |-- viewport.TakeSnapshot() (single RLock)
    |   |-- CreateCanvasFromSnapshot()
    |   |     |-- Belt + Trails (parallel goroutines)
    |   |     |-- Skybox, Spacetime grid, Sun glow, Planets with textures+lighting
    |   |     |-- Comet tails, Labels, Trajectory overlay, Distance line
    |   |-- canvas.Refresh() (Fyne redraws)
```

**Key synchronization mechanisms:**
- `atomic.Pointer[SimSnapshot]` — Physics publishes, render reads. Zero contention.
- `SimCommand` channel (cap 32) — UI sends mutations to physics goroutine. Non-blocking with lock fallback.
- `viewport.Snapshot` — Single RLock per frame, all `WorldToScreen` calls are lock-free after that.
- `AppState` — Central state with observer pattern. Debounced listener notifications (50ms).

## Build Tag System

| Build Tag | Effect | Files |
|-----------|--------|-------|
| (none) | Pure Go. CPU rendering, Go physics. Default. | `*_noop.go`, `*_nocgo.go` |
| `rust_physics` | Rust N-body backend via CGO | `backend_init_rust.go`, `backend_rust.go`, `ffi/physics_rust.go` |
| `rust_render` | Rust wgpu GPU renderer via CGO | `gpu_renderer.go`, `ffi/render_rust.go`, `ui/gpu.go` |
| `metal_render` | Native Metal ray tracer via CGO (macOS) | `ffi/render_metal.go` |
| `cuda_render` | Native CUDA ray tracer via CGO (NVIDIA) | `ffi/render_cuda.go` |
| `rocm_render` | Native ROCm ray tracer via CGO (AMD) | `ffi/render_rocm.go` |
| `nogui` | Headless CLI (no Fyne dependency) | `cmd/solar-sim/gui_noop.go` |
| `cgo` / `!cgo` | C memset for buffer clearing vs Go loop | `memset_cgo.go`, `memset_nocgo.go` |

**Backend selection logic:**
- Physics: `rust_physics` tag compiles `backend_init_rust.go` which calls `NewRustBackend(s)` in `NewSimulator()`. Without the tag, `initBackend` is a no-op and Go integrators run.
- Rendering: Any of `rust_render`, `metal_render`, `cuda_render`, `rocm_render` compiles the real `GPURenderer`. Without any, `NewGPURenderer()` always returns nil.
- The `ffi` package uses mutual exclusion: each render tag defines its own `GPUHardwareInfo` type and `RustRenderer` struct. Only one render tag should be active.

## Entrypoints

| Binary | Source | Purpose |
|--------|--------|---------|
| `bin/solar-system-sim` | `cmd/gui/main.go` | GUI-only. Calls `ui.NewApp().Run()` |
| `bin/solar-system-cli` | `cmd/cli/main.go` | CLI: `validate` and launch planning |
| `bin/solar-sim` | `cmd/solar-sim/main.go` | Unified CLI: `gui`, `run`, `validate`, `launch`, `assets` subcommands |
| `bin/meshgen` | `cmd/meshgen/main.go` | GLB sphere mesh generator |
| `bin/validate-assets` | `cmd/validate-assets/main.go` | Asset directory validator |
