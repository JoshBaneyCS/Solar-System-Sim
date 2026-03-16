use bevy::prelude::*;
use bevy_egui::{egui, EguiContexts};

use crate::camera_plugin::OrbitCamera;
use crate::follow_plugin::compute_auto_fit_distance;
use crate::physics_plugin::{CelestialBody, IntegratorType, SimulationConfig};
use crate::ui_about::AboutWindowOpen;
use crate::ui_bodies::FollowTarget;

pub fn simulation_controls_panel(
    mut contexts: EguiContexts,
    mut config: ResMut<SimulationConfig>,
    mut speed_exp: Local<Option<f64>>,
    mut orbit: ResMut<OrbitCamera>,
    mut follow_target: ResMut<FollowTarget>,
    mut about_open: ResMut<AboutWindowOpen>,
    body_query: Query<&Transform, With<CelestialBody>>,
    named_query: Query<(Entity, &CelestialBody)>,
) {
    // Initialize speed exponent from config on first frame
    let exp = speed_exp.get_or_insert_with(|| config.time_speed.abs().max(0.001).log2());

    let ctx = contexts.ctx_mut();

    egui::SidePanel::left("simulation_controls")
        .default_width(280.0)
        .resizable(true)
        .show(ctx, |ui| {
            ui.heading("Simulation Controls");
            ui.separator();

            // Play / Pause / Rewind / Fast-Forward
            ui.horizontal(|ui| {
                if ui.button(if config.is_playing { "\u{23F8} Pause" } else { "\u{25B6} Play" }).clicked() {
                    config.is_playing = !config.is_playing;
                }
                if ui.button("\u{23EA} Rewind").clicked() {
                    config.time_speed = -config.time_speed.abs();
                    config.is_playing = true;
                }
                if ui.button("\u{23E9} FF").clicked() {
                    config.time_speed = config.time_speed.abs();
                    config.is_playing = true;
                }
            });

            ui.add_space(8.0);

            // Speed slider (exponential: 2^exp)
            let mut exp_val = *exp;
            ui.horizontal(|ui| {
                ui.label("Speed:");
                ui.label(format!("{:.2}x", config.time_speed));
            });
            if ui.add(egui::Slider::new(&mut exp_val, -10.0..=10.0).text("2^x")).changed() {
                *exp = exp_val;
                let sign = if config.time_speed < 0.0 { -1.0 } else { 1.0 };
                config.time_speed = sign * 2.0_f64.powf(exp_val);
            }

            ui.add_space(12.0);
            ui.separator();
            ui.heading("Display");
            ui.add_space(4.0);

            ui.checkbox(&mut config.show_trails, "Show Orbital Trails");
            ui.checkbox(&mut config.show_labels, "Show Labels");
            ui.checkbox(&mut config.show_belt, "Show Asteroid Belt");
            ui.checkbox(&mut config.show_spacetime, "Show Spacetime Fabric");

            ui.add_space(12.0);
            ui.separator();
            ui.heading("Bodies");
            ui.add_space(4.0);

            ui.checkbox(&mut config.show_moons, "Show Moons");
            ui.checkbox(&mut config.show_comets, "Show Comets");
            ui.checkbox(&mut config.show_asteroids, "Show Asteroids");

            ui.add_space(12.0);
            ui.separator();
            ui.heading("Physics");
            ui.add_space(4.0);

            ui.checkbox(&mut config.planet_gravity, "Planet-Planet Gravity");
            ui.checkbox(&mut config.general_relativity, "General Relativity");

            // Integrator selection
            ui.horizontal(|ui| {
                ui.label("Integrator:");
                egui::ComboBox::from_id_salt("integrator_select")
                    .selected_text(match config.integrator {
                        IntegratorType::Verlet => "Verlet (symplectic)",
                        IntegratorType::RK4 => "RK4 (classic)",
                    })
                    .show_ui(ui, |ui| {
                        ui.selectable_value(&mut config.integrator, IntegratorType::Verlet, "Verlet (symplectic)");
                        ui.selectable_value(&mut config.integrator, IntegratorType::RK4, "RK4 (classic)");
                    });
            });

            ui.add_space(8.0);

            // Sun mass slider
            let mut sun_mass = config.sun_mass_multiplier as f32;
            ui.horizontal(|ui| {
                ui.label(format!("Sun Mass: {:.1}x", sun_mass));
            });
            if ui.add(egui::Slider::new(&mut sun_mass, 0.1..=5.0)).changed() {
                config.sun_mass_multiplier = sun_mass as f64;
            }

            ui.add_space(12.0);
            ui.separator();

            if ui.button("\u{21BA} Reset Simulation").clicked() {
                // TODO: send reset event
            }

            ui.add_space(12.0);
            ui.separator();
            ui.heading("Camera");
            ui.add_space(4.0);

            // Follow body dropdown
            let current_follow = if follow_target.name.is_empty() {
                "None (Free Camera)"
            } else {
                &follow_target.name
            };
            egui::ComboBox::from_id_salt("follow_select")
                .selected_text(current_follow)
                .show_ui(ui, |ui| {
                    if ui.selectable_label(follow_target.entity.is_none(), "None (Free Camera)").clicked() {
                        follow_target.entity = None;
                        follow_target.name.clear();
                    }
                    for (entity, body) in named_query.iter() {
                        let selected = follow_target.entity == Some(entity);
                        if ui.selectable_label(selected, &body.name).clicked() {
                            follow_target.entity = Some(entity);
                            follow_target.name = body.name.clone();
                        }
                    }
                });

            if ui.button("Auto-Fit All").clicked() {
                let dist = compute_auto_fit_distance(&body_query);
                orbit.distance = dist;
                orbit.focus = Vec3::ZERO;
                follow_target.entity = None;
                follow_target.name.clear();
            }

            // Zoom slider
            let mut zoom = orbit.distance;
            if ui.add(egui::Slider::new(&mut zoom, 0.5..=5000.0).logarithmic(true).text("Zoom")).changed() {
                orbit.distance = zoom;
            }

            ui.add_space(12.0);
            ui.separator();

            if ui.button("About...").clicked() {
                about_open.0 = true;
            }
        });
}
