use bevy::prelude::*;
use bevy::diagnostic::{DiagnosticsStore, FrameTimeDiagnosticsPlugin};
use bevy_egui::{egui, EguiContexts};

use crate::camera_plugin::OrbitCamera;
use crate::physics_plugin::{SimulationConfig, SimulationTime};

// ---------------------------------------------------------------------------
// Plugin integration (diagnostics must be registered)
// ---------------------------------------------------------------------------

pub struct StatusBarDiagnostics;

impl Plugin for StatusBarDiagnostics {
    fn build(&self, app: &mut App) {
        app.add_plugins(FrameTimeDiagnosticsPlugin);
    }
}

// ---------------------------------------------------------------------------
// Status bar panel
// ---------------------------------------------------------------------------

pub fn status_bar_panel(
    mut contexts: EguiContexts,
    diagnostics: Res<DiagnosticsStore>,
    sim_time: Res<SimulationTime>,
    config: Res<SimulationConfig>,
    orbit: Res<OrbitCamera>,
) {
    let ctx = contexts.ctx_mut();

    egui::TopBottomPanel::bottom("status_bar")
        .exact_height(28.0)
        .show(ctx, |ui| {
            ui.horizontal_centered(|ui| {
                // FPS
                if let Some(fps) = diagnostics
                    .get(&FrameTimeDiagnosticsPlugin::FPS)
                    .and_then(|d| d.smoothed())
                {
                    ui.label(format!("FPS: {:.0}", fps));
                } else {
                    ui.label("FPS: --");
                }

                ui.separator();

                // Simulation time
                let total_seconds = sim_time.elapsed_seconds;
                let days = total_seconds / 86400.0;
                let years = days / 365.25;
                if years.abs() >= 1.0 {
                    ui.label(format!("Time: {:.2} yr", years));
                } else {
                    ui.label(format!("Time: {:.1} d", days));
                }

                ui.separator();

                // Speed
                ui.label(format!("Speed: {:.2}x", config.time_speed));

                ui.separator();

                // Zoom (distance)
                ui.label(format!("Zoom: {:.1}", orbit.distance));
            });
        });
}
