use bevy::prelude::*;
use bevy_egui::{egui, EguiContexts};

use crate::camera_plugin::OrbitCameraMarker;
use crate::physics_plugin::{CelestialBody, SimulationConfig};

// ---------------------------------------------------------------------------
// Plugin
// ---------------------------------------------------------------------------

pub struct LabelPlugin;

impl Plugin for LabelPlugin {
    fn build(&self, app: &mut App) {
        app.add_systems(Update, draw_labels);
    }
}

// ---------------------------------------------------------------------------
// System
// ---------------------------------------------------------------------------

/// Project body positions to screen and draw egui labels.
fn draw_labels(
    mut contexts: EguiContexts,
    config: Res<SimulationConfig>,
    camera_query: Query<(&Camera, &GlobalTransform), With<OrbitCameraMarker>>,
    body_query: Query<(&CelestialBody, &GlobalTransform)>,
) {
    if !config.show_labels {
        return;
    }

    let Ok((camera, cam_transform)) = camera_query.get_single() else {
        return;
    };

    let ctx = contexts.ctx_mut();

    // Use a transparent area overlay for the labels
    egui::Area::new(egui::Id::new("body_labels"))
        .order(egui::Order::Background)
        .interactable(false)
        .show(ctx, |ui| {
            let screen_rect = ui.ctx().screen_rect();
            ui.set_clip_rect(screen_rect);

            for (body, body_transform) in &body_query {
                let world_pos = body_transform.translation();
                if let Some(ndc) = camera.world_to_ndc(cam_transform, world_pos) {
                    // NDC is -1..1, convert to screen coords
                    let screen_x = (ndc.x + 1.0) * 0.5 * screen_rect.width();
                    let screen_y = (1.0 - ndc.y) * 0.5 * screen_rect.height();

                    // Only draw if in front of camera
                    if ndc.z > 0.0 && ndc.z < 1.0 {
                        let label_pos = egui::pos2(screen_x + 10.0, screen_y - 10.0);
                        ui.painter().text(
                            label_pos,
                            egui::Align2::LEFT_BOTTOM,
                            &body.name,
                            egui::FontId::proportional(12.0),
                            egui::Color32::from_rgba_unmultiplied(220, 220, 230, 200),
                        );
                    }
                }
            }
        });
}
