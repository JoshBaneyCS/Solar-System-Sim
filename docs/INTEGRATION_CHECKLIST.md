# Integration Checklist

## Agent 01: UI State Sync
- [ ] `internal/ui/state.go` created with `AppState` struct
- [ ] All toggles route through `AppState` setters
- [ ] Menu items update via listeners
- [ ] Settings dialog reads/writes through `AppState`
- [ ] Reset simulation resets `AppState`
- [ ] `state_test.go` passes

## Agent 05: Camera Controls
- [ ] `internal/ui/input_handler.go` created
- [ ] Mouse drag orbits (RotationX/Y)
- [ ] Scroll wheel zooms exponentially
- [ ] WASD pans, R/F zooms vertically
- [ ] Focus-on-click keyboard capture
- [ ] Viewport methods: `AdjustZoom`, `AdjustRotation`

## Agent 02: Texture/Materials
- [ ] `internal/render/textures.go` created
- [ ] `internal/assets/resolve.go` created
- [ ] All planet textures load at startup
- [ ] Circular-masked textures replace solid circles
- [ ] Fallback to solid color on missing texture
- [ ] Image pooling in `RenderCache`

## Agent 03: Lighting
- [ ] `internal/render/lighting.go` created
- [ ] Lambertian diffuse from Sun position
- [ ] Day/night terminator visible on planets
- [ ] Sun emissive glow effect
- [ ] Shading cached, invalidated on angle change

## Agent 04: Performance
- [ ] `internal/ui/diagnostics.go` created
- [ ] Apple Silicon detection
- [ ] Trail LOD at low zoom
- [ ] Dirty flag skip when viewport static
- [ ] Decoupled sim/render tick rates
- [ ] FPS counter in View menu

## Agent 06: Launch Visualization
- [ ] `internal/ui/mission_playback.go` created
- [ ] Animated vehicle marker on trajectory
- [ ] Timeline scrubber slider
- [ ] Play/Pause/speed controls
- [ ] Telemetry HUD (time, speed, distance)

## Agent 07: GUI Overhaul
- [ ] `internal/ui/theme.go` created
- [ ] Custom dark space theme applied
- [ ] `internal/ui/statusbar.go` created
- [ ] Accordion sidebar layout
- [ ] Enhanced About dialog

## Agent 08: Test Automation
- [ ] `viewport_test.go` — WorldToScreen transforms
- [ ] `textures_test.go` — Texture loading
- [ ] `lighting_test.go` — Shading correctness
- [ ] `state_test.go` — State sync
- [ ] `diagnostics_test.go` — Runtime detection
- [ ] `mission_playback_test.go` — Interpolation
- [ ] `resolve_test.go` — Asset resolution
- [ ] CI race detection enabled

## Agent 09: Packaging
- [ ] Asset resolver centralized
- [ ] macOS .app bundle includes textures
- [ ] Windows .zip includes textures
- [ ] Ad-hoc codesigning for macOS
- [ ] Post-package validation

## Agent 10: CI/CD
- [ ] `ci.yml` uses go-version-file
- [ ] Race detector on Linux tests
- [ ] Asset validation step
- [ ] `release.yml` artifact verification
- [ ] `lint.yml` workflow created
