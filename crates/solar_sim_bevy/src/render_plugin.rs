use bevy::prelude::*;

use crate::physics_plugin::{self, BodyType, CelestialBody};

// ---------------------------------------------------------------------------
// Components
// ---------------------------------------------------------------------------

/// Marker: this entity has been assigned a mesh by the render plugin.
#[derive(Component)]
struct RenderInitialized;

/// Stores the display scale for a celestial body so we can re-apply it
/// after the physics plugin updates the Transform translation.
#[derive(Component)]
pub struct DisplayScale(pub f32);

// ---------------------------------------------------------------------------
// Plugin
// ---------------------------------------------------------------------------

pub struct CelestialRenderPlugin;

impl Plugin for CelestialRenderPlugin {
    fn build(&self, app: &mut App) {
        app.add_systems(Startup, setup_lighting)
            .add_systems(PostStartup, attach_meshes_to_bodies)
            .add_systems(PostUpdate, maintain_display_scale);
    }
}

// ---------------------------------------------------------------------------
// Startup: lighting
// ---------------------------------------------------------------------------

fn setup_lighting(mut commands: Commands) {
    // Point light at the Sun's position
    commands.spawn((
        PointLight {
            intensity: 2_000_000_000.0,
            range: 10000.0,
            shadows_enabled: false,
            ..default()
        },
        Transform::from_xyz(0.0, 0.0, 0.0),
    ));

    // Dim ambient light so planets facing away from the sun are darker
    commands.insert_resource(AmbientLight {
        color: Color::WHITE,
        brightness: 50.0,
    });
}

// ---------------------------------------------------------------------------
// PostStartup: attach sphere meshes and materials to CelestialBody entities
// ---------------------------------------------------------------------------

fn attach_meshes_to_bodies(
    mut commands: Commands,
    mut meshes: ResMut<Assets<Mesh>>,
    mut materials: ResMut<Assets<StandardMaterial>>,
    query: Query<(Entity, &CelestialBody), Without<RenderInitialized>>,
) {
    let planet_data = physics_plugin::planet_data();

    // Shared sphere mesh (unit radius, subdivided)
    let sphere_mesh = meshes.add(Sphere::new(1.0).mesh().uv(32, 18));

    for (entity, body) in &query {
        if body.body_type == BodyType::Star {
            // Sun: emissive unlit sphere
            let scale = 0.5;
            commands.entity(entity).insert((
                Mesh3d(sphere_mesh.clone()),
                MeshMaterial3d(materials.add(StandardMaterial {
                    base_color: Color::srgb(1.0, 0.9, 0.3),
                    emissive: LinearRgba::new(15.0, 12.0, 3.0, 1.0),
                    unlit: true,
                    ..default()
                })),
                DisplayScale(scale),
                RenderInitialized,
            ));
        } else {
            // Find matching planet def for color and display radius
            let (color, radius) = planet_data
                .iter()
                .find(|pd| pd.name() == body.name)
                .map(|pd| {
                    let c = pd.color();
                    (Color::srgb(c[0], c[1], c[2]), pd.display_radius())
                })
                .unwrap_or((Color::srgb(0.5, 0.5, 0.5), 0.1));

            commands.entity(entity).insert((
                Mesh3d(sphere_mesh.clone()),
                MeshMaterial3d(materials.add(StandardMaterial {
                    base_color: color,
                    perceptual_roughness: 0.8,
                    metallic: 0.0,
                    ..default()
                })),
                DisplayScale(radius),
                RenderInitialized,
            ));
        }
    }
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
