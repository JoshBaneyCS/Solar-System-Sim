use bevy::prelude::*;

use crate::physics_plugin::{BlackHoleMarker, BodyType, CelestialBody, SimulationConfig};

// ---------------------------------------------------------------------------
// Components
// ---------------------------------------------------------------------------

/// Marker: this entity has been assigned a mesh by the render plugin.
#[derive(Component)]
pub(crate) struct RenderInitialized;

/// Stores the display scale for a celestial body so we can re-apply it
/// after the physics plugin updates the Transform translation.
#[derive(Component)]
pub struct DisplayScale(pub f32);

/// Marker for Saturn's ring entity.
#[derive(Component)]
struct SaturnRing;

// ---------------------------------------------------------------------------
// Plugin
// ---------------------------------------------------------------------------

pub struct CelestialRenderPlugin;

impl Plugin for CelestialRenderPlugin {
    fn build(&self, app: &mut App) {
        app.add_systems(Startup, setup_lighting)
            .add_systems(Update, attach_meshes_to_bodies)
            .add_systems(PostUpdate, (maintain_display_scale, apply_body_spin).chain());
    }
}

// ---------------------------------------------------------------------------
// Startup: lighting
// ---------------------------------------------------------------------------

fn setup_lighting(mut commands: Commands) {
    commands.spawn((
        PointLight {
            intensity: 2_000_000_000.0,
            range: 10000.0,
            shadows_enabled: false,
            ..default()
        },
        Transform::from_xyz(0.0, 0.0, 0.0),
    ));

    commands.insert_resource(AmbientLight {
        color: Color::WHITE,
        brightness: 50.0,
    });
}

// ---------------------------------------------------------------------------
// PostStartup: attach sphere meshes, textures, and materials
// ---------------------------------------------------------------------------

fn attach_meshes_to_bodies(
    mut commands: Commands,
    mut meshes: ResMut<Assets<Mesh>>,
    mut materials: ResMut<Assets<StandardMaterial>>,
    asset_server: Res<AssetServer>,
    query: Query<(Entity, &CelestialBody, Option<&BlackHoleMarker>), Without<RenderInitialized>>,
) {
    let sphere_mesh = meshes.add(Sphere::new(1.0).mesh().uv(32, 18));

    for (entity, body, bh_marker) in &query {
        // Black hole: unlit black sphere, no spin
        if bh_marker.is_some() {
            commands.entity(entity).insert((
                Mesh3d(sphere_mesh.clone()),
                MeshMaterial3d(materials.add(StandardMaterial {
                    base_color: Color::BLACK,
                    emissive: LinearRgba::NONE,
                    unlit: true,
                    ..default()
                })),
                DisplayScale(body.display_radius),
                RenderInitialized,
            ));
        } else if body.body_type == BodyType::Star {
            // Sun: textured emissive sphere
            let texture: Handle<Image> = asset_server.load("textures/sun/albedo.jpg");
            commands.entity(entity).insert((
                Mesh3d(sphere_mesh.clone()),
                MeshMaterial3d(materials.add(StandardMaterial {
                    base_color_texture: Some(texture),
                    emissive: LinearRgba::new(15.0, 12.0, 3.0, 1.0),
                    emissive_exposure_weight: 0.5,
                    unlit: true,
                    ..default()
                })),
                DisplayScale(body.display_radius),
                body_spin_for(&body.name),
                RenderInitialized,
            ));
        } else {
            // Use render info stored on the CelestialBody component
            let c = body.color;
            let material = if !body.texture_name.is_empty() {
                let ext = "jpg";
                let texture_path = format!("textures/{}/albedo.{}", body.texture_name, ext);
                let texture: Handle<Image> = asset_server.load(&texture_path);
                StandardMaterial {
                    base_color: Color::srgb(c[0], c[1], c[2]),
                    base_color_texture: Some(texture),
                    perceptual_roughness: 0.8,
                    metallic: 0.0,
                    ..default()
                }
            } else {
                StandardMaterial {
                    base_color: Color::srgb(c[0], c[1], c[2]),
                    perceptual_roughness: 0.8,
                    metallic: 0.0,
                    ..default()
                }
            };

            let radius = if body.display_radius > 0.0 {
                body.display_radius
            } else {
                0.05
            };

            commands.entity(entity).insert((
                Mesh3d(sphere_mesh.clone()),
                MeshMaterial3d(materials.add(material)),
                DisplayScale(radius),
                body_spin_for(&body.name),
                RenderInitialized,
            ));

            // Saturn ring
            if body.name == "Saturn" {
                let ring_texture: Handle<Image> =
                    asset_server.load("textures/saturn/ring_alpha.png");
                let ring_mesh = meshes.add(create_ring_mesh(1.5, 2.5, 64));
                let ring_entity = commands
                    .spawn((
                        Mesh3d(ring_mesh),
                        MeshMaterial3d(materials.add(StandardMaterial {
                            base_color: Color::srgba(0.85, 0.78, 0.65, 0.7),
                            base_color_texture: Some(ring_texture),
                            alpha_mode: AlphaMode::Blend,
                            unlit: false,
                            double_sided: true,
                            cull_mode: None,
                            ..default()
                        })),
                        Transform::from_scale(Vec3::splat(radius)),
                        SaturnRing,
                    ))
                    .id();
                commands.entity(entity).add_child(ring_entity);
            }
        }
    }
}

// ---------------------------------------------------------------------------
// Create a flat annular ring mesh for Saturn
// ---------------------------------------------------------------------------

fn create_ring_mesh(inner_radius: f32, outer_radius: f32, segments: u32) -> Mesh {
    let mut positions = Vec::new();
    let mut normals = Vec::new();
    let mut uvs = Vec::new();
    let mut indices = Vec::new();

    for i in 0..=segments {
        let angle = (i as f32 / segments as f32) * std::f32::consts::TAU;
        let cos_a = angle.cos();
        let sin_a = angle.sin();
        let u = i as f32 / segments as f32;

        // Inner vertex
        positions.push([inner_radius * cos_a, 0.0, inner_radius * sin_a]);
        normals.push([0.0, 1.0, 0.0]);
        uvs.push([u, 0.0]);

        // Outer vertex
        positions.push([outer_radius * cos_a, 0.0, outer_radius * sin_a]);
        normals.push([0.0, 1.0, 0.0]);
        uvs.push([u, 1.0]);

        if i < segments {
            let base = i * 2;
            indices.push(base);
            indices.push(base + 1);
            indices.push(base + 2);
            indices.push(base + 1);
            indices.push(base + 3);
            indices.push(base + 2);
        }
    }

    Mesh::new(
        bevy::render::mesh::PrimitiveTopology::TriangleList,
        bevy::render::render_asset::RenderAssetUsages::default(),
    )
    .with_inserted_attribute(Mesh::ATTRIBUTE_POSITION, positions)
    .with_inserted_attribute(Mesh::ATTRIBUTE_NORMAL, normals)
    .with_inserted_attribute(Mesh::ATTRIBUTE_UV_0, uvs)
    .with_inserted_indices(bevy::render::mesh::Indices::U32(indices))
}

// ---------------------------------------------------------------------------
// PostUpdate: ensure scale stays correct after physics updates translation
// ---------------------------------------------------------------------------

fn maintain_display_scale(
    mut query: Query<(&DisplayScale, &mut Transform), With<RenderInitialized>>,
) {
    for (scale, mut transform) in &mut query {
        transform.scale = Vec3::splat(scale.0);
    }
}

// ---------------------------------------------------------------------------
// Body spin (axial rotation)
// ---------------------------------------------------------------------------

/// Visual axial rotation for celestial bodies.
#[derive(Component)]
pub struct BodySpin {
    /// Axis of rotation (normalized, in Bevy Y-up coords).
    pub axis: Vec3,
    /// Radians per simulation-second. Negative = retrograde.
    pub radians_per_sim_sec: f64,
}

const SECS_PER_DAY: f64 = 86400.0;
const TAU: f64 = std::f64::consts::TAU;

/// Rotation period in Earth days → radians/sim-second.
fn period_to_rps(days: f64) -> f64 {
    TAU / (days * SECS_PER_DAY)
}

/// Axial tilt in degrees → Bevy axis (tilted from Y-up around Z).
fn tilt_to_axis(tilt_deg: f64) -> Vec3 {
    let tilt = (tilt_deg as f32).to_radians();
    Vec3::new(tilt.sin(), tilt.cos(), 0.0).normalize()
}

/// Lookup spin data for a body by name.
fn body_spin_for(name: &str) -> BodySpin {
    // (rotation_period_days, axial_tilt_degrees)
    // Negative period = retrograde rotation
    let (period_days, tilt_deg) = match name {
        "Sun"       => (25.05, 7.25),
        "Mercury"   => (58.646, 0.034),
        "Venus"     => (-243.025, 177.36),
        "Earth"     => (0.9973, 23.44),
        "Mars"      => (1.026, 25.19),
        "Jupiter"   => (0.4135, 3.13),
        "Saturn"    => (0.4440, 26.73),
        "Uranus"    => (-0.7183, 97.77),
        "Neptune"   => (0.6713, 28.32),
        "Pluto"     => (-6.387, 122.53),
        // Moons (approximate)
        "Moon"      => (27.322, 6.68),
        "Io"        => (1.769, 0.0),
        "Europa"    => (3.551, 0.1),
        "Ganymede"  => (7.155, 0.2),
        "Callisto"  => (16.689, 0.2),
        "Titan"     => (15.945, 0.3),
        "Phobos"    => (0.319, 0.0),
        "Deimos"    => (1.263, 0.0),
        // Default: slow rotation
        _           => (10.0, 0.0),
    };

    BodySpin {
        axis: tilt_to_axis(tilt_deg),
        radians_per_sim_sec: period_to_rps(period_days),
    }
}

fn apply_body_spin(
    config: Res<SimulationConfig>,
    time: Res<Time>,
    mut query: Query<(&BodySpin, &mut Transform), With<RenderInitialized>>,
) {
    if !config.is_playing {
        return;
    }

    // Simulation seconds elapsed this real frame
    let real_dt = time.delta_secs_f64();
    let sim_dt = real_dt * config.time_speed * (config.fixed_dt / (1.0 / 60.0));

    for (spin, mut transform) in &mut query {
        let angle = (spin.radians_per_sim_sec * sim_dt) as f32;
        if angle.abs() > f32::EPSILON {
            transform.rotate_local_axis(Dir3::new(spin.axis).unwrap_or(Dir3::Y), angle);
        }
    }
}
