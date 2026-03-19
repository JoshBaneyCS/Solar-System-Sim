use bevy::prelude::*;
use bevy_egui::{egui, EguiContexts};
use physics_core::constants::G;

use crate::camera_plugin::OrbitCamera;
use crate::physics_plugin::{CelestialBody, SimulationConfig, SimulationState, SimulationTime, AU};

pub fn physics_panel(
    mut contexts: EguiContexts,
    sim_state: Option<Res<SimulationState>>,
    sim_time: Res<SimulationTime>,
    config: Res<SimulationConfig>,
    orbit: Res<OrbitCamera>,
    body_query: Query<&CelestialBody>,
) {
    let ctx = contexts.ctx_mut();

    egui::SidePanel::right("physics_panel")
        .default_width(300.0)
        .resizable(true)
        .show(ctx, |ui| {
            ui.heading("Physics");
            ui.separator();

            egui::ScrollArea::vertical().show(ui, |ui| {
                // --- Equations ---
                ui.strong("Gravitational Force");
                ui.label("F = -GMm/r\u{00b2} \u{00b7} r\u{0302}");
                ui.label(format!(
                    "G = {:.4e} m\u{00b3}/(kg\u{00b7}s\u{00b2})",
                    G
                ));
                ui.label(format!("M\u{2299} = {:.3e} kg", 1.989e30));

                ui.add_space(8.0);
                ui.separator();

                // --- N-body ---
                let n_bodies = sim_state.as_ref().map_or(0, |s| s.inner.n_bodies);
                ui.strong("N-Body Equation");
                ui.label(
                    "a\u{1d62} = \u{2211}(j\u{2260}i) G\u{00b7}m\u{2c7c}/|r\u{2c7c}-r\u{1d62}|\u{00b3} \u{00b7} (r\u{2c7c}-r\u{1d62})"
                );
                ui.label(format!("Bodies: {} (+ Sun)", n_bodies));

                if config.general_relativity {
                    ui.add_space(8.0);
                    ui.separator();
                    ui.strong("General Relativity (1PN)");
                    ui.label("a\u{0047}\u{0052} = (GM/(c\u{00b2}r\u{00b3})) \u{00b7}");
                    ui.label("  [(4GM/r - v\u{00b2})r + 4(r\u{00b7}v)v]");
                    ui.label("Mercury precession: 42.97\"/century");
                }

                ui.add_space(8.0);
                ui.separator();

                // --- Earth values ---
                if let Some(sim) = &sim_state {
                    let earth_idx = body_query
                        .iter()
                        .find(|b| b.name == "Earth")
                        .map(|b| b.sim_index);

                    if let Some(idx) = earth_idx {
                        if idx < sim.inner.n_bodies {
                            let pos = sim.inner.positions[idx];
                            let vel = sim.inner.velocities[idx];

                            let dist_m = (pos.x * pos.x + pos.y * pos.y + pos.z * pos.z).sqrt();
                            let dist_au = dist_m / AU;
                            let speed = (vel.x * vel.x + vel.y * vel.y + vel.z * vel.z).sqrt();

                            let period_days = if speed > 0.0 {
                                2.0 * std::f64::consts::PI * dist_m / speed / 86400.0
                            } else {
                                0.0
                            };

                            let force = G * 1.989e30 * 5.972e24 / (dist_m * dist_m);

                            ui.strong("Earth (Live Values)");
                            ui.label(format!(
                                "Position: ({:.3e}, {:.3e}, {:.3e}) m",
                                pos.x, pos.y, pos.z
                            ));
                            ui.label(format!("Distance: {:.6} AU ({:.3e} m)", dist_au, dist_m));
                            ui.label(format!(
                                "Velocity: {:.2} km/s ({:.2} m/s)",
                                speed / 1000.0,
                                speed
                            ));
                            ui.label(format!("Orbital Period: {:.2} days", period_days));
                            ui.label(format!("Total Force: {:.3e} N", force));
                        }
                    }
                }

                ui.add_space(8.0);
                ui.separator();

                // --- Simulation info ---
                ui.strong("Simulation");
                let days = sim_time.elapsed_seconds / 86400.0;
                let years = days / 365.25;
                ui.label(format!("Time: {:.2} days ({:.4} years)", days, years));
                ui.label(format!("Speed: {:.2}x", config.time_speed));
                ui.label(format!("Zoom: {:.1}", orbit.distance));
                ui.label(format!(
                    "Integrator: {}",
                    match config.integrator {
                        crate::physics_plugin::IntegratorType::RK4 => "RK4",
                        crate::physics_plugin::IntegratorType::Verlet => "Verlet",
                    }
                ));
            });
        });
}
