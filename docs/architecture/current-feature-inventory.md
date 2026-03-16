# Current Feature Inventory

## Celestial Bodies

| Feature | Files | Status |
|---------|-------|--------|
| Sun (star) | `physics/simulator.go:127-137` | Complete |
| 8 planets (Mercury-Neptune) | `physics/planets.go` | Complete. Real orbital elements, 3D inclination. |
| Pluto (dwarf planet) | `physics/planets.go:130-145` | Complete |
| 8 moons (Moon, 4 Galilean, Titan, Phobos, Deimos) | `physics/moons.go` | Complete. Heliocentric conversion from parent-relative elements. |
| 4 comets (Halley, Hale-Bopp, Encke, Swift-Tuttle) | `physics/comets.go` | Complete |
| 6 named asteroids (Ceres, Vesta, Pallas, Hygiea, Apophis, Bennu) | `physics/asteroids.go` | Complete |
| Visual asteroid belt (1500 particles) | `physics/asteroids.go:116-134`, `render/belt.go` | Complete. Kirkwood gaps modeled. Not N-body simulated. |
| Body types: Star, Planet, DwarfPlanet, Moon, Comet, Asteroid | `physics/body.go:12-19` | Complete |
| Creation from orbital elements | `physics/simulator.go:162-218` | Complete. Perifocal velocity formula (mu/h method). |
| Moon creation (parent-relative to heliocentric) | `physics/simulator.go:411-465` | Complete |
| Dynamic add/remove by type | `physics/simulator.go:469-515` | Complete |
| Physical radius (for zoom-to-fill rendering) | `physics/body.go:32` | Complete |

## Physics

| Feature | Files | Status |
|---------|-------|--------|
| N-body gravitational simulation | `physics/simulator.go:220-268` | Complete. All bodies interact via Newtonian gravity. |
| RK4 integrator | `physics/simulator.go:319-387` | Complete. Pre-allocated scratch buffers. |
| Velocity Verlet integrator (default) | `physics/verlet.go` | Complete. Symplectic, better energy conservation. |
| General relativity (1PN) correction | `physics/gr/correction.go` | Complete. Standard 1PN: `(GM/(c^2*r^3))[(4GM/r-v^2)r+4(r.v)v]`. 42.97"/century for Mercury. |
| Softening length | `physics/simulator.go:103` | Implemented but default is 0 (unused). |
| Parallel acceleration computation | `physics/simulator.go:272-292` | Complete. Parallelizes when body count >= 12. |
| Substep subdivision | `physics/simulator.go:389-406` | Complete. Subdivides when `|effectiveDt| > MaxSafeDt` (28800s). |
| Time control (play/pause, speed, rewind) | `ui/state.go`, `ui/app.go:216-243` | Complete. Speed range: 2^(-10) to 2^10. Negative speeds for rewind. |
| Physics backend abstraction | `physics/backend.go` | Complete. `PhysicsBackend` interface with Step/GetState/SetConfig/Close. |
| Rust physics backend | `physics/backend_rust.go`, `ffi/physics_rust.go` | Complete. GR applied only to Mercury. |
| Decoupled physics loop | `physics/simulator.go:652-696` | Complete. 60Hz goroutine, atomic snapshot publishing. |
| Command channel (UI -> physics) | `physics/simulator.go:620-648` | Complete. Buffered channel (32), lock fallback. |

## Rendering

| Feature | Files | Status |
|---------|-------|--------|
| CPU rendering (Fyne canvas objects) | `render/renderer.go` | Complete |
| GPU rendering (Rust wgpu) | `render/gpu_renderer.go` | Complete |
| Native Metal ray tracer | `ffi/render_metal.go`, `native_gpu/metal/` | Complete |
| Native CUDA ray tracer | `ffi/render_cuda.go`, `native_gpu/cuda/` | Complete |
| Native ROCm ray tracer | `ffi/render_rocm.go` | Partial. No native ROCm source in `native_gpu/rocm/` (Makefile exists but no source files found). |
| Skybox (milky way background) | `render/textures.go:81-95`, `render/renderer.go:102-113` | Complete |
| Planet textures (albedo maps) | `render/textures.go:37-78` | Complete. Dynamic directory discovery. |
| Circular texture masking | `render/textures.go:143-176` | Complete. Nearest-neighbor resize. |
| Lambertian diffuse shading | `render/lighting.go:30-143` | Complete. Parallelized, direct pixel access. |
| Sun glow (radial gradient) | `render/lighting.go:147-181` | Complete. Cached by diameter. |
| Orbital trails (Catmull-Rom + Bresenham) | `render/trail_buffer.go` | Complete. Max 200 segments, alpha blending. |
| Asteroid belt visualization | `render/belt.go` | Complete. Kepler equation solved per particle per frame. |
| Comet tails | `render/renderer.go:477-523` | Complete. 8-segment gradient away from Sun. |
| Irregular asteroid shapes | `render/textures.go:217-293` | Complete. Procedural with deterministic RNG. |
| Object pool (circles, lines, text, images) | `render/cache.go` | Complete |
| Spacetime fabric grid | `spacetime/spacetime.go` | Complete. Adaptive resolution, cached. |
| Distance measurement line | `render/renderer.go:292-319` | Complete. AU, km, light-minutes. |
| Launch trajectory overlay | `render/renderer.go:365-414` | Complete. Color gradient by progress. |
| Launch vehicle marker | `render/renderer.go:282-290` | Complete. Green dot at interpolated position. |
| Label overlay (GPU mode) | `render/renderer.go:417-475` | Complete. Fyne text on top of GPU raster. |
| Physical radius rendering | `render/renderer.go:216-226` | Complete. Uses physical radius when zoomed in enough. Max 5000px. |
| Fast buffer clearing (C memset) | `render/memset_cgo.go` | Complete. Fallback Go loop for !cgo. |

## Camera

| Feature | Files | Status |
|---------|-------|--------|
| Zoom (scroll wheel, slider, R/F keys) | `viewport/viewport.go:69-106`, `ui/input_handler.go:56-59` | Complete. Range: 0.01x to 10,000,000x. |
| Pan (Shift+drag, WASD keys) | `viewport/viewport.go:81-93`, `ui/input_handler.go:68-74` | Complete |
| 3D rotation (drag orbit, Q/E roll) | `ui/input_handler.go:64-89`, `viewport/viewport.go:108-114` | Complete. Pitch clamped to +/- 90 degrees. |
| Follow body | `viewport/viewport.go:26`, `ui/app.go:327-348` | Complete. Centers camera on selected body. |
| Auto-fit all planets | `viewport/viewport.go:125-165` | Complete. Computes bounding box with 10% margin. |
| 3D view toggle | `ui/app.go:351-355` | Complete. Auto-enables on drag. |
| Slider controls (pitch, yaw, roll) | `ui/app.go:357-398` | Complete |
| Reset 3D view | `ui/app.go:390-399` | Complete |

## UI

| Feature | Files | Status |
|---------|-------|--------|
| Main window with HSplit layout | `ui/app.go:509-551` | Complete. Left tabs + center canvas + right physics panel. |
| Controls panel (Simulation tab) | `ui/app.go:215-507` | Complete. Play/pause, speed, zoom, follow, 3D, display options, physics options, sun mass, integrator, reset. |
| Physics panel (equations display) | `ui/app.go:82-213` | Complete. Live-updating Earth values, equations. Updates at 10Hz. |
| Bodies panel | `ui/bodies_panel.go` | Complete. Grouped by type, live distance/velocity/period. Follow + trail toggle per body. |
| Launch planner panel | `ui/launch_panel.go` | Complete. Destination/vehicle selection, simulate, clear, results display. |
| Mission playback | `ui/mission_playback.go`, `ui/launch_panel.go:96-213` | Complete. Play/pause, speed control (1x-64x), timeline scrubbing, telemetry display. |
| Status bar | `ui/statusbar.go` | Complete. FPS, sim time, speed, zoom, system info. Throttled to every 4th frame. |
| Settings dialog | `ui/settings.go` | Complete. GPU mode, ray tracing, quality, integrator, display toggles. Persisted via Fyne preferences. |
| Settings persistence | `ui/settings.go:46-80` | Complete. Loads on startup, saves on apply. |
| About dialog | `ui/about.go` | Complete. Logo, version, author, links, system info, credits. |
| Main menu | `ui/menu.go` | Complete. File (screenshot, quit), View (toggles, window), Simulation (play/pause, reset, integrator), Settings, About. |
| Screenshot export | `ui/menu.go:14-23` | Complete. PNG via file save dialog. |
| Custom dark theme | `ui/theme.go` | Complete. Space-themed colors, custom sizing. |
| AppState (centralized state) | `ui/state.go` | Complete. Observer pattern with debounced notifications. |
| Interactive canvas (mouse/keyboard) | `ui/input_handler.go` | Complete. Implements Scrollable, Draggable, Focusable, MouseUp/MouseDown, Tapped. |
| Diagnostics / runtime info | `ui/diagnostics.go` | Complete. OS, arch, CPU count, Go version, GPU detection. |
| GPU info detection | `ui/gpu.go` | Complete (with `rust_render` tag). Vendor, device, backend, tier. |

## Launch Planner

| Feature | Files | Status |
|---------|-------|--------|
| Hohmann transfer computation | `launch/orbital.go:19-31` | Complete |
| Hohmann transfer time | `launch/orbital.go:35-38` | Complete |
| Plane change delta-v | `launch/orbital.go:42-44` | Complete |
| Hyperbolic excess delta-v | `launch/orbital.go:49-53` | Complete |
| Vis-viva equation | `launch/orbital.go:57-59` | Complete |
| Circular/escape velocity | `launch/orbital.go:7-15` | Complete |
| LEO direct insertion | `launch/planner.go:79-83` | Complete |
| GTO (elliptical transfer) | `launch/planner.go:95-101` | Complete |
| Lunar TLI | `launch/planner.go:104-117` | Complete. Includes lunar orbit insertion. |
| Mars interplanetary transfer | `launch/planner.go:120-134` | Complete. Heliocentric Hohmann + hyperbolic excess. |
| Vehicle presets (Generic, Falcon-like, Saturn V-like) | `launch/vehicle.go` | Complete |
| Destination presets (LEO, ISS, GTO, Moon, Mars) | `launch/destination.go` | Complete |
| Trajectory propagation (RK4 2-body) | `launch/propagator.go` | Complete. ~1000 output points. |
| Delta-v budget breakdown | `launch/planner.go:11-17` | Complete. Ascent, plane change, transfer, arrival. |
| Tsiolkovsky rocket equation | `launch/rocket.go` | Complete |
| CSV export | `launch/csv.go` | Complete. Time, position, velocity, acceleration, distance. |
| Feasibility check | `launch/planner.go:88-89` | Complete. Compares vehicle dv to budget. |
| Summary text | `launch/planner.go:185-229` | Complete. Human-readable with formulas used. |

## Validation Harness

| Scenario | Files | Status |
|----------|-------|--------|
| Energy conservation | `validation/energy.go` | Complete. Measures total energy drift over N years. |
| Angular momentum conservation | `validation/angular_momentum.go` | Complete |
| Kepler period (Earth) | `validation/kepler.go` | Complete. Measures orbital period from position crossings. |
| Kepler period (Mercury) | `validation/kepler.go` | Complete |
| Mercury precession | `validation/mercury_precession.go` | Complete. Uses Laplace-Runge-Lenz vector. |
| Unified runner | `validation/validation.go` | Complete. `RunAll()`, `RunScenario()`, human-readable output. |

## CLI

| Command | Entrypoint | Status |
|---------|-----------|--------|
| `solar-sim gui` | `cmd/solar-sim/gui_enabled.go` | Complete. Launches Fyne app. `nogui` tag disables. |
| `solar-sim run` | `cmd/solar-sim/cmd_run.go` | Complete. Headless sim with CSV/JSON export. Configurable integrator, dt, years, sample interval. |
| `solar-sim validate` | `cmd/solar-sim/cmd_validate.go` | Complete. Runs physics validation scenarios. |
| `solar-sim launch` | `cmd/solar-sim/cmd_launch.go` | Complete. Launch planning with CSV export. |
| `solar-sim assets verify` | `cmd/solar-sim/cmd_assets.go` | Complete. Validates asset directory structure. |
| `cli` (standalone) | `cmd/cli/main.go` | Complete. `validate` subcommand + launch planning. |

## Asset Pipeline

| Feature | Files | Status |
|---------|-------|--------|
| Asset directory resolution | `assets/resolve.go` | Complete. Searches CWD, exe-relative, macOS .app bundle. |
| Texture validation (size, format) | `assets/validate.go` | Complete. Min 512px wide, max 16384px. |
| GLB model validation (header check) | `assets/validate.go:105-122` | Complete |
| Mesh generation (GLB spheres) | `cmd/meshgen/main.go` | Complete. 32 and 64 segment spheres. |
| Asset setup from source textures | `Makefile:175-190` | Complete. Copies from `space-object-textures/` to `assets/textures/`. |

## Build & Packaging

| Feature | Files | Status |
|---------|-------|--------|
| Makefile with 30+ targets | `Makefile` | Complete |
| macOS packaging | `packaging/package-macos.sh` | Referenced in Makefile, not verified |
| Linux packaging | `packaging/package-linux.sh` | Referenced in Makefile, not verified |
| Windows packaging | `packaging/package-windows.sh` | Referenced in Makefile, not verified |
| Race detector dev build | `Makefile:219-223` | Complete |
| Dependency check (otool) | `Makefile:254-264` | Complete |
| CI workflow | `.github/workflows/ci.yml` | Present (modified) |
| Release workflow | `.github/workflows/release.yml` | Present (modified) |
| Lint workflow | `.github/workflows/lint.yml` | Present (new, untracked) |
