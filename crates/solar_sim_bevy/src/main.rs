use bevy::prelude::*;
use bevy_embedded_assets::EmbeddedAssetPlugin;

mod belt_plugin;
mod body_catalog;
mod camera_plugin;
mod follow_plugin;
mod label_plugin;
mod launch_core;
mod launch_plugin;
mod physics_plugin;
mod render_plugin;
mod skybox_plugin;
mod spacetime_plugin;
mod trail_plugin;
mod ui_about;
mod ui_bodies;
mod ui_controls;
mod ui_physics;
mod ui_plugin;
mod ui_statusbar;

fn main() {
    App::new()
        .add_plugins(
            DefaultPlugins
                .build()
                .add_before::<bevy::asset::AssetPlugin>(EmbeddedAssetPlugin::default())
                .set(WindowPlugin {
                    primary_window: Some(Window {
                        title: "Solar System Simulator".into(),
                        resolution: (1600., 900.).into(),
                        ..default()
                    }),
                    ..default()
                }),
        )
        .add_plugins((
            physics_plugin::PhysicsPlugin,
            render_plugin::CelestialRenderPlugin,
            camera_plugin::CameraPlugin,
            skybox_plugin::SkyboxPlugin,
            ui_plugin::UIPlugin,
            ui_statusbar::StatusBarDiagnostics,
            trail_plugin::TrailPlugin,
            label_plugin::LabelPlugin,
            belt_plugin::BeltPlugin,
            follow_plugin::FollowPlugin,
            launch_plugin::LaunchPlugin,
            spacetime_plugin::SpacetimePlugin,
        ))
        .run();
}
