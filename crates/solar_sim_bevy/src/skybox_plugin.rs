use bevy::prelude::*;
use bevy::core_pipeline::bloom::Bloom;

use crate::camera_plugin::OrbitCameraMarker;

// ---------------------------------------------------------------------------
// Plugin
// ---------------------------------------------------------------------------

pub struct SkyboxPlugin;

impl Plugin for SkyboxPlugin {
    fn build(&self, app: &mut App) {
        app.add_systems(Startup, setup_skybox_sphere)
            .add_systems(PostStartup, setup_bloom);
    }
}

// ---------------------------------------------------------------------------
// Skybox: large inverted sphere with milky way texture
// ---------------------------------------------------------------------------

/// Marker for the skybox sphere.
#[derive(Component)]
struct SkyboxSphere;

fn setup_skybox_sphere(
    mut commands: Commands,
    mut meshes: ResMut<Assets<Mesh>>,
    mut materials: ResMut<Assets<StandardMaterial>>,
    asset_server: Res<AssetServer>,
) {
    let texture: Handle<Image> = asset_server.load("textures/skybox/milky_way.jpg");

    // Large inverted sphere surrounding the scene
    let mut mesh = Sphere::new(4000.0).mesh().uv(64, 32);
    // Flip normals inward so the texture is visible from inside
    mesh.asset_usage = bevy::render::render_asset::RenderAssetUsages::default();

    let sphere_mesh = meshes.add(mesh);

    commands.spawn((
        Mesh3d(sphere_mesh),
        MeshMaterial3d(materials.add(StandardMaterial {
            base_color_texture: Some(texture),
            unlit: true,
            cull_mode: Some(bevy::render::render_resource::Face::Front),
            ..default()
        })),
        Transform::from_xyz(0.0, 0.0, 0.0),
        SkyboxSphere,
    ));
}

// ---------------------------------------------------------------------------
// Bloom: add bloom settings to the camera for sun glow
// ---------------------------------------------------------------------------

fn setup_bloom(
    mut commands: Commands,
    query: Query<Entity, With<OrbitCameraMarker>>,
) {
    for entity in &query {
        commands.entity(entity).insert((
            Bloom {
                intensity: 0.15,
                low_frequency_boost: 0.6,
                low_frequency_boost_curvature: 0.4,
                high_pass_frequency: 1.5,
                ..default()
            },
            // HDR is required for bloom
            bevy::core_pipeline::tonemapping::Tonemapping::TonyMcMapface,
        ));
    }
}
