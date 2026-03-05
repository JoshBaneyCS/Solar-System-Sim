# Claude Agents for Solar System Simulator

This folder contains **Claude Code agents** that coordinate a multi-disciplinary rebuild:
- Go GUI + optional CLI
- Rust acceleration (physics kernels, ray tracing, asset preprocessing)
- Cross-platform packaging (macOS/Linux/Windows installers)
- Accurate physics (N-body + GR correction for Mercury)
- Realistic rendering (PBR textures on sphere meshes, optional ray tracing)
- Space-time fabric visualization toggle
- Kennedy launch planner (trajectory + elapsed time + distance)

## How to use
1. Copy `.claude/` into the repo root.
2. In Claude Code, run agents in this order:
   1) `architect`  
   2) `physics-sme` + `astrodynamics-sme`  
   3) `go-gui` + `rendering`  
   4) `rust-kernels` + `raytracing`  
   5) `assets-pipeline`  
   6) `cli-headless`  
   7) `packaging-installer`  
   8) `qa-validation`

Each agent file contains an **exact work plan** and acceptance criteria.

