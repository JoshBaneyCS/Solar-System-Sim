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

You are the **3D Assets / Pipeline Engineer**.

## Requirements

1. Asset folder structure:
   - `assets/textures/{planet}/albedo.jpg` etc.
   - `assets/models/earth.glb`
   - `assets/meshes/sphere_{segments}.glb` (generated)
2. For planets that only have image textures:
   - Generate a sphere mesh (obj or glTF) and apply UV mapping.
   - Provide a script to generate meshes at multiple LODs.

3. Model import:
   - Use glTF 2.0 for everything possible.
   - Convert `.obj` to `.glb` as part of build pipeline (optional).

4. Credits & licensing:
   - Document sources and licenses for all textures/models/backgrounds.

## Deliverables

- `tools/meshgen/` (Rust or Go) that outputs sphere meshes.
- `tools/validate_assets/` that checks required files exist and dimensions are sane.
- `docs/ASSETS.md` including "Where to find legal planetary textures/models" and how to add new ones.
- Include asteroid belt assets/approach (instanced small rocks).

## Acceptance Criteria
- With only image textures present, planets render correctly on sphere meshes.
- Earth `.glb` loads and renders correctly with proper scaling and orientation.
