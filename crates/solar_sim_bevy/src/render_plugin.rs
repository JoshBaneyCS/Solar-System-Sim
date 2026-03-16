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
            .add_systems(PostStartup, attach_meshes_to_bodies)
            .add_systems(PostUpdate, maintain_display_scale);
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
    query: Query<(Entity, &CelestialBody), Without<RenderInitialized>>,
) {
    let planet_data = physics_plugin::planet_data();
    let sphere_mesh = meshes.add(Sphere::new(1.0).mesh().uv(32, 18));

    for (entity, body) in &query {
        if body.body_type == BodyType::Star {
            // Sun: textured emissive sphere
            let texture: Handle<Image> = asset_server.load("textures/sun/albedo.jpg");
            let scale = 0.5;
            commands.entity(entity).insert((
                Mesh3d(sphere_mesh.clone()),
                MeshMaterial3d(materials.add(StandardMaterial {
                    base_color_texture: Some(texture),
                    emissive: LinearRgba::new(15.0, 12.0, 3.0, 1.0),
                    emissive_exposure_weight: 0.5,
                    unlit: true,
                    ..default()
                })),
                DisplayScale(scale),
                RenderInitialized,
            ));
        } else {
            // Find matching planet def for texture, color, and display radius
            let pd = planet_data.iter().find(|pd| pd.name() == body.name);

            let (material, radius) = if let Some(pd) = pd {
                let tex_name = pd.texture_name();
                // Earth uses png, others use jpg
                let ext = if tex_name == "earth" { "png" } else { "jpg" };
                let texture_path = format!("textures/{}/albedo.{}", tex_name, ext);
                let texture: Handle<Image> = asset_server.load(&texture_path);

                let c = pd.color();
                let mat = StandardMaterial {
                    base_color: Color::srgb(c[0], c[1], c[2]),
                    base_color_texture: Some(texture),
                    perceptual_roughness: 0.8,
                    metallic: 0.0,
                    ..default()
                };
                (mat, pd.display_radius())
            } else {
                let mat = StandardMaterial {
                    base_color: Color::srgb(0.5, 0.5, 0.5),
                    perceptual_roughness: 0.8,
                    metallic: 0.0,
                    ..default()
                };
                (mat, 0.1)
            };

            commands.entity(entity).insert((
                Mesh3d(sphere_mesh.clone()),
                MeshMaterial3d(materials.add(material)),
                DisplayScale(radius),
                RenderInitialized,
            ));

            // Saturn ring
            if body.name == "Saturn" {
                if let Some(pd) = pd {
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
                            Transform::from_scale(Vec3::splat(pd.display_radius())),
                            SaturnRing,
                        ))
                        .id();
                    commands.entity(entity).add_child(ring_entity);
                }
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
