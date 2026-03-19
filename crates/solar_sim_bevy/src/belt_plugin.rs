use bevy::prelude::*;
use rand::prelude::*;

use crate::physics_plugin::{SimulationConfig, SimulationTime, AU, RENDER_SCALE};

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const BELT_PARTICLE_COUNT: usize = 1500;
const GM_SUN: f64 = 6.674e-11 * 1.989e30;

// ---------------------------------------------------------------------------
// Components
// ---------------------------------------------------------------------------

/// A visual-only asteroid belt particle (not N-body simulated).
#[derive(Component)]
pub struct BeltParticle {
    pub semi_major_axis_au: f64,
    pub eccentricity: f64,
    pub inclination_rad: f64,
    pub initial_anomaly_rad: f64,
}

// ---------------------------------------------------------------------------
// Plugin
// ---------------------------------------------------------------------------

pub struct BeltPlugin;

impl Plugin for BeltPlugin {
    fn build(&self, app: &mut App) {
        app.add_systems(PostStartup, spawn_belt_particles)
            .add_systems(
                Update,
                (update_belt_visibility, update_belt_positions).chain(),
            );
    }
}

// ---------------------------------------------------------------------------
// Startup
// ---------------------------------------------------------------------------

fn spawn_belt_particles(
    mut commands: Commands,
    mut meshes: ResMut<Assets<Mesh>>,
    mut materials: ResMut<Assets<StandardMaterial>>,
) {
    let mut rng = StdRng::seed_from_u64(42);
    let sphere_mesh = meshes.add(Sphere::new(1.0).mesh().uv(6, 4));

    let belt_colors: &[[f32; 3]] = &[
        [0.549, 0.510, 0.451],
        [0.627, 0.569, 0.471],
        [0.471, 0.431, 0.392],
        [0.667, 0.608, 0.510],
        [0.510, 0.471, 0.431],
    ];

    for i in 0..BELT_PARTICLE_COUNT {
        // Main belt: 2.1 to 3.3 AU with Kirkwood gaps
        let mut a = 2.1 + rng.gen::<f64>() * 1.2;
        while (a > 2.48 && a < 2.52) || (a > 2.80 && a < 2.84) || (a > 2.93 && a < 2.97) {
            a = 2.1 + rng.gen::<f64>() * 1.2;
        }

        let e = rng.gen::<f64>() * 0.15;
        let inc = (rng.gen::<f64>() * 2.0 - 1.0) * 20.0_f64.to_radians();
        let anomaly = rng.gen::<f64>() * std::f64::consts::TAU;

        let c = belt_colors[i % belt_colors.len()];
        let size = if i % 17 == 0 {
            0.015
        } else if i % 3 == 0 {
            0.01
        } else {
            0.006
        };

        commands.spawn((
            BeltParticle {
                semi_major_axis_au: a,
                eccentricity: e,
                inclination_rad: inc,
                initial_anomaly_rad: anomaly,
            },
            Mesh3d(sphere_mesh.clone()),
            MeshMaterial3d(materials.add(StandardMaterial {
                base_color: Color::srgba(c[0], c[1], c[2], 0.7),
                unlit: true,
                alpha_mode: AlphaMode::Blend,
                ..default()
            })),
            Transform::from_scale(Vec3::splat(size)),
            Visibility::Hidden,
        ));
    }
}

// ---------------------------------------------------------------------------
// Update systems
// ---------------------------------------------------------------------------

fn update_belt_visibility(
    config: Res<SimulationConfig>,
    mut query: Query<&mut Visibility, With<BeltParticle>>,
) {
    if !config.is_changed() {
        return;
    }
    let vis = if config.show_belt {
        Visibility::Visible
    } else {
        Visibility::Hidden
    };
    for mut v in &mut query {
        *v = vis;
    }
}

fn update_belt_positions(
    config: Res<SimulationConfig>,
    sim_time: Res<SimulationTime>,
    mut query: Query<(&BeltParticle, &mut Transform)>,
) {
    if !config.show_belt {
        return;
    }

    let t = sim_time.elapsed_seconds;

    for (particle, mut transform) in &mut query {
        let pos = belt_particle_position(particle, t);
        // Preserve scale
        let scale = transform.scale;
        transform.translation = pos;
        transform.scale = scale;
    }
}

/// Compute the 3D Bevy world position of a belt particle at the given simulation time.
fn belt_particle_position(p: &BeltParticle, sim_time: f64) -> Vec3 {
    let a = p.semi_major_axis_au * AU;
    let e = p.eccentricity;

    // Mean motion
    let n = (GM_SUN / (a * a * a)).sqrt();

    // Mean anomaly
    let m = (p.initial_anomaly_rad + n * sim_time) % std::f64::consts::TAU;

    // Solve Kepler's equation (Newton's method, 5 iterations)
    let mut big_e = m;
    for _ in 0..5 {
        big_e -= (big_e - e * big_e.sin() - m) / (1.0 - e * big_e.cos());
    }

    // True anomaly
    let nu = 2.0
        * ((1.0 + e).sqrt() * (big_e / 2.0).sin()).atan2((1.0 - e).sqrt() * (big_e / 2.0).cos());

    // Radial distance
    let r = a * (1.0 - e * big_e.cos());

    // Position in orbital plane
    let x_orb = r * nu.cos();
    let y_orb = r * nu.sin();

    // Apply inclination
    let x = x_orb;
    let y = y_orb * p.inclination_rad.cos();
    let z = y_orb * p.inclination_rad.sin();

    // Convert to Bevy world coords (same as physics_to_render)
    Vec3::new(
        (x / AU * RENDER_SCALE) as f32,
        (z / AU * RENDER_SCALE) as f32,
        (y / AU * RENDER_SCALE) as f32,
    )
}
