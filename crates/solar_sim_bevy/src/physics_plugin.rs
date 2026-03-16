use bevy::prelude::*;
use physics_core::constants::G;
use physics_core::sim::Simulation;
use physics_core::vec3::Vec3 as PVec3;

use crate::body_catalog::{ASTEROID_DATA, COMET_DATA, MOON_DATA};
use crate::trail_plugin::TrailBuffer;

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

/// 1 AU in meters.
pub const AU: f64 = 1.496e11;

/// How many Bevy world-units correspond to 1 AU.
pub const RENDER_SCALE: f64 = 10.0;

/// Base simulation timestep in seconds (2 hours, matching Go).
const BASE_TIME_STEP: f64 = 7200.0;

/// Maximum safe per-substep dt before we subdivide (8 hours).
const MAX_SAFE_DT: f64 = 28800.0;

/// Sun mass in kg.
const SUN_MASS: f64 = 1.989e30;

// ---------------------------------------------------------------------------
// Public helpers (used by other plugins)
// ---------------------------------------------------------------------------

/// Convert a physics position (meters, f64) to Bevy world position (f32).
pub fn physics_to_render(pos: PVec3) -> Vec3 {
    Vec3::new(
        (pos.x / AU * RENDER_SCALE) as f32,
        (pos.z / AU * RENDER_SCALE) as f32, // z -> up in Bevy
        (pos.y / AU * RENDER_SCALE) as f32, // y -> forward
    )
}

// ---------------------------------------------------------------------------
// Components
// ---------------------------------------------------------------------------

/// Links a Bevy entity to its index inside the physics `Simulation`.
#[derive(Component)]
pub struct CelestialBody {
    pub sim_index: usize,
    pub name: String,
    pub body_type: BodyType,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum BodyType {
    Star,
    Planet,
    DwarfPlanet,
    Moon,
    Comet,
    Asteroid,
}

/// Marker for the Sun entity.
#[derive(Component)]
pub struct Sun;

// ---------------------------------------------------------------------------
// Resources
// ---------------------------------------------------------------------------

#[derive(Resource)]
pub struct SimulationConfig {
    pub time_speed: f64,
    pub is_playing: bool,
    pub fixed_dt: f64,
    // Toggles
    pub show_trails: bool,
    pub show_labels: bool,
    pub show_moons: bool,
    pub show_comets: bool,
    pub show_asteroids: bool,
    pub show_belt: bool,
    pub show_spacetime: bool,
    pub planet_gravity: bool,
    pub general_relativity: bool,
    pub sun_mass_multiplier: f64,
    pub integrator: IntegratorType,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum IntegratorType {
    RK4,
    Verlet,
}

impl Default for SimulationConfig {
    fn default() -> Self {
        Self {
            time_speed: 1.0,
            is_playing: true,
            fixed_dt: BASE_TIME_STEP,
            show_trails: true,
            show_labels: true,
            show_moons: false,
            show_comets: false,
            show_asteroids: false,
            show_belt: false,
            show_spacetime: false,
            planet_gravity: true,
            general_relativity: true,
            sun_mass_multiplier: 1.0,
            integrator: IntegratorType::Verlet,
        }
    }
}

#[derive(Resource)]
pub struct SimulationState {
    pub inner: Simulation,
}

/// Tracks elapsed simulation time in seconds.
#[derive(Resource, Default)]
pub struct SimulationTime {
    pub elapsed_seconds: f64,
}

// ---------------------------------------------------------------------------
// Planet catalog — ported from internal/physics/planets.go
// ---------------------------------------------------------------------------

pub(crate) struct PlanetDef {
    name: &'static str,
    mass: f64,
    semi_major_axis_au: f64,
    eccentricity: f64,
    inclination_deg: f64,
    long_ascending_node_deg: f64,
    arg_perihelion_deg: f64,
    initial_anomaly_rad: f64,
    body_type: BodyType,
    color: [f32; 3],
    display_radius: f32,
    #[allow(dead_code)]
    physical_radius: f64,
    /// Texture directory name under assets/textures/
    pub texture_name: &'static str,
}

const PI: f64 = std::f64::consts::PI;

static PLANET_DATA: &[PlanetDef] = &[
    PlanetDef {
        name: "Mercury",
        mass: 3.3011e23,
        semi_major_axis_au: 0.387,
        eccentricity: 0.2056,
        inclination_deg: 7.005,
        long_ascending_node_deg: 48.331,
        arg_perihelion_deg: 29.124,
        initial_anomaly_rad: 0.0,
        body_type: BodyType::Planet,
        color: [0.663, 0.663, 0.663],
        display_radius: 0.06,
        physical_radius: 2.4397e6,
        texture_name: "mercury",
    },
    PlanetDef {
        name: "Venus",
        mass: 4.8675e24,
        semi_major_axis_au: 0.723,
        eccentricity: 0.0068,
        inclination_deg: 3.395,
        long_ascending_node_deg: 76.680,
        arg_perihelion_deg: 54.884,
        initial_anomaly_rad: PI / 4.0,
        body_type: BodyType::Planet,
        color: [1.0, 0.776, 0.286],
        display_radius: 0.08,
        physical_radius: 6.0518e6,
        texture_name: "venus",
    },
    PlanetDef {
        name: "Earth",
        mass: 5.972e24,
        semi_major_axis_au: 1.0,
        eccentricity: 0.0167,
        inclination_deg: 0.0,
        long_ascending_node_deg: 0.0,
        arg_perihelion_deg: 102.937,
        initial_anomaly_rad: PI / 2.0,
        body_type: BodyType::Planet,
        color: [0.392, 0.584, 0.929],
        display_radius: 0.08,
        physical_radius: 6.371e6,
        texture_name: "earth",
    },
    PlanetDef {
        name: "Mars",
        mass: 6.4171e23,
        semi_major_axis_au: 1.524,
        eccentricity: 0.0934,
        inclination_deg: 1.850,
        long_ascending_node_deg: 49.558,
        arg_perihelion_deg: 286.502,
        initial_anomaly_rad: 3.0 * PI / 4.0,
        body_type: BodyType::Planet,
        color: [0.757, 0.267, 0.055],
        display_radius: 0.07,
        physical_radius: 3.3895e6,
        texture_name: "mars",
    },
    PlanetDef {
        name: "Jupiter",
        mass: 1.8982e27,
        semi_major_axis_au: 5.203,
        eccentricity: 0.0489,
        inclination_deg: 1.303,
        long_ascending_node_deg: 100.464,
        arg_perihelion_deg: 273.867,
        initial_anomaly_rad: PI,
        body_type: BodyType::Planet,
        color: [0.847, 0.792, 0.616],
        display_radius: 0.20,
        physical_radius: 6.9911e7,
        texture_name: "jupiter",
    },
    PlanetDef {
        name: "Saturn",
        mass: 5.6834e26,
        semi_major_axis_au: 9.537,
        eccentricity: 0.0565,
        inclination_deg: 2.485,
        long_ascending_node_deg: 113.665,
        arg_perihelion_deg: 339.392,
        initial_anomaly_rad: 5.0 * PI / 4.0,
        body_type: BodyType::Planet,
        color: [0.980, 0.871, 0.643],
        display_radius: 0.18,
        physical_radius: 5.8232e7,
        texture_name: "saturn",
    },
    PlanetDef {
        name: "Uranus",
        mass: 8.6810e25,
        semi_major_axis_au: 19.191,
        eccentricity: 0.0457,
        inclination_deg: 0.773,
        long_ascending_node_deg: 74.006,
        arg_perihelion_deg: 96.998,
        initial_anomaly_rad: 3.0 * PI / 2.0,
        body_type: BodyType::Planet,
        color: [0.310, 0.816, 0.906],
        display_radius: 0.14,
        physical_radius: 2.5362e7,
        texture_name: "uranus",
    },
    PlanetDef {
        name: "Neptune",
        mass: 1.02413e26,
        semi_major_axis_au: 30.07,
        eccentricity: 0.0113,
        inclination_deg: 1.770,
        long_ascending_node_deg: 131.784,
        arg_perihelion_deg: 276.336,
        initial_anomaly_rad: 7.0 * PI / 4.0,
        body_type: BodyType::Planet,
        color: [0.247, 0.329, 0.729],
        display_radius: 0.14,
        physical_radius: 2.4622e7,
        texture_name: "neptune",
    },
    PlanetDef {
        name: "Pluto",
        mass: 1.303e22,
        semi_major_axis_au: 39.482,
        eccentricity: 0.2488,
        inclination_deg: 17.16,
        long_ascending_node_deg: 110.299,
        arg_perihelion_deg: 113.834,
        initial_anomaly_rad: 2.0 * PI,
        body_type: BodyType::DwarfPlanet,
        color: [0.824, 0.745, 0.667],
        display_radius: 0.05,
        physical_radius: 1.1883e6,
        texture_name: "pluto",
    },
];

// ---------------------------------------------------------------------------
// Orbital element -> Cartesian conversion (port of Go CreatePlanetFromElements)
// ---------------------------------------------------------------------------

pub(crate) fn create_planet_from_elements(p: &PlanetDef, sun_mass: f64) -> (PVec3, PVec3) {
    let a = p.semi_major_axis_au * AU;
    let e = p.eccentricity;
    let i = p.inclination_deg.to_radians();
    let big_omega = p.long_ascending_node_deg.to_radians();
    let omega = p.arg_perihelion_deg.to_radians();
    let nu = p.initial_anomaly_rad;

    // Radius from the focus
    let r = a * (1.0 - e * e) / (1.0 + e * nu.cos());

    // Position in the orbital plane
    let x_orb = r * nu.cos();
    let y_orb = r * nu.sin();

    // Rotate by argument of perihelion
    let x1 = x_orb * omega.cos() - y_orb * omega.sin();
    let y1 = x_orb * omega.sin() + y_orb * omega.cos();
    let z1 = 0.0_f64;

    // Rotate by inclination
    let x2 = x1;
    let y2 = y1 * i.cos() - z1 * i.sin();
    let z2 = y1 * i.sin() + z1 * i.cos();

    // Rotate by longitude of ascending node
    let x = x2 * big_omega.cos() - y2 * big_omega.sin();
    let y = x2 * big_omega.sin() + y2 * big_omega.cos();
    let z = z2;

    // Velocity in the orbital plane (using mu/h formulation)
    let gm = G * sun_mass;
    let h = (gm * a * (1.0 - e * e)).sqrt();
    let mu_over_h = gm / h;

    let vx_orb = -mu_over_h * nu.sin();
    let vy_orb = mu_over_h * (e + nu.cos());

    // Rotate velocity through same three rotations
    let vx1 = vx_orb * omega.cos() - vy_orb * omega.sin();
    let vy1 = vx_orb * omega.sin() + vy_orb * omega.cos();
    let vz1 = 0.0_f64;

    let vx2 = vx1;
    let vy2 = vy1 * i.cos() - vz1 * i.sin();
    let vz2 = vy1 * i.sin() + vz1 * i.cos();

    let vx = vx2 * big_omega.cos() - vy2 * big_omega.sin();
    let vy = vx2 * big_omega.sin() + vy2 * big_omega.cos();
    let vz = vz2;

    (PVec3::new(x, y, z), PVec3::new(vx, vy, vz))
}

// ---------------------------------------------------------------------------
// Plugin
// ---------------------------------------------------------------------------

pub struct PhysicsPlugin;

impl Plugin for PhysicsPlugin {
    fn build(&self, app: &mut App) {
        app.insert_resource(SimulationConfig::default())
            .insert_resource(SimulationTime::default())
            .insert_resource(Time::<Fixed>::from_hz(60.0))
            .add_systems(Startup, spawn_solar_system)
            .add_systems(FixedUpdate, (step_simulation, sync_ecs_from_simulation).chain())
            .add_systems(Update, manage_dynamic_bodies);
    }
}

// ---------------------------------------------------------------------------
// Startup system: create Sun + 9 planets
// ---------------------------------------------------------------------------

fn spawn_solar_system(mut commands: Commands) {
    let mut masses = Vec::with_capacity(PLANET_DATA.len());
    let mut positions = Vec::with_capacity(PLANET_DATA.len());
    let mut velocities = Vec::with_capacity(PLANET_DATA.len());
    let mut gr_flags = Vec::with_capacity(PLANET_DATA.len());

    for p in PLANET_DATA.iter() {
        let (pos, vel) = create_planet_from_elements(p, SUN_MASS);
        masses.push(p.mass);
        positions.push(pos);
        velocities.push(vel);
        gr_flags.push(p.name == "Mercury");
    }

    let sim = Simulation::new(
        PLANET_DATA.len(),
        SUN_MASS,
        masses,
        positions.clone(),
        velocities,
        gr_flags,
        true,
    );

    // Spawn Sun entity
    commands.spawn((
        Sun,
        CelestialBody {
            sim_index: usize::MAX,
            name: "Sun".into(),
            body_type: BodyType::Star,
        },
        Transform::from_xyz(0.0, 0.0, 0.0),
    ));

    // Spawn planet entities with trail buffers
    for (idx, p) in PLANET_DATA.iter().enumerate() {
        let render_pos = physics_to_render(positions[idx]);
        commands.spawn((
            CelestialBody {
                sim_index: idx,
                name: p.name.into(),
                body_type: p.body_type,
            },
            Transform::from_translation(render_pos),
            TrailBuffer::new(2000),
        ));
    }

    commands.insert_resource(SimulationState { inner: sim });
    commands.insert_resource(DynamicBodyState::default());
}

// ---------------------------------------------------------------------------
// Fixed-update systems
// ---------------------------------------------------------------------------

fn step_simulation(
    config: Res<SimulationConfig>,
    sim_state: Option<ResMut<SimulationState>>,
    mut sim_time: ResMut<SimulationTime>,
) {
    let Some(mut sim) = sim_state else { return };
    if !config.is_playing {
        return;
    }

    let effective_dt = config.fixed_dt * config.time_speed;
    let abs_dt = effective_dt.abs();

    if abs_dt <= MAX_SAFE_DT {
        sim.inner.step(effective_dt);
    } else {
        let n_sub = (abs_dt / MAX_SAFE_DT).ceil() as usize;
        let sub_dt = effective_dt / n_sub as f64;
        for _ in 0..n_sub {
            sim.inner.step(sub_dt);
        }
    }

    sim_time.elapsed_seconds += effective_dt;
}

fn sync_ecs_from_simulation(
    sim_state: Option<Res<SimulationState>>,
    mut query: Query<(&CelestialBody, &mut Transform), Without<Sun>>,
) {
    let Some(sim) = sim_state else { return };

    for (body, mut transform) in &mut query {
        if body.sim_index >= sim.inner.n_bodies {
            continue;
        }
        let pos = sim.inner.positions[body.sim_index];
        transform.translation = physics_to_render(pos);
    }
}

// ---------------------------------------------------------------------------
// Dynamic body spawning (moons, comets, asteroids)
// ---------------------------------------------------------------------------

/// Tracks which dynamic body groups are currently spawned.
#[derive(Resource, Default)]
struct DynamicBodyState {
    moons_spawned: bool,
    comets_spawned: bool,
    asteroids_spawned: bool,
}

/// Marker for dynamically spawned bodies so we can despawn them.
#[derive(Component)]
pub struct DynamicBody;

fn manage_dynamic_bodies(
    mut commands: Commands,
    config: Res<SimulationConfig>,
    mut state: ResMut<DynamicBodyState>,
    sim_state: Option<ResMut<SimulationState>>,
    query: Query<(Entity, &CelestialBody), With<DynamicBody>>,
    planet_query: Query<(&CelestialBody, &Transform), Without<DynamicBody>>,
) {
    if !config.is_changed() {
        return;
    }

    let Some(mut sim) = sim_state else { return };

    // --- Moons ---
    if config.show_moons && !state.moons_spawned {
        for moon_def in MOON_DATA.iter() {
            // Find parent body position/velocity in the simulation
            let parent_state = planet_query.iter().find(|(b, _)| b.name == moon_def.parent_name);
            let (parent_pos, parent_vel) = if let Some((body, _)) = parent_state {
                if body.sim_index < sim.inner.n_bodies {
                    (sim.inner.positions[body.sim_index], sim.inner.velocities[body.sim_index])
                } else {
                    (PVec3::default(), PVec3::default())
                }
            } else if moon_def.parent_name == "Sun" {
                (PVec3::default(), PVec3::default())
            } else {
                continue;
            };

            let (rel_pos, rel_vel) = create_moon_from_elements(moon_def);
            let abs_pos = PVec3::new(
                parent_pos.x + rel_pos.x,
                parent_pos.y + rel_pos.y,
                parent_pos.z + rel_pos.z,
            );
            let abs_vel = PVec3::new(
                parent_vel.x + rel_vel.x,
                parent_vel.y + rel_vel.y,
                parent_vel.z + rel_vel.z,
            );

            // Add to physics simulation
            let idx = sim.inner.n_bodies;
            sim.inner.masses.push(moon_def.mass);
            sim.inner.positions.push(abs_pos);
            sim.inner.velocities.push(abs_vel);
            sim.inner.gr_flags.push(false);
            sim.inner.n_bodies += 1;

            let render_pos = physics_to_render(abs_pos);
            commands.spawn((
                CelestialBody {
                    sim_index: idx,
                    name: moon_def.name.into(),
                    body_type: BodyType::Moon,
                },
                Transform::from_translation(render_pos),
                TrailBuffer::new(2000),
                DynamicBody,
            ));
        }
        state.moons_spawned = true;
    } else if !config.show_moons && state.moons_spawned {
        remove_bodies_by_type(&mut commands, &query, &mut sim.inner, BodyType::Moon);
        state.moons_spawned = false;
    }

    // --- Comets ---
    if config.show_comets && !state.comets_spawned {
        for comet_def in COMET_DATA.iter() {
            let pdef = comet_to_planet_def(comet_def);
            let (pos, vel) = create_planet_from_elements(&pdef, SUN_MASS);
            let idx = sim.inner.n_bodies;
            sim.inner.masses.push(comet_def.mass);
            sim.inner.positions.push(pos);
            sim.inner.velocities.push(vel);
            sim.inner.gr_flags.push(false);
            sim.inner.n_bodies += 1;

            commands.spawn((
                CelestialBody {
                    sim_index: idx,
                    name: comet_def.name.into(),
                    body_type: BodyType::Comet,
                },
                Transform::from_translation(physics_to_render(pos)),
                TrailBuffer::new(2000),
                DynamicBody,
            ));
        }
        state.comets_spawned = true;
    } else if !config.show_comets && state.comets_spawned {
        remove_bodies_by_type(&mut commands, &query, &mut sim.inner, BodyType::Comet);
        state.comets_spawned = false;
    }

    // --- Asteroids ---
    if config.show_asteroids && !state.asteroids_spawned {
        for asteroid_def in ASTEROID_DATA.iter() {
            let pdef = asteroid_to_planet_def(asteroid_def);
            let (pos, vel) = create_planet_from_elements(&pdef, SUN_MASS);
            let idx = sim.inner.n_bodies;
            sim.inner.masses.push(asteroid_def.mass);
            sim.inner.positions.push(pos);
            sim.inner.velocities.push(vel);
            sim.inner.gr_flags.push(false);
            sim.inner.n_bodies += 1;

            commands.spawn((
                CelestialBody {
                    sim_index: idx,
                    name: asteroid_def.name.into(),
                    body_type: asteroid_def.body_type,
                },
                Transform::from_translation(physics_to_render(pos)),
                TrailBuffer::new(2000),
                DynamicBody,
            ));
        }
        state.asteroids_spawned = true;
    } else if !config.show_asteroids && state.asteroids_spawned {
        // Remove both Asteroid and DwarfPlanet types that were dynamically added
        remove_bodies_by_type(&mut commands, &query, &mut sim.inner, BodyType::Asteroid);
        remove_bodies_by_type(&mut commands, &query, &mut sim.inner, BodyType::DwarfPlanet);
        state.asteroids_spawned = false;
    }
}

fn remove_bodies_by_type(
    commands: &mut Commands,
    query: &Query<(Entity, &CelestialBody), With<DynamicBody>>,
    sim: &mut Simulation,
    body_type: BodyType,
) {
    // Collect indices to remove (must remove from highest to lowest to preserve indices)
    let mut to_remove: Vec<(Entity, usize)> = query
        .iter()
        .filter(|(_, b)| b.body_type == body_type)
        .map(|(e, b)| (e, b.sim_index))
        .collect();
    to_remove.sort_by(|a, b| b.1.cmp(&a.1)); // Descending by index

    for (entity, idx) in to_remove {
        commands.entity(entity).despawn_recursive();
        if idx < sim.n_bodies {
            sim.masses.remove(idx);
            sim.positions.remove(idx);
            sim.velocities.remove(idx);
            sim.gr_flags.remove(idx);
            sim.n_bodies -= 1;
        }
    }
}

/// Create moon position/velocity relative to parent.
fn create_moon_from_elements(
    m: &crate::body_catalog::MoonDef,
) -> (PVec3, PVec3) {
    let a = m.semi_major_axis_au * AU;
    let e = m.eccentricity;
    let i = m.inclination_deg.to_radians();
    let big_omega = m.long_ascending_node_deg.to_radians();
    let omega = m.arg_perihelion_deg.to_radians();
    let nu = m.initial_anomaly_rad;

    let r = a * (1.0 - e * e) / (1.0 + e * nu.cos());
    let x_orb = r * nu.cos();
    let y_orb = r * nu.sin();

    let x1 = x_orb * omega.cos() - y_orb * omega.sin();
    let y1 = x_orb * omega.sin() + y_orb * omega.cos();

    let x2 = x1;
    let y2 = y1 * i.cos();
    let z2 = y1 * i.sin();

    let x = x2 * big_omega.cos() - y2 * big_omega.sin();
    let y = x2 * big_omega.sin() + y2 * big_omega.cos();
    let z = z2;

    let gm = G * m.parent_mass;
    let h = (gm * a * (1.0 - e * e)).sqrt();
    let mu_over_h = gm / h;

    let vx_orb = -mu_over_h * nu.sin();
    let vy_orb = mu_over_h * (e + nu.cos());

    let vx1 = vx_orb * omega.cos() - vy_orb * omega.sin();
    let vy1 = vx_orb * omega.sin() + vy_orb * omega.cos();

    let vx2 = vx1;
    let vy2 = vy1 * i.cos();
    let vz2 = vy1 * i.sin();

    let vx = vx2 * big_omega.cos() - vy2 * big_omega.sin();
    let vy = vx2 * big_omega.sin() + vy2 * big_omega.cos();
    let vz = vz2;

    (PVec3::new(x, y, z), PVec3::new(vx, vy, vz))
}

/// Convert CometDef to temporary PlanetDef for orbital element conversion.
fn comet_to_planet_def(c: &crate::body_catalog::CometDef) -> PlanetDef {
    PlanetDef {
        name: c.name,
        mass: c.mass,
        semi_major_axis_au: c.semi_major_axis_au,
        eccentricity: c.eccentricity,
        inclination_deg: c.inclination_deg,
        long_ascending_node_deg: c.long_ascending_node_deg,
        arg_perihelion_deg: c.arg_perihelion_deg,
        initial_anomaly_rad: c.initial_anomaly_rad,
        body_type: BodyType::Comet,
        color: c.color,
        display_radius: c.display_radius,
        physical_radius: c.physical_radius,
        texture_name: "",
    }
}

/// Convert AsteroidDef to temporary PlanetDef for orbital element conversion.
fn asteroid_to_planet_def(a: &crate::body_catalog::AsteroidDef) -> PlanetDef {
    PlanetDef {
        name: a.name,
        mass: a.mass,
        semi_major_axis_au: a.semi_major_axis_au,
        eccentricity: a.eccentricity,
        inclination_deg: a.inclination_deg,
        long_ascending_node_deg: a.long_ascending_node_deg,
        arg_perihelion_deg: a.arg_perihelion_deg,
        initial_anomaly_rad: a.initial_anomaly_rad,
        body_type: a.body_type,
        color: a.color,
        display_radius: a.display_radius,
        physical_radius: a.physical_radius,
        texture_name: a.texture_name,
    }
}

// ---------------------------------------------------------------------------
// Public re-exports for other plugins
// ---------------------------------------------------------------------------

pub fn planet_data() -> &'static [PlanetDef] {
    PLANET_DATA
}

impl PlanetDef {
    pub fn color(&self) -> [f32; 3] {
        self.color
    }
    pub fn display_radius(&self) -> f32 {
        self.display_radius
    }
    pub fn name(&self) -> &str {
        self.name
    }
    pub fn texture_name(&self) -> &str {
        self.texture_name
    }
}
