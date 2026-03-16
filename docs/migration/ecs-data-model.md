# ECS Data Model

Complete definition of all Bevy ECS components, resources, events, and bundles for the Solar System Simulator.

---

## Components

### CelestialBody

Maps an ECS entity to its index in the `solar_sim_core::Simulation` struct. Carries immutable body metadata.

```rust
#[derive(Component)]
pub struct CelestialBody {
    /// Index into SimulationState.inner.positions/velocities/masses
    pub sim_index: usize,
    /// Human-readable name ("Earth", "Io", "Halley")
    pub name: String,
    /// Mass in kg
    pub mass: f64,
    /// Body classification for rendering and UI grouping
    pub body_type: BodyType,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum BodyType {
    Star,
    Planet,
    DwarfPlanet,
    Moon,
    Comet,
    Asteroid,
}
```

### Velocity

Current velocity vector in meters/second. Synced from `solar_sim_core` each physics tick.

```rust
/// Velocity in m/s. Stored as f64 for physics precision.
/// The corresponding Transform uses f32 for rendering.
#[derive(Component, Default)]
pub struct Velocity(pub DVec3);
```

Note: We use `DVec3` (f64) for physics precision. Bevy's `Transform` uses `Vec3` (f32) for rendering, which is sufficient for visual positions but not for physics. The authoritative f64 state lives in `SimulationState.inner`.

### Orbit

Trail history and visibility for orbital path rendering.

```rust
use std::collections::VecDeque;

#[derive(Component)]
pub struct Orbit {
    /// Ring buffer of past positions (meters, f64).
    /// Managed by PhysicsPlugin's manage_trails system.
    pub trail_history: VecDeque<DVec3>,
    /// Maximum number of trail points to retain.
    pub max_trail_len: usize,
    /// Whether this body's trail is visible.
    pub show_trail: bool,
    /// Entity ID of the associated Polyline entity (if spawned).
    pub polyline_entity: Option<Entity>,
}

impl Default for Orbit {
    fn default() -> Self {
        Self {
            trail_history: VecDeque::with_capacity(500),
            max_trail_len: 500,
            show_trail: true,
            polyline_entity: None,
        }
    }
}
```

### PhysicalRadius

Real physical radius in meters. Used for zoom-dependent display scaling.

```rust
/// Real physical radius in meters.
/// Earth = 6.371e6, Jupiter = 6.9911e7, Sun = 6.9634e8
#[derive(Component)]
pub struct PhysicalRadius(pub f64);
```

### DisplayRadius

Base display radius in pixels (from the Go `Body.Radius` field). Used as minimum visible size.

```rust
/// Minimum display radius in pixels when zoomed out.
/// Ensures small bodies remain visible.
#[derive(Component)]
pub struct DisplayRadius(pub f32);
```

### FollowTarget

Marker component. The camera tracks the entity that has this component.

```rust
/// Marker: the camera follows this entity.
/// At most one entity should have this at any time.
/// Managed by CameraPlugin when user selects a follow target.
#[derive(Component)]
pub struct FollowTarget;
```

### Selected

Marker component for distance measurement. Exactly 0 or 2 entities may be selected at once.

```rust
/// Marker: this body is selected for distance measurement.
/// When exactly 2 entities have this, a distance line is drawn between them.
#[derive(Component)]
pub struct Selected;
```

### CometTail

Marker component that enables comet tail rendering for this entity.

```rust
/// Marker: this entity renders a comet tail (gradient away from Sun).
#[derive(Component)]
pub struct CometTail;
```

### BeltParticle

Keplerian orbital elements for an asteroid belt visual particle. NOT N-body simulated.

```rust
/// Asteroid belt visual particle. Position computed analytically each frame.
#[derive(Component)]
pub struct BeltParticle {
    pub semi_major_axis: f64,    // meters
    pub eccentricity: f64,
    pub inclination: f64,        // radians
    pub long_ascending_node: f64,// radians
    pub arg_perihelion: f64,     // radians
    pub initial_anomaly: f64,    // radians (mean anomaly at t=0)
    pub mean_motion: f64,        // radians/second
}
```

### BodyLabel

Links a body entity to its text label entity.

```rust
/// Links a celestial body to its text label child entity.
#[derive(Component)]
pub struct BodyLabel {
    pub label_entity: Entity,
}
```

### SpacetimeGrid

Marker component for the spacetime visualization mesh entity.

```rust
/// Marker: this entity is the spacetime curvature grid mesh.
#[derive(Component)]
pub struct SpacetimeGrid;
```

### TrajectoryOverlay

Marker component for the launch trajectory polyline entity.

```rust
/// Marker: this entity is the launch trajectory visualization.
#[derive(Component)]
pub struct TrajectoryOverlay;
```

### VehicleMarker

Marker component for the launch vehicle position indicator.

```rust
/// Marker: this entity is the green dot showing launch vehicle position
/// during mission playback.
#[derive(Component)]
pub struct VehicleMarker;
```

---

## Resources

### SimulationConfig

Central configuration resource. Replaces Go's `AppState`. All UI panels read from and write to this.

```rust
#[derive(Resource)]
pub struct SimulationConfig {
    // Time control
    pub time_speed: f64,           // Multiplier: 2^(-10) to 2^10. Negative = rewind.
    pub is_playing: bool,
    pub fixed_dt: f64,             // Base timestep in seconds (e.g., 86400.0 = 1 day)

    // Physics options
    pub integrator: IntegratorType,
    pub planet_gravity: bool,
    pub relativity: bool,
    pub sun_mass_multiplier: f64,  // 0.1 to 5.0

    // Display toggles
    pub show_trails: bool,
    pub show_spacetime: bool,
    pub show_labels: bool,
    pub show_belt: bool,

    // Body visibility
    pub show_moons: bool,
    pub show_comets: bool,
    pub show_asteroids: bool,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum IntegratorType {
    Verlet,
    RK4,
}

impl Default for SimulationConfig {
    fn default() -> Self {
        Self {
            time_speed: 1.0,
            is_playing: true,
            fixed_dt: 86400.0,
            integrator: IntegratorType::Verlet,
            planet_gravity: true,
            relativity: true,
            sun_mass_multiplier: 1.0,
            show_trails: true,
            show_spacetime: false,
            show_labels: true,
            show_belt: true,
            show_moons: true,
            show_comets: false,
            show_asteroids: false,
        }
    }
}
```

### SimulationClock

Derived time values for UI display. Updated each frame from `SimulationState`.

```rust
#[derive(Resource, Default)]
pub struct SimulationClock {
    /// Raw simulation time in seconds since epoch
    pub current_time: f64,
    /// Simulation time in days
    pub elapsed_days: f64,
    /// Simulation time in years
    pub elapsed_years: f64,
}
```

### SimulationState

Wraps the `solar_sim_core::Simulation` struct. This is the authoritative physics state. ECS `Transform`/`Velocity` components are synced FROM this each tick, not the other way around.

```rust
use solar_sim_core::sim::Simulation;

#[derive(Resource)]
pub struct SimulationState {
    /// The authoritative physics simulation (f64 precision).
    pub inner: Simulation,
    /// Map from entity to sim_index, for dynamic add/remove.
    pub entity_map: Vec<Entity>,
}
```

### OrbitCamera

Camera controller state. The CameraPlugin reads this to compute the final camera `Transform`.

```rust
#[derive(Resource)]
pub struct OrbitCamera {
    /// Distance from focus point (in rendering units)
    pub distance: f64,
    /// Horizontal angle in radians
    pub yaw: f64,
    /// Vertical angle in radians (clamped to +/- 89 degrees)
    pub pitch: f64,
    /// Roll angle in radians
    pub roll: f64,
    /// Camera pan offset (world units)
    pub pan_offset: DVec3,
    /// Point the camera orbits around (world units)
    pub focus_point: DVec3,
    /// Entity to follow (None = free camera)
    pub follow_target: Option<Entity>,
    /// Zoom level for UI display (logarithmic)
    pub zoom_level: f64,
}

impl Default for OrbitCamera {
    fn default() -> Self {
        Self {
            distance: 50.0,
            yaw: 0.0,
            pitch: std::f64::consts::FRAC_PI_4, // 45 degrees down
            roll: 0.0,
            pan_offset: DVec3::ZERO,
            focus_point: DVec3::ZERO,
            follow_target: None,
            zoom_level: 1.0,
        }
    }
}
```

### BeltConfig

Configuration for asteroid belt rendering.

```rust
#[derive(Resource)]
pub struct BeltConfig {
    pub particle_count: usize,
    pub visible: bool,
}
```

### SpacetimeConfig

Configuration and cached data for spacetime curvature visualization.

```rust
#[derive(Resource)]
pub struct SpacetimeConfig {
    pub enabled: bool,
    pub grid_resolution: usize,
    /// Cached potential field values (flattened 2D grid).
    pub potential_field: Vec<f64>,
    /// Cached grid positions before warping (for cache invalidation).
    pub cached_bounds: (DVec3, DVec3),
    pub cache_valid: bool,
}
```

### LaunchState

State for the launch planner and mission playback.

```rust
use solar_sim_core::launch::{LaunchPlan, Trajectory};

#[derive(Resource, Default)]
pub struct LaunchState {
    /// Most recent launch plan result
    pub plan: Option<LaunchPlan>,
    /// Propagated trajectory points
    pub trajectory: Option<Trajectory>,
    /// Selected vehicle index
    pub vehicle_index: usize,
    /// Selected destination index
    pub destination_index: usize,

    // Mission playback
    pub playback_active: bool,
    pub playback_time: f64,      // seconds into trajectory
    pub playback_speed: f64,     // 1x, 2x, 4x, ..., 64x
    pub playback_paused: bool,
    /// Interpolated world position of vehicle during playback
    pub vehicle_world_pos: Option<DVec3>,
}
```

### UIPanelState

Transient UI state (which dialogs are open, scroll positions, etc.).

```rust
#[derive(Resource, Default)]
pub struct UIPanelState {
    pub settings_open: bool,
    pub about_open: bool,
    pub active_left_tab: LeftTab,
}

#[derive(Default, Clone, Copy, PartialEq, Eq)]
pub enum LeftTab {
    #[default]
    Simulation,
    LaunchPlanner,
    Bodies,
}
```

---

## Events

### SimCommand

UI-to-physics commands. Replaces the Go `SimCommand` channel pattern. In Bevy, these are events processed by `process_sim_commands` system.

```rust
#[derive(Event)]
pub enum SimCommand {
    // Time control
    SetPlaying(bool),
    SetTimeSpeed(f64),

    // Physics toggles
    SetPlanetGravity(bool),
    SetRelativity(bool),
    SetIntegrator(IntegratorType),
    SetSunMass(f64),  // multiplier

    // Display toggles
    SetShowTrails(bool),
    SetShowSpacetime(bool),
    SetShowLabels(bool),
    SetShowBelt(bool),

    // Body management
    SetShowMoons(bool),
    SetShowComets(bool),
    SetShowAsteroids(bool),

    // Camera
    FollowBody(Option<Entity>),
    AutoFit,

    // Simulation lifecycle
    Reset,
    ClearTrails,
}
```

### BodySelected

Fired when a user clicks on a celestial body. Used for distance measurement (2 selections) and body info display.

```rust
#[derive(Event)]
pub struct BodySelected {
    pub entity: Entity,
}
```

### LaunchCommand

UI-to-launch-planner commands.

```rust
#[derive(Event)]
pub enum LaunchCommand {
    /// Compute a launch plan with current vehicle/destination selection
    Simulate,
    /// Clear the current trajectory and plan
    Clear,
    /// Start mission playback
    PlaybackStart,
    /// Pause/resume playback
    PlaybackToggle,
    /// Set playback speed (1x, 2x, 4x, ..., 64x)
    PlaybackSetSpeed(f64),
    /// Seek to a specific time in the trajectory
    PlaybackSeek(f64),
}
```

### LaunchComputed

Sent by LaunchPlugin when a plan computation completes. Received by UIPlugin to display results.

```rust
#[derive(Event)]
pub struct LaunchComputed {
    pub plan: LaunchPlan,
    pub feasible: bool,
}
```

### AutoFitEvent

Triggers the camera to auto-fit all visible bodies.

```rust
#[derive(Event)]
pub struct AutoFitEvent;
```

---

## Bundles

### PlanetBundle

Spawned for each planet, dwarf planet, and named asteroid during `spawn_solar_system`.

```rust
#[derive(Bundle)]
pub struct PlanetBundle {
    // Identification
    pub body: CelestialBody,
    pub velocity: Velocity,
    pub orbit: Orbit,
    pub physical_radius: PhysicalRadius,
    pub display_radius: DisplayRadius,

    // Bevy rendering
    pub mesh: Mesh3d,
    pub material: MeshMaterial3d<StandardMaterial>,
    pub transform: Transform,
    pub global_transform: GlobalTransform,
    pub visibility: Visibility,
    pub inherited_visibility: InheritedVisibility,
    pub view_visibility: ViewVisibility,
}
```

Usage:
```rust
fn spawn_planet(
    commands: &mut Commands,
    meshes: &mut ResMut<Assets<Mesh>>,
    materials: &mut ResMut<Assets<StandardMaterial>>,
    asset_server: &Res<AssetServer>,
    body_data: &BodyCatalogEntry,
    sim_index: usize,
) -> Entity {
    let texture_handle: Handle<Image> = asset_server
        .load(format!("textures/{}/albedo.jpg", body_data.name.to_lowercase()));

    commands.spawn(PlanetBundle {
        body: CelestialBody {
            sim_index,
            name: body_data.name.clone(),
            mass: body_data.mass,
            body_type: body_data.body_type,
        },
        velocity: Velocity(DVec3::ZERO),
        orbit: Orbit::default(),
        physical_radius: PhysicalRadius(body_data.physical_radius),
        display_radius: DisplayRadius(body_data.display_radius),
        mesh: Mesh3d(meshes.add(Sphere::new(1.0).mesh().uv(32, 18))),
        material: MeshMaterial3d(materials.add(StandardMaterial {
            base_color_texture: Some(texture_handle),
            perceptual_roughness: 0.8,
            metallic: 0.0,
            ..default()
        })),
        transform: Transform::from_xyz(0.0, 0.0, 0.0),
        global_transform: GlobalTransform::default(),
        visibility: Visibility::Visible,
        inherited_visibility: InheritedVisibility::default(),
        view_visibility: ViewVisibility::default(),
    }).id()
}
```

### MoonBundle

Identical structure to `PlanetBundle`. Moons are just planet bundles with `BodyType::Moon`.

```rust
// Moons use the same PlanetBundle with body_type = BodyType::Moon.
// No separate bundle needed. The body_type field distinguishes them.
```

### CometBundle

Planet bundle plus the `CometTail` marker component.

```rust
#[derive(Bundle)]
pub struct CometBundle {
    pub planet: PlanetBundle,
    pub comet_tail: CometTail,
}
```

### SunBundle

The Sun is special: it uses an emissive material (no texture, pure glow) and has no `Orbit` component.

```rust
#[derive(Bundle)]
pub struct SunBundle {
    pub body: CelestialBody,
    pub physical_radius: PhysicalRadius,

    // Bevy rendering -- emissive material for bloom glow
    pub mesh: Mesh3d,
    pub material: MeshMaterial3d<StandardMaterial>,
    pub transform: Transform,
    pub global_transform: GlobalTransform,
    pub visibility: Visibility,
    pub inherited_visibility: InheritedVisibility,
    pub view_visibility: ViewVisibility,
}
```

Usage:
```rust
fn spawn_sun(
    commands: &mut Commands,
    meshes: &mut ResMut<Assets<Mesh>>,
    materials: &mut ResMut<Assets<StandardMaterial>>,
) -> Entity {
    commands.spawn(SunBundle {
        body: CelestialBody {
            sim_index: usize::MAX,  // Sun is not in the bodies array
            name: "Sun".into(),
            mass: 1.989e30,
            body_type: BodyType::Star,
        },
        physical_radius: PhysicalRadius(6.9634e8),
        mesh: Mesh3d(meshes.add(Sphere::new(1.0).mesh().uv(32, 18))),
        material: MeshMaterial3d(materials.add(StandardMaterial {
            base_color: Color::srgb(1.0, 0.8, 0.0),
            emissive: LinearRgba::new(10.0, 8.0, 2.0, 1.0),
            unlit: true,
            ..default()
        })),
        transform: Transform::from_xyz(0.0, 0.0, 0.0),
        global_transform: GlobalTransform::default(),
        visibility: Visibility::Visible,
        inherited_visibility: InheritedVisibility::default(),
        view_visibility: ViewVisibility::default(),
    }).id()
}
```

---

## Scale Considerations

The solar system spans ~60 AU (Neptune orbit radius = 4.5e12 meters). Bevy's `Transform` uses `f32`, which has ~7 decimal digits of precision. At 4.5e12 meters, f32 precision is ~500 km -- unacceptable for Moon orbits but fine for visual rendering of planet positions.

**Strategy:**
- `SimulationState.inner` holds all positions in f64 meters (authoritative)
- `Velocity` component stores f64 for physics queries
- `Transform` is updated each frame by scaling: `world_meters / AU * RENDER_SCALE`
- `RENDER_SCALE` is a constant (e.g., 100.0) so 1 AU = 100 Bevy units
- At this scale, Neptune is at ~3007 Bevy units, Earth at ~100, Mercury at ~38.7
- f32 precision at 3007 units is ~0.0003 units = ~450 km, which is subpixel

```rust
/// Conversion factor: 1 AU in Bevy rendering units.
pub const RENDER_SCALE: f64 = 100.0;
pub const AU: f64 = 1.496e11;

/// Convert physics position (meters, f64) to Bevy Transform position (f32).
pub fn physics_to_render(pos: DVec3) -> Vec3 {
    Vec3::new(
        (pos.x / AU * RENDER_SCALE) as f32,
        (pos.y / AU * RENDER_SCALE) as f32,
        (pos.z / AU * RENDER_SCALE) as f32,
    )
}
```

---

## Entity Count Estimate

| Category | Count | Notes |
|----------|-------|-------|
| Sun | 1 | SunBundle |
| Planets | 9 | PlanetBundle (Mercury-Pluto) |
| Moons | 8 | PlanetBundle with Moon type |
| Comets | 4 | CometBundle |
| Named asteroids | 6 | PlanetBundle with Asteroid type |
| Belt particles | 1500 | BeltParticle + Transform + Mesh3d |
| Labels | ~28 | Text2d children of body entities |
| Trail polylines | ~28 | Polyline entities (one per body with trails) |
| Spacetime grid | 1 | Dynamic mesh entity |
| Trajectory overlay | 1 | Polyline entity |
| Vehicle marker | 1 | Small sphere entity |
| Camera | 1 | Camera3d entity |
| Lights | 1 | PointLight at Sun |
| **Total** | **~1589** | Dominated by belt particles |

Bevy handles 1600 entities trivially. The belt particles benefit from automatic instanced rendering since they share the same mesh.
