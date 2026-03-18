use bevy::prelude::*;
use bevy_egui::EguiPlugin;

use crate::camera_plugin::EguiWantsInput;
use crate::physics_plugin::SimulationConfig;
use crate::ui_about;
use crate::ui_bodies;
use crate::ui_controls;
use crate::ui_physics;
use crate::ui_statusbar;

// ---------------------------------------------------------------------------
// Plugin
// ---------------------------------------------------------------------------

pub struct UIPlugin;

impl Plugin for UIPlugin {
    fn build(&self, app: &mut App) {
        app.add_plugins(EguiPlugin)
            .insert_resource(ui_about::AboutWindowOpen::default())
            .add_systems(Update, (
                ui_controls::simulation_controls_panel,
                ui_bodies::bodies_panel
                    .after(ui_controls::simulation_controls_panel),
                ui_physics::physics_panel
                    .after(ui_bodies::bodies_panel),
                ui_statusbar::status_bar_panel
                    .after(ui_physics::physics_panel),
                ui_about::about_dialog
                    .after(ui_statusbar::status_bar_panel),
            ))
            .add_systems(Update, simulation_keyboard_shortcuts);
    }
}

// ---------------------------------------------------------------------------
// Keyboard shortcuts (when egui doesn't want keyboard input)
// ---------------------------------------------------------------------------

fn simulation_keyboard_shortcuts(
    keys: Res<ButtonInput<KeyCode>>,
    mut config: ResMut<SimulationConfig>,
    egui_wants: Res<EguiWantsInput>,
) {
    if egui_wants.keyboard {
        return;
    }

    // Space = toggle play/pause
    if keys.just_pressed(KeyCode::Space) {
        config.is_playing = !config.is_playing;
    }

    // +/= key = increase speed
    if keys.just_pressed(KeyCode::Equal) || keys.just_pressed(KeyCode::NumpadAdd) {
        config.time_speed *= 2.0;
    }

    // - key = decrease speed
    if keys.just_pressed(KeyCode::Minus) || keys.just_pressed(KeyCode::NumpadSubtract) {
        config.time_speed /= 2.0;
    }

    // L = toggle labels
    if keys.just_pressed(KeyCode::KeyL) {
        config.show_labels = !config.show_labels;
    }

    // T = toggle trails
    if keys.just_pressed(KeyCode::KeyT) {
        config.show_trails = !config.show_trails;
    }

    // G = toggle spacetime grid
    if keys.just_pressed(KeyCode::KeyG) {
        config.show_spacetime = !config.show_spacetime;
    }
}
