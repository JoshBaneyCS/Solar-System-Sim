# Solar System Simulator — Remediation Plan

## Overview

Ten major issues identified across UI state management, rendering, performance, controls, visualization, packaging, and CI/CD. This plan prescribes changes across 10 specialist agents executed sequentially.

## Execution Order

| # | Agent | Key Problem | Status |
|---|-------|-------------|--------|
| 1 | 01_ui-state-sync | Toggle buttons don't reflect actual state | Planned |
| 2 | 05_camera-input-controls | No mouse/keyboard camera controls | Planned |
| 3 | 02_texture-materials-assets | Planets render as solid colors | Planned |
| 4 | 03_lighting-raytrace-sun | No lighting; Sun not acting as light source | Planned |
| 5 | 04_performance-acceleration-macos | Performance unknown; no GPU detection | Planned |
| 6 | 06_launch-visualization | No animated trajectory or playback controls | Planned |
| 7 | 07_gui-overhaul | UI not professional | Planned |
| 8 | 08_test-automation | Missing UI/render/asset tests | Planned |
| 9 | 09_packaging-installers | macOS/Windows installers broken | Planned |
| 10 | 10_github-actions-release | CI builds unreliable | Planned |

## New Files

- `internal/ui/state.go` — Centralized reactive state model
- `internal/ui/input_handler.go` — Mouse/keyboard camera controls
- `internal/ui/diagnostics.go` — Runtime hardware detection
- `internal/ui/mission_playback.go` — Launch trajectory playback engine
- `internal/ui/theme.go` — Custom dark space theme
- `internal/ui/statusbar.go` — Status bar with FPS/sim time/zoom
- `internal/render/textures.go` — Planet texture loading and caching
- `internal/render/lighting.go` — Lambertian diffuse shading from Sun
- `internal/assets/resolve.go` — Cross-platform asset directory resolution
- `.github/workflows/lint.yml` — Go linting workflow

## Modified Files

- `internal/ui/app.go` — State sync, camera integration, layout overhaul
- `internal/ui/menu.go` — Route toggles through AppState
- `internal/ui/settings.go` — Read/write through AppState
- `internal/ui/launch_panel.go` — Playback controls
- `internal/ui/about.go` — Enhanced about dialog
- `internal/render/renderer.go` — Textures, lighting, vehicle marker
- `internal/render/cache.go` — Image object pooling
- `internal/viewport/viewport.go` — New camera methods
- `internal/physics/simulator.go` — Optimized snapshot methods
- `.github/workflows/ci.yml` — Race detection, asset validation
- `.github/workflows/release.yml` — Artifact validation
- `Makefile` — Packaging fixes

## Cross-Cutting Concerns

1. **Asset Resolution**: `internal/assets/resolve.go` used by TextureManager, GPURenderer, packaging
2. **Thread Safety**: All state via `AppState` setters; viewport uses existing mutex
3. **Render Budget**: Pre-cache textures, re-shade only on angle change, trail LOD
4. **Backward Compatibility**: Physics tests unchanged, Rust backends via build tags
