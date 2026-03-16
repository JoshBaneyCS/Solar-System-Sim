# Bevy Plugin Layout

All plugins are defined in `crates/solar_sim_bevy/src/plugins/`. The main app registers them in dependency order.

## App Registration

```rust
// crates/solar_sim_bevy/src/main.rs

use bevy::prelude::*;

mod bundles;
mod components;
mod events;
mod plugins;
mod resources;

fn main() {
    App::new()
        .add_plugins(DefaultPlugins.set(WindowPlugin {
            primary_window: Some(Window {
                title: "Solar System Simulator".into(),
                resolution: (1600., 900.).into(),
                ..default()
            }),
            ..default()
        }))
        .add_plugins((
            plugins::PhysicsPlugin,
            plugins::CelestialRenderPlugin,
            plugins::TrailPlugin,
            plugins::BeltPlugin,
            plugins::CameraPlugin,
            plugins::UIPlugin,
            plugins::SpacetimePlugin,
            plugins::LaunchPlugin,
        ))
        .run();
}
```

---

## PhysicsPlugin

**Responsibility:** Runs the N-body simulation at a fixed timestep, manages body lifecycle (add/remove moons, comets, asteroids), processes simulation commands from UI.

**Dependencies:** None (foundational plugin).

```rust
pub struct PhysicsPlugin;

impl Plugin for PhysicsPlugin {
    fn build(&self, app: &mut App) {
        app
            // Resources
            .insert_resource(SimulationConfig::default())
            .insert_resource(SimulationClock::default())
            .insert_resource(SimulationState::new())
            // Events
            .add_event::<SimCommand>()
            // Fixed timestep configuration
            .insert_resource(Time::<Fixed>::from_hz(60.0))
            // Systems
            .add_systems(Startup, spawn_solar_system)
            .add_systems(FixedUpdate, (
                process_sim_commands,
                step_simulation,
                sync_ecs_from_simulation,
                manage_trails,
            ).chain())
            .add_systems(Update, update_simulation_clock);
    }
}
```

### Systems

| System | Schedule | Description |
|--------|----------|-------------|
| `spawn_solar_system` | Startup | Creates Sun + 9 planets + moons as entities with `PlanetBundle`. Initializes `SimulationState` with body catalog from `solar_sim_core`. |
| `process_sim_commands` | FixedUpdate | Reads `EventReader<SimCommand>`, applies mutations to `SimulationConfig` and `SimulationState`. Handles add/remove body type commands. |
| `step_simulation` | FixedUpdate | If `config.is_playing`, computes `effective_dt = fixed_dt * time_speed`, runs substep subdivision if `|effective_dt| > MAX_SAFE_DT`, calls `sim.step()` for each substep. |
| `sync_ecs_from_simulation` | FixedUpdate | After physics step, copies positions/velocities from `SimulationState.inner` back to ECS `Transform` and `Velocity` components. Scales from meters to rendering units. |
| `manage_trails` | FixedUpdate | If trails enabled, appends current position to each body's `Orbit.trail_history`. Truncates to `max_trail_len`. |
| `update_simulation_clock` | Update | Updates `SimulationClock` resource with current simulation time, formatted time strings for UI display. |

### Resources Created

- `SimulationConfig` -- time_speed, is_playing, integrator, planet_gravity, relativity, show_trails, etc.
- `SimulationClock` -- current_time_seconds, elapsed_days, elapsed_years
- `SimulationState` -- wraps `solar_sim_core::Simulation`, owns the authoritative physics state

### Components Used

- Reads: `CelestialBody` (to map entity to sim index)
- Writes: `Transform`, `Velocity`, `Orbit`

### Events

- Receives: `SimCommand` (from UIPlugin)
- Sends: none

---

## CelestialRenderPlugin

**Responsibility:** Manages planet/moon/comet/asteroid mesh entities, textures, PBR materials, sun glow via bloom, labels, and dynamic display radius scaling.

**Dependencies:** PhysicsPlugin (reads `Transform`, `CelestialBody`).

```rust
pub struct CelestialRenderPlugin;

impl Plugin for CelestialRenderPlugin {
    fn build(&self, app: &mut App) {
        app
            .add_systems(Startup, (
                setup_lighting,
                setup_skybox,
            ))
            .add_systems(PostUpdate, (
                update_display_radius,
                update_label_positions,
                toggle_label_visibility,
                update_comet_tails,
            ));
    }
}
```

### Systems

| System | Schedule | Description |
|--------|----------|-------------|
| `setup_lighting` | Startup | Spawns a `PointLight` at the Sun's position with high intensity and range. Configures `BloomSettings` on the camera for sun glow. |
| `setup_skybox` | Startup | Loads the milky way texture and applies it as a `Skybox` component on the camera entity. |
| `update_display_radius` | PostUpdate | For each body, computes display scale from `PhysicalRadius` and camera distance. Updates `Transform::scale`. Clamps to minimum pixel size (4px) and maximum (5000px). |
| `update_label_positions` | PostUpdate | Positions `Text2d` label entities near their parent body's screen position. Offsets below the body sphere. |
| `toggle_label_visibility` | PostUpdate | Reads `SimulationConfig.show_labels` and sets `Visibility` on all label entities. |
| `update_comet_tails` | PostUpdate | For entities with `CometTail` marker, computes tail direction (away from Sun), updates a particle emitter or billboard gradient quad. Tail length inversely proportional to Sun distance. |

### Resources Created

None (uses Bevy's built-in `AssetServer`, `Materials`, `Meshes`).

### Components Used

- Reads: `CelestialBody`, `PhysicalRadius`, `Transform`, `CometTail`
- Writes: `Transform` (scale), `Visibility`
- Spawns children: `Text2d` labels per body

### Events

- Receives: none
- Sends: none

---

## TrailPlugin

**Responsibility:** Renders orbital trails as GPU-accelerated polylines. Manages trail visual state independently from trail data (which lives in PhysicsPlugin).

**Dependencies:** PhysicsPlugin (reads `Orbit` component).

```rust
pub struct TrailPlugin;

impl Plugin for TrailPlugin {
    fn build(&self, app: &mut App) {
        app
            .add_plugins(bevy_polyline::PolylinePlugin)
            .add_systems(Update, (
                update_trail_polylines,
                toggle_trail_visibility,
            ));
    }
}
```

### Systems

| System | Schedule | Description |
|--------|----------|-------------|
| `update_trail_polylines` | Update | For each entity with `Orbit` where `show_trail` is true, updates the `Polyline` vertices from `trail_history`. Applies Catmull-Rom interpolation (4 sub-segments per trail segment). Downsamples to max 200 visual segments. Sets per-vertex alpha (old=faint, new=opaque). |
| `toggle_trail_visibility` | Update | Reads `SimulationConfig.show_trails`. When disabled globally, hides all trail polyline entities. Per-body `Orbit.show_trail` also respected. |

### Resources Created

None.

### Components Used

- Reads: `Orbit` (trail_history, show_trail), `CelestialBody` (color)
- Writes: `Polyline` vertex data, `Visibility`

---

## BeltPlugin

**Responsibility:** Renders 1500 visual asteroid belt particles using Keplerian orbital mechanics. Particles are NOT N-body simulated; their positions are computed analytically each frame.

**Dependencies:** PhysicsPlugin (reads `SimulationClock`).

```rust
pub struct BeltPlugin;

impl Plugin for BeltPlugin {
    fn build(&self, app: &mut App) {
        app
            .insert_resource(BeltConfig {
                particle_count: 1500,
                visible: true,
            })
            .add_systems(Startup, spawn_belt_particles)
            .add_systems(Update, (
                update_belt_positions,
                toggle_belt_visibility,
            ));
    }
}
```

### Systems

| System | Schedule | Description |
|--------|----------|-------------|
| `spawn_belt_particles` | Startup | Spawns 1500 entities with `BeltParticle` component and tiny sphere `Mesh` + gray `StandardMaterial`. Orbital elements randomized between 2.1-3.3 AU with Kirkwood gaps excluded (2.5, 2.82, 2.95 AU). Uses instanced rendering (all share same mesh handle). |
| `update_belt_positions` | Update | For each `BeltParticle`, computes position from Keplerian elements using current simulation time. Solves Kepler equation with 5 Newton iterations. Updates `Transform.translation`. |
| `toggle_belt_visibility` | Update | Reads `BeltConfig.visible` (set by UIPlugin). Sets `Visibility` on all belt particle entities. |

### Resources Created

- `BeltConfig` -- particle_count, visible

### Components Used

- Reads: `BeltParticle` (semi_major_axis, eccentricity, inclination, omega, initial_anomaly, mean_motion)
- Writes: `Transform`

---

## CameraPlugin

**Responsibility:** Orbit camera with mouse/keyboard controls, follow-body tracking, zoom, pan, 3D rotation, auto-fit.

**Dependencies:** PhysicsPlugin (reads `Transform` of followed body).

```rust
pub struct CameraPlugin;

impl Plugin for CameraPlugin {
    fn build(&self, app: &mut App) {
        app
            .insert_resource(OrbitCamera::default())
            .add_systems(Startup, spawn_camera)
            .add_systems(Update, (
                camera_zoom_input,
                camera_pan_input,
                camera_rotate_input,
                camera_follow_body,
                camera_apply_transform,
                camera_auto_fit,
            ).chain());
    }
}
```

### Systems

| System | Schedule | Description |
|--------|----------|-------------|
| `spawn_camera` | Startup | Spawns `Camera3d` with `BloomSettings`, `Skybox`, and `Msaa::Sample4`. Initial position looking down at ecliptic plane. |
| `camera_zoom_input` | Update | Reads `MouseWheel` events and R/F keys. Updates `OrbitCamera.distance`. Range: 0.01x to 10,000,000x (logarithmic). |
| `camera_pan_input` | Update | Reads Shift+drag and WASD keys. Updates `OrbitCamera.pan_offset`. |
| `camera_rotate_input` | Update | Reads mouse drag (without Shift) and Q/E keys. Updates `OrbitCamera.yaw`, `OrbitCamera.pitch`, `OrbitCamera.roll`. Pitch clamped to +/-89 degrees. |
| `camera_follow_body` | Update | If `OrbitCamera.follow_target` is `Some(entity)`, reads that entity's `Transform` and sets `OrbitCamera.focus_point` to its position. |
| `camera_apply_transform` | Update | Computes final camera `Transform` from `OrbitCamera` state: focus_point + spherical offset (distance, yaw, pitch) + pan. |
| `camera_auto_fit` | Update | Triggered by `AutoFitEvent`. Computes bounding sphere of all `CelestialBody` entities, sets camera distance and focus to fit with 10% margin. |

### Resources Created

- `OrbitCamera` -- distance, yaw, pitch, roll, pan_offset, focus_point, follow_target, zoom_level

### Components Used

- Reads: `Transform` (of followed body), `CelestialBody` (for auto-fit bounding)
- Writes: `Transform` (of camera entity)

### Events

- Receives: `AutoFitEvent` (from UIPlugin)
- Sends: none

---

## UIPlugin

**Responsibility:** All egui-based UI panels: simulation controls, bodies list, launch planner, physics display, status bar, settings, about dialog.

**Dependencies:** PhysicsPlugin, CameraPlugin, LaunchPlugin (reads/writes their resources).

```rust
pub struct UIPlugin;

impl Plugin for UIPlugin {
    fn build(&self, app: &mut App) {
        app
            .add_plugins(bevy_egui::EguiPlugin)
            .insert_resource(UIPanelState::default())
            .add_systems(Update, (
                ui_controls_panel,
                ui_bodies_panel,
                ui_physics_panel,
                ui_launch_panel,
                ui_status_bar,
                ui_settings_dialog,
                ui_about_dialog,
                handle_keyboard_shortcuts,
            ));
    }
}
```

### Systems

| System | Schedule | Description |
|--------|----------|-------------|
| `ui_controls_panel` | Update | Left sidebar: Play/Pause, speed slider, zoom, follow-body dropdown, 3D rotation sliders, display toggles (trails, spacetime, labels, belt), celestial body toggles (moons, comets, asteroids), physics options (planet gravity, relativity, integrator), sun mass slider, reset. Sends `SimCommand` events. |
| `ui_bodies_panel` | Update | Tab: lists all bodies grouped by type. Per-body: name, distance from Sun, velocity, orbital period, follow button, trail toggle. |
| `ui_physics_panel` | Update | Right panel: live physics equations, Earth orbital parameters, force vectors, simulation time. Updated each frame from `SimulationState`. |
| `ui_launch_panel` | Update | Tab: destination selector, vehicle selector, simulate button, results display, mission playback controls. Sends `LaunchCommand` events. |
| `ui_status_bar` | Update | Bottom bar: FPS, simulation time, speed, zoom, system info. Updated every 4th frame. |
| `ui_settings_dialog` | Update | Modal: display toggles, physics options, integrator. Persists to disk via `ron` serialization. |
| `ui_about_dialog` | Update | Modal: version, author, system info, credits, links. |
| `handle_keyboard_shortcuts` | Update | Space=play/pause, +/-=speed, Escape=deselect, F11=fullscreen. |

### Resources Created

- `UIPanelState` -- which panels are open, dialog state, transient UI state

### Components Used

- Reads: `CelestialBody`, `Transform`, `Velocity` (for bodies panel)
- Writes: none directly (sends events)

### Events

- Sends: `SimCommand`, `AutoFitEvent`, `BodySelected`, `LaunchCommand`
- Receives: `LaunchComputed` (to display results)

---

## SpacetimePlugin

**Responsibility:** Visualizes gravitational curvature as a deformed grid with heat-map coloring.

**Dependencies:** PhysicsPlugin (reads body positions/masses).

```rust
pub struct SpacetimePlugin;

impl Plugin for SpacetimePlugin {
    fn build(&self, app: &mut App) {
        app
            .insert_resource(SpacetimeConfig {
                enabled: false,
                grid_resolution: 80,
                cache_valid: false,
            })
            .add_systems(Update, (
                compute_spacetime_grid,
                update_spacetime_mesh,
                toggle_spacetime_visibility,
            ).chain());
    }
}
```

### Systems

| System | Schedule | Description |
|--------|----------|-------------|
| `compute_spacetime_grid` | Update | When enabled and cache invalid, calls `solar_sim_core::spacetime::compute_potential_field()` with current body positions/masses and camera bounds. Adaptive resolution: 40-150 lines based on zoom. Stores result in `SpacetimeConfig`. Invalidates cache when camera moves > 5%. |
| `update_spacetime_mesh` | Update | Generates a dynamic `Mesh` from the computed grid data. Vertex positions are warped by potential. Vertex colors follow gradient: purple (low) -> red (medium) -> orange (high). Updates mesh handle each frame the grid changes. |
| `toggle_spacetime_visibility` | Update | Reads `SimulationConfig.show_spacetime`. Sets `Visibility` on the spacetime mesh entity. |

### Resources Created

- `SpacetimeConfig` -- enabled, grid_resolution, cached potential field, cache_valid flag

### Components Used

- Reads: `CelestialBody`, `Transform` (body positions for potential computation)
- Writes: `Mesh` (dynamic grid mesh)

---

## LaunchPlugin

**Responsibility:** Hohmann transfer planner, trajectory overlay rendering, mission playback, vehicle/destination management.

**Dependencies:** PhysicsPlugin (reads `SimulationClock`, body positions).

```rust
pub struct LaunchPlugin;

impl Plugin for LaunchPlugin {
    fn build(&self, app: &mut App) {
        app
            .insert_resource(LaunchState::default())
            .add_event::<LaunchCommand>()
            .add_event::<LaunchComputed>()
            .add_systems(Update, (
                process_launch_commands,
                update_mission_playback,
                render_trajectory_overlay,
                render_vehicle_marker,
            ));
    }
}
```

### Systems

| System | Schedule | Description |
|--------|----------|-------------|
| `process_launch_commands` | Update | Reads `LaunchCommand` events (Simulate, Clear). On Simulate: calls `solar_sim_core::launch::Planner::plan()` and `propagate_trajectory()`. Stores result in `LaunchState`. Sends `LaunchComputed` event. |
| `update_mission_playback` | Update | If playback is active, advances playback time by `real_dt * playback_speed`. Interpolates vehicle position along trajectory. Updates playback telemetry (altitude, velocity, acceleration, distance). |
| `render_trajectory_overlay` | Update | If a trajectory exists, renders it as a `Polyline` with color gradient (green at departure -> red at arrival). For heliocentric trajectories, positions are in world space. For Earth-centered, positions offset by Earth's current position. |
| `render_vehicle_marker` | Update | During playback, renders a small green sphere at the interpolated vehicle position. |

### Resources Created

- `LaunchState` -- current plan, trajectory, playback state, playback_time, playback_speed, playback_active

### Components Used

- Reads: `CelestialBody` (to find Earth position for trajectory offset)
- Writes: `Polyline` (trajectory), `Transform` (vehicle marker)

### Events

- Receives: `LaunchCommand` (from UIPlugin)
- Sends: `LaunchComputed` (to UIPlugin for results display)

---

## Plugin Dependency Summary

```
                    PhysicsPlugin (foundational)
                   /      |      \        \
                  /       |       \        \
    CelestialRenderPlugin |   CameraPlugin  SpacetimePlugin
                    TrailPlugin    |
                    BeltPlugin     |
                                   |
                              LaunchPlugin
                                   |
                              UIPlugin (depends on all)
```

All plugins are loosely coupled through Bevy's ECS -- they share data via components, resources, and events rather than direct function calls. Any plugin can be removed without breaking the others (UI panels for missing features simply won't render).
