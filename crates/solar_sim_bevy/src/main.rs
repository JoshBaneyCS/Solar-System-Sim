use bevy::prelude::*;

mod camera_plugin;
mod physics_plugin;
mod render_plugin;

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
            physics_plugin::PhysicsPlugin,
            render_plugin::CelestialRenderPlugin,
            camera_plugin::CameraPlugin,
        ))
        .run();
}
