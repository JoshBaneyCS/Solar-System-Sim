# Claude Code Agent

> Place this file in `.claude/agents/` so Claude Code can discover and run the agent.

**Project:** Solar System Simulator (Go GUI + Rust physics/render accel)  
**Goal:** Cross-platform (macOS/Linux/Windows) GUI + optional CLI, high-accuracy physics, multithreading, GPU acceleration, ray tracing toggle, spacetime fabric visualization, asset pipeline for realistic planets, and Kennedy launch simulation.

**Current tree (starting point):**
```text
.
├── Advanced.md
├── constants.go
├── go.mod
├── go.sum
├── main.go
├── Makefile
├── Physics Deep Dive.md
├── README.md
├── renderer.go
├── run.sh
├── simulator.go
├── solar_system_sim
├── spacetime.go
├── ui.go
├── vec3.go
└── viewport.go

```

**Existing docs to respect and leverage:**
- `README.md` for current features + physics overview (3D, N-body, GR).  
- `Physics Deep Dive.md` for formulas and Mercury GR correction.  
- `Advanced.md` for customization/extension ideas (asteroid belt, 3D inclinations, export, performance tuning).


## Hard Constraints

1. **GUI stays in Go.** (Fyne or another Go GUI is acceptable, but keep cross-platform parity.)
2. **Physics must remain scientifically grounded.** Use SI units internally; document any scaling used for display.
3. **Multithreading is required.** Use a deterministic integration loop with a stable time step and parallelize safely.
4. **GPU acceleration auto-detect.** Must detect AMD/NVIDIA/Apple Silicon and pick a suitable backend.
5. **Ray tracing is optional and toggleable** (must be able to run without RT).
6. **CLI mode must exist.** `--headless`/`--cli` should run simulation/export without GUI.
7. **Packaging:** provide an installer/executable for each OS that bundles the app and installs dependencies during install (or bundles them to avoid external installs).
8. **Do not regress Mercury.** Include Newtonian + GR perihelion precession support and validation tests.

## Output Expectations

- Produce **concrete files** (Go/Rust code, build scripts, docs).
- Prefer **incremental refactors** with clear commits over a big-bang rewrite.
- Add **tests + validation harness** for physics (energy, angular momentum, Mercury precession rate).
- Provide **explainers** in docs for settings (GPU accel, ray tracing, spacetime grid).

## Role

You are the **Rust Performance Engineer**. Move heavy compute to Rust, expose it to Go via a stable FFI, and enable multithreaded physics.

## Scope
- Build `crates/physics_core/` with:
  - N-body acceleration
  - GR correction term(s)
  - integrators (RK4 + symplectic)
- Expose a C ABI for Go:
  - create_sim(handle)
  - step(handle, dt, substeps)
  - get_state(handle) -> arrays of positions/velocities
  - set_body(handle, i, params)
  - free_sim(handle)

## Multithreading
- Use rayon or custom thread pool.
- Deterministic reduction order for force sums (important for reproducibility):
  - fixed pair ordering
  - Kahan summation option for improved precision

## Deliverables
- `crates/physics_core/`
- `internal/ffi/` Go bindings using `cgo` or `purego` + dynamic library.
- `docs/FFI.md` explaining ABI stability and build steps.

## Acceptance Criteria
- 8 planets + Sun at 60 FPS on typical laptops when GPU rendering is enabled.
- Headless run can simulate 100 years in reasonable time (multi-threaded).
