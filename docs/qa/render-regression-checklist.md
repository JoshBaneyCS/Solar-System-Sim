# Render & Functional Regression Checklist

Visual and functional regression tests for the Bevy migration. Each item must pass before the Go app can be deprecated.

---

## Visual Regression Tests

| # | Feature | How to Test | Acceptance Criteria | Automated? | Phase | Source Reference |
|---|---------|------------|--------------------|-----------| ------|-----------------|
| R1 | Planet positions match physics | Run simulation for 1 Earth year. Verify Earth returns to approximately starting position. Cross-check planet positions with golden test data at 100 and 1000 steps. | Positions match golden values within 0.1 m. Visual orbit is elliptical and does not spiral or diverge. | Yes (golden test). Visual: manual screenshot comparison. | B | `golden_test.go:17-39` |
| R2 | Planet colors match specification | Screenshot each planet at default zoom. Compare RGBA values against `planets.go` color definitions. | Each planet sphere color matches the Go source RGBA values (converted to linear sRGB). Side-by-side comparison shows no visible color shift. | Semi-auto: pixel-sample comparison script. | B | `physics/planets.go` (Color fields) |
| R3 | Planet sizes proportional to data | At default zoom, verify that Jupiter is visibly larger than Earth, and Mercury is smaller than Earth. Compare display radii. | Relative sizes follow the `DisplayRadius` ratios from `planets.go`. Absolute size scaled by camera distance. | Manual | B | `render/renderer.go:216-226` |
| R4 | Sun rendered with glow/bloom | Screenshot at default zoom. Sun should have a bright center with a soft radial glow extending beyond the sphere edge. | Sun emissive material visible. Bloom effect extends at least 2x the Sun's rendered diameter. No hard edge cutoff. | Manual | B | `render/lighting.go:147-181` (sun glow), Bevy `BloomSettings` |
| R5 | Orbital trails visible when enabled | Enable trails, run for several orbits. Trails should appear behind each planet. | Each planet has a visible trail tracing its orbital path. Trail is smooth (no jagged segments at default zoom). Inner planets show complete orbit loops. | Manual | C | `render/trail_buffer.go` |
| R6 | Trail color matches planet color | Enable trails and zoom to individual planets. Compare trail color to planet color. | Trail color is the same hue as the planet body. | Manual | C | `render/trail_buffer.go` (uses body color) |
| R7 | Trail fades with age | Enable trails and observe the oldest segment vs newest segment of a trail. | Oldest trail points are visibly more transparent than newest points. Fade is smooth and continuous. | Manual | C | `render/trail_buffer.go` (alpha blending, max 200 segments) |
| R8 | Asteroid belt visible (1500 particles) | Enable belt, zoom out to show full solar system. Belt should appear between Mars and Jupiter. | Belt is visible as a ring of particles between ~2.1 and 3.3 AU. Kirkwood gaps at 2.5, 2.82, 2.95 AU are discernible as density reductions. Particle count is 1500. | Manual (count via entity query in debug mode) | C | `render/belt.go`, `physics/asteroids.go:116-134` |
| R9 | Comet tail renders away from sun | Enable comets, find a comet, zoom in. Tail should point away from the Sun regardless of comet's orbital position. | Tail direction is anti-sunward (within 15 degrees of the sun-comet radial vector extended outward). Tail is 8 visible segments with gradient. | Manual | C | `render/renderer.go:477-523` |
| R10 | Labels appear next to bodies | Enable labels. Each body should show its name. | Text label is visible, positioned below or near each body sphere. Font is legible at default zoom. Labels do not overlap at default zoom for outer planets. | Manual | C | `render/renderer.go:417-475` |
| R11 | Skybox/background renders | Rotate camera to look away from all bodies. The background should show a milky way texture, not solid black. | Skybox texture covers the full celestial sphere. No visible seams or distortion. Stars are visible. | Manual | B | `render/textures.go:81-95`, `render/renderer.go:102-113` |
| R12 | Spacetime grid deforms near massive bodies | Enable spacetime overlay. The grid should show visible warping near the Sun. | Grid lines curve inward near the Sun. Deformation magnitude is proportional to `h_00 = 2GM/(c^2*r)`. Grid is adaptive resolution (finer near the Sun). | Manual | C | `spacetime/spacetime.go` |
| R13 | Distance measurement line between selected bodies | Select two bodies. A line should appear between them with distance readout. | Line is visible between the two bodies. Distance is displayed in AU, km, and light-minutes. Values update as bodies orbit. | Manual | C | `render/renderer.go:292-319` |
| R14 | Launch trajectory overlay renders | Open launch planner, simulate a mission (e.g., Moon). Trajectory should appear as a colored line. | Trajectory line visible. Color gradient from green (start) to red (end). Line follows expected transfer orbit shape. | Manual | D | `render/renderer.go:365-414` |
| R15 | Launch vehicle marker | During mission playback, a marker should appear at the vehicle's current position along the trajectory. | Green dot/sphere visible at the interpolated position. Moves along trajectory during playback. | Manual | D | `render/renderer.go:282-290` |
| R16 | Camera zoom works smoothly | Scroll mouse wheel in and out. | Zoom is continuous and smooth (no jumping). Range covers at least 0.01x to 10,000,000x (or equivalent Bevy range). Logarithmic feel. | Manual | B | `viewport/viewport.go:69-106` |
| R17 | Camera rotation works smoothly | Click and drag to rotate the view. | Rotation is continuous, follows mouse movement. Pitch clamped to avoid gimbal lock (no flip at poles). Yaw unrestricted. | Manual | B | `ui/input_handler.go:64-89` |
| R18 | Follow-body camera tracks planet | Select "Follow Earth" from dropdown. Camera should center on Earth and track it as it orbits. | Camera center stays on the followed body. Body remains centered as it moves. Zoom and rotation still work relative to the followed body. | Manual | B | `viewport/viewport.go:26`, `ui/app.go:327-348` |
| R19 | Physical radius rendering at close zoom | Zoom very close to a planet (e.g., Jupiter). The sphere should scale to reflect the planet's physical radius. | At close zoom, planet sphere size is proportional to its `PhysicalRadius` value. Max rendered size is capped at 5000px equivalent. | Manual | B | `render/renderer.go:216-226` |
| R20 | Planet textures render correctly | Each planet should show its albedo texture (if loaded), not just flat color. | Textures are visible and correctly UV-mapped on sphere surface. No obvious stretching or seam artifacts. Lambertian shading visible (lit side brighter than dark side). | Manual | B | `render/textures.go:37-78`, `render/lighting.go:30-143` |
| R21 | Irregular asteroid shapes | Enable asteroids, zoom to a named asteroid. Shape should appear non-spherical. | Asteroids have procedurally-generated irregular shapes (8-lobe radial perturbation). Shape is deterministic per asteroid name. | Manual | D | `render/textures.go:217-293` |

---

## Functional Regression Tests (AppState Toggles)

Each test verifies that a UI control changes the simulation/rendering behavior correctly. Source: `internal/ui/state.go`.

| # | Feature | How to Test | Acceptance Criteria | Automated? | Phase | Default Value |
|---|---------|------------|--------------------|-----------| ------|--------------|
| F1 | Play/pause toggles simulation | Click pause button. Planets should stop moving. Click play. They resume. | When paused: all body positions are frozen. Time counter stops incrementing. When unpaused: simulation resumes from the same state. | Yes (unit test: set isPlaying=false, step, verify positions unchanged) | B | Playing (true) |
| F2 | Time speed slider affects simulation rate | Move speed slider to max. Planets should orbit faster. Move to min, they should be nearly stopped. | At 2x speed, Earth completes an orbit in half the real time. At 0.5x, it takes double. Speed range: 2^(-10) to 2^(10). | Semi-auto (measure orbit completion time at different speeds) | B | 1.0 |
| F3 | Time reversal (negative speed) works | Set speed to a negative value. Planets should orbit backward. | Planets move in reverse along their orbits. Position history rewinds. No NaN or simulation instability. | Manual + stability check (no NaN after 1000 reverse steps) | C | N/A (positive by default) |
| F4 | Trail toggle shows/hides trails | Toggle trails off. Trails disappear. Toggle on. They reappear and grow. | Off: no trail lines visible. On: trails begin accumulating from current position. Existing trails are cleared when toggled off (per Go behavior: `sim.Planets[i].Trail = sim.Planets[i].Trail[:0]`). | Yes (unit test + visual) | C | On (true) |
| F5 | Spacetime toggle shows/hides grid | Toggle spacetime on. Grid appears in the ecliptic plane. Toggle off. It disappears. | On: warped grid visible near Sun. Off: no grid. Toggle is instantaneous (no fade). | Manual | C | Off (false) |
| F6 | Planet gravity toggle affects N-body | Toggle planet gravity off. Run simulation. Mercury's orbit should become a perfect Keplerian ellipse (no perturbations). Toggle on. Perturbations reappear. | Off: energy conservation improves dramatically (relative drift < 1e-6 vs < 1e-5 with N-body). Orbits are smoother. On: slight perturbations visible in inner planet orbits over many orbits. | Semi-auto (measure energy drift with/without) | B | On (true) |
| F7 | Relativity toggle affects Mercury precession | Toggle relativity off. Run Mercury for many orbits. Perihelion should not precess (beyond numerical noise). Toggle on. Precession should resume at ~43 arcsec/century. | Off: Mercury precession rate ~0 arcsec/century (Newtonian only). On: ~43 arcsec/century. This requires long simulation runs to observe. | Yes (validation harness test) | B | On (true) |
| F8 | Integrator switch (RK4 to Verlet) works | Switch integrator to RK4. Run simulation. Switch to Verlet. Simulation continues without crash or visible discontinuity. | No crash on switch. Simulation continues. Energy conservation properties change as expected (Verlet is symplectic). | Yes (unit test: switch mid-simulation, verify no NaN) | B | Verlet |
| F9 | Moon visibility toggle | Toggle moons on. 8 moons should appear orbiting their parent planets. Toggle off. They disappear. | On: Moon orbits Earth, 4 Galilean moons orbit Jupiter, Titan orbits Saturn, Phobos and Deimos orbit Mars. Off: no moons visible, moon entities removed from simulation. | Manual (count entities) | C | On (true) |
| F10 | Comet visibility toggle | Toggle comets on. 4 comets should appear with tails. Toggle off. They disappear. | On: Halley, Hale-Bopp, Encke, Swift-Tuttle visible with tails. Off: no comets. | Manual | C | Off (false) |
| F11 | Asteroid visibility toggle | Toggle asteroids on. 6 named asteroids should appear. Toggle off. They disappear. | On: Ceres, Vesta, Pallas, Hygiea, Apophis, Bennu visible. Off: no named asteroids. | Manual | C | Off (false) |
| F12 | Belt visibility toggle | Toggle belt off. The 1500-particle asteroid belt disappears. Toggle on. It reappears. | Off: no belt particles visible between Mars and Jupiter. On: belt particles visible. Toggle does not affect named asteroids (separate toggle). | Manual | C | On (true) |
| F13 | Label visibility toggle | Toggle labels off. Planet names disappear. Toggle on. They reappear. | Off: no text labels. On: each body shows its name. | Manual | C | On (true) |
| F14 | Reset returns to defaults | Change multiple settings (trails off, speed 10x, GR off). Click reset. All should return to defaults. | After reset: trails=on, spacetime=off, labels=on, planetGravity=on, relativity=on, integrator=Verlet, timeSpeed=1.0, isPlaying=true, moons=on, comets=off, asteroids=off, belt=on. | Yes (unit test: compare state to defaults table) | C | See defaults table above |
| F15 | Sun mass slider affects orbits | Increase sun mass to 2x. Planets should orbit faster (shorter periods). Decrease to 0.5x. Orbits widen and slow. | At 2x mass: orbital period decreases by factor ~sqrt(2)/2. Orbits do not immediately collapse or diverge. At 0.5x: periods increase. | Manual (observe Earth orbit period change) | C | 1.0x |

---

## Launch Planner Regression Tests

| # | Feature | How to Test | Acceptance Criteria | Phase |
|---|---------|------------|--------------------| ------|
| L1 | LEO mission delta-v | Select Generic vehicle, LEO destination. Simulate. | Delta-v budget matches Go output (within 0.1 m/s). Feasibility assessment correct. | D |
| L2 | Moon mission trajectory | Select Saturn V-like vehicle, Moon destination. Simulate. | Trajectory renders correctly. Transfer time ~3 days. TLI + lunar orbit insertion delta-v computed. | D |
| L3 | Mars mission trajectory | Select Falcon-like vehicle, Mars destination. Simulate. | Hohmann transfer time ~259 days. Hyperbolic excess delta-v included. | D |
| L4 | Mission playback controls | Start a mission, then play/pause/scrub the timeline. | Playback: vehicle marker moves along trajectory. Pause: marker stops. Scrub: marker jumps to scrubbed position. Speed control works (1x to 64x). | D |
| L5 | Telemetry display during playback | During mission playback, telemetry panel shows live data. | Altitude, velocity, acceleration, and distance from Earth update continuously during playback. Values are physically plausible. | D |
| L6 | CSV export | After simulation, export trajectory to CSV. | CSV file contains columns: Time, X, Y, Z, Vx, Vy, Vz, acceleration, distance. Values match simulation output. | D |

---

## Camera Regression Tests

| # | Feature | How to Test | Acceptance Criteria | Phase |
|---|---------|------------|--------------------| ------|
| C1 | Scroll wheel zoom | Scroll up and down. | Zoom in/out is smooth and continuous. Logarithmic progression. Minimum and maximum zoom limits respected. | B |
| C2 | Drag to rotate | Right-click drag (or primary drag in Bevy). | View rotates around the focus point. Pitch clamped to +/- 90 degrees. | B |
| C3 | Shift+drag to pan (or middle-click) | Middle-click drag. | Focus point moves in the camera's local XY plane. Panning does not affect zoom or rotation. | B |
| C4 | WASD keyboard pan | Press W/A/S/D keys. | Camera focus point moves relative to the camera's forward direction. Movement speed is proportional to current zoom level. | B |
| C5 | Q/E keyboard rotation | Press Q/E keys. | View rotates (yaw) around the focus point. | B |
| C6 | R/F keyboard zoom | Press R/F keys. | Zoom in (R) and out (F). Same behavior as scroll wheel. | B |
| C7 | Follow body + zoom/rotate | Follow a planet, then zoom and rotate. | Zoom and rotation operate relative to the followed body. Camera stays centered on the body. | B |
| C8 | Auto-fit all planets | Trigger auto-fit (if implemented). | Camera zooms and repositions to show all planets with ~10% margin. | C |
| C9 | 3D view toggle | Enable 3D mode or drag to auto-enable. | View transitions from top-down 2D to 3D perspective. Pitch/yaw sliders become active. | C |

---

## Screenshot Comparison Methodology

For visual regression tests (R1-R21), the recommended process is:

1. **Capture Go baseline screenshots** at specific simulation times (t=0, t=100 steps, t=1000 steps) with a fixed camera position and known state.
2. **Capture Bevy screenshots** at the same simulation times with equivalent camera position and state.
3. **Compare** using perceptual image diff (e.g., `pixelmatch` or SSIM). Threshold: SSIM > 0.85 for overall layout, allowing for expected rendering differences (PBR vs CPU shading).
4. **Per-feature comparison**: For features like bloom, trails, and spacetime grid, compare feature presence rather than pixel-exact match.

Not all visual tests can be fully automated due to rendering pipeline differences (Fyne CPU vs Bevy PBR). The goal is functional equivalence, not pixel identity.
