# Module Dependency Map

## Package Import Map

### Pure Logic Packages (no GUI, no CGO)

These packages have zero dependency on Fyne or CGO and could be ported to Rust or any other language:

| Package | Internal Imports | Stdlib Imports |
|---------|-----------------|----------------|
| `pkg/constants` | (none) | (none) |
| `internal/math3d` | (none) | `math` |
| `internal/physics/gr` | `math3d`, `pkg/constants` | (none) |
| `internal/launch` | `math3d` | `math`, `fmt`, `io`, `encoding/csv` |
| `internal/validation` | `physics`, `math3d`, `pkg/constants` | `math`, `fmt`, `strings` |
| `internal/assets` | (none) | `os`, `path/filepath`, `fmt`, `image`, `image/jpeg`, `image/png` |

### Packages with Fyne Dependency

| Package | Internal Imports | Fyne Imports |
|---------|-----------------|--------------|
| `internal/viewport` | `math3d`, `physics`, `pkg/constants` | (none, but imports `physics.Body` for FollowBody) |
| `internal/spacetime` | `physics`, `viewport`, `pkg/constants` | `fyne/v2`, `fyne/v2/canvas` |
| `internal/render` | `physics`, `viewport`, `spacetime`, `launch`, `math3d`, `assets`, `ffi`, `pkg/constants` | `fyne/v2`, `fyne/v2/canvas`, `fyne/v2/container` |
| `internal/ui` | `physics`, `render`, `viewport`, `launch`, `math3d`, `ffi`, `pkg/constants` | `fyne/v2`, `fyne/v2/app`, `fyne/v2/canvas`, `fyne/v2/container`, `fyne/v2/widget`, `fyne/v2/dialog`, `fyne/v2/driver/desktop`, `fyne/v2/layout`, `fyne/v2/theme` |

### CGO Boundary Packages

| Package | Build Tags | Links To |
|---------|-----------|----------|
| `internal/ffi` (physics) | `rust_physics` | `libphysics_core` (Rust cdylib) |
| `internal/ffi` (render/rust) | `rust_render` | `librender_core` (Rust cdylib) + Metal/Foundation frameworks |
| `internal/ffi` (render/metal) | `metal_render` | `libnative_render_metal` + Metal/Foundation/CoreGraphics frameworks |
| `internal/ffi` (render/cuda) | `cuda_render` | `libnative_render_cuda` + `libcudart` |
| `internal/ffi` (render/rocm) | `rocm_render` | `libnative_render_rocm` |
| `internal/render` (memset) | `cgo` | C `memset()` via `<string.h>` |

### Physics Package Detail

`internal/physics` imports vary by build tag:

**Default (no tags):**
- `math3d`, `physics/gr`, `pkg/constants`
- stdlib: `image/color`, `math`, `sync`, `sync/atomic`, `time`, `math/rand`

**With `rust_physics`:**
- Adds: `internal/ffi`

### Entrypoint Dependencies

| Entrypoint | Packages Used |
|-----------|---------------|
| `cmd/gui` | `internal/ui` (pulls in everything) |
| `cmd/cli` | `internal/launch`, `internal/validation` |
| `cmd/solar-sim` (gui) | `internal/ui` (conditional on `nogui` tag) |
| `cmd/solar-sim` (run) | `internal/physics`, `pkg/constants` |
| `cmd/solar-sim` (validate) | `internal/validation` |
| `cmd/solar-sim` (launch) | `internal/launch` |
| `cmd/solar-sim` (assets) | `internal/assets` |
| `cmd/meshgen` | stdlib only (generates GLB files) |
| `cmd/validate-assets` | `internal/assets` |

---

## Fyne Coupling Analysis

### Tightly coupled to Fyne

| Package | Coupling Points |
|---------|----------------|
| `internal/ui` | `app.New()`, `fyne.Window`, all widgets, dialog, theme, desktop events |
| `internal/render` | `canvas.Circle`, `canvas.Line`, `canvas.Text`, `canvas.Image`, `fyne.Container` |
| `internal/spacetime` | `canvas.Line`, `fyne.CanvasObject` |

### Indirectly coupled

| Package | Coupling |
|---------|----------|
| `internal/viewport` | Imports `physics.Body` for `FollowBody *physics.Body`. No Fyne imports, but the pointer creates a coupling to the simulator's live data. |
| `internal/render/gpu_renderer.go` | `canvas.Raster` for displaying GPU framebuffer |

### Zero Fyne coupling (portable)

`pkg/constants`, `internal/math3d`, `internal/physics`, `internal/physics/gr`, `internal/launch`, `internal/validation`, `internal/assets`, `internal/ffi`

---

## Build Tag Matrix

| Configuration | Tags | Physics | Rendering | CGO Required |
|--------------|------|---------|-----------|-------------|
| Default | (none) | Go (Verlet/RK4) | CPU (Fyne canvas) | No* |
| Rust physics | `rust_physics` | Rust FFI | CPU (Fyne canvas) | Yes |
| Rust GPU | `rust_render` | Go | GPU (wgpu) | Yes |
| Full Rust | `rust_physics,rust_render` | Rust FFI | GPU (wgpu) | Yes |
| Metal | `metal_render` | Go | GPU (Metal) | Yes |
| Metal + Rust physics | `rust_physics,metal_render` | Rust FFI | GPU (Metal) | Yes |
| CUDA | `cuda_render` | Go | GPU (CUDA) | Yes |
| CUDA + Rust physics | `rust_physics,cuda_render` | Rust FFI | GPU (CUDA) | Yes |
| ROCm | `rocm_render` | Go | GPU (ROCm) | Yes |
| Headless | `nogui` | Go | None (CLI only) | No |

*CGO is used for `memset` optimization even in default build, but falls back gracefully with `!cgo`.

**Tested combinations (per Makefile targets):**
- `make build` — default
- `make build-rust` — `rust_physics`
- `make build-gpu` — `rust_physics,rust_render`
- `make build-metal-gpu` — `metal_render`
- `make build-metal-full` — `rust_physics,metal_render`
- `make build-cuda-gpu` — `cuda_render`
- `make build-cuda-full` — `rust_physics,cuda_render`
- `make build-rocm-gpu` — `rocm_render`
- `make build-rocm-full` — `rust_physics,rocm_render`
- `make build-solar-sim-headless` — `nogui`

**Mutual exclusion:** Only one render tag (`rust_render`, `metal_render`, `cuda_render`, `rocm_render`) should be active at a time. The `ffi` package defines `GPUHardwareInfo` and `RustRenderer` types in each render file, so multiple render tags would cause compilation errors.
