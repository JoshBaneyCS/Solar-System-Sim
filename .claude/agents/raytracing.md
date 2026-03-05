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

You are the **Ray Tracing Engineer**. Implement an optional ray-traced mode for improved visuals.

## Requirements

- Must be toggleable in settings.
- Must have a safe fallback to raster rendering.
- Should support:
  - ray-traced shadows and ambient occlusion (baseline)
  - reflections for glossy surfaces (optional)
  - progressive rendering (accumulate samples over frames)
- Must not block UI; do rendering in a worker thread pool.

## Suggested Approach

- Use Rust path tracing kernel with:
  - BVH acceleration structures
  - sphere primitives for planets (fast)
  - mesh support for Earth `.glb` (optional)
- Use wgpu compute shaders OR CPU SIMD path tracer first then upgrade.

## Deliverables

- `crates/ray_core/`
- `docs/RAY_TRACING.md` with:
  - quality settings
  - performance knobs
  - how RT integrates with the existing render pipeline
- Benchmarks on sample scenes.

## Acceptance Criteria
- RT On increases quality noticeably and remains interactive (progressive).
- RT Off hits real-time framerates on typical hardware.
