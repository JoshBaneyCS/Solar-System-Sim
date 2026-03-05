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

You are the **Rendering Engineer** (real-time). Build rasterized PBR planet rendering + spacetime fabric overlay. Integrate with Go UI.

## Requirements

1. **Planet rendering**
   - Each planet uses a sphere mesh + textures:
     - albedo/color map
     - normal map (optional)
     - roughness/metalness (optional)
     - emissive map for Sun / city lights (optional)
   - Support `.glb` for Earth (existing), and **image-only** assets for others:
     - generate `sphere_mesh.obj` (or glTF sphere) and map textures.
2. **Starfield / background**
   - HDRI skybox or cube map (image files).
3. **Asteroid belt visualization**
   - Instanced meshes with LOD and frustum culling.
4. **Spacetime fabric**
   - Render as grid mesh (plane) with vertex displacement from curvature field.
   - Toggle on/off; adjustable resolution.
5. **GPU auto-detect**
   - Use Rust `wgpu` backend as primary.
   - Provide a CPU fallback renderer (simpler, for headless/testing).

## Integration Options (choose one and document)
- Option A: Rust renderer outputs RGBA frames to shared memory; Go displays in canvas.
- Option B: Embed wgpu surface directly in Go window (harder).
- Option C: Render in Go (OpenGL) with go-gl; use Rust only for ray tracing.

## Deliverables
- `crates/render_core/` (Rust wgpu)
- A stable C ABI to:
  - init renderer (width/height)
  - set camera
  - set bodies meshes/textures
  - render frame -> pointer to pixel buffer
- `docs/RENDERING.md` including performance targets and feature flags.

## Acceptance Criteria
- Runs on macOS (Metal), Windows (DX12), Linux (Vulkan) with same binary set.
- Can disable GPU in settings and still run with CPU fallback.
