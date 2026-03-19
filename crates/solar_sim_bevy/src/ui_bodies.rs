use bevy::prelude::*;
use bevy_egui::{egui, EguiContexts};

use crate::physics_plugin::{BodyType, CelestialBody, SimulationState, AU};

/// Resource to track which body to follow.
#[derive(Resource, Default)]
pub struct FollowTarget {
    pub entity: Option<Entity>,
    pub name: String,
}

pub fn bodies_panel(
    mut contexts: EguiContexts,
    mut follow_target: ResMut<FollowTarget>,
    sim_state: Option<Res<SimulationState>>,
    body_query: Query<(Entity, &CelestialBody, &Transform)>,
) {
    let ctx = contexts.ctx_mut();

    egui::SidePanel::right("bodies_panel")
        .default_width(260.0)
        .resizable(true)
        .show(ctx, |ui| {
            ui.heading("Bodies");
            ui.separator();

            egui::ScrollArea::vertical().show(ui, |ui| {
                let sim = sim_state.as_ref();

                // Group bodies by type
                let groups: &[(BodyType, &str)] = &[
                    (BodyType::Star, "Star"),
                    (BodyType::Planet, "Planets"),
                    (BodyType::DwarfPlanet, "Dwarf Planets"),
                    (BodyType::Moon, "Moons"),
                    (BodyType::Comet, "Comets"),
                    (BodyType::Asteroid, "Asteroids"),
                ];

                for &(btype, label) in groups {
                    let bodies: Vec<_> = body_query
                        .iter()
                        .filter(|(_, b, _)| b.body_type == btype)
                        .collect();

                    if bodies.is_empty() {
                        continue;
                    }

                    ui.add_space(4.0);
                    ui.strong(label);
                    ui.separator();

                    for (entity, body, _transform) in &bodies {
                        ui.horizontal(|ui| {
                            ui.label(&body.name);

                            let is_following = follow_target.entity == Some(*entity);

                            if ui.selectable_label(is_following, "Follow").clicked() {
                                if is_following {
                                    follow_target.entity = None;
                                    follow_target.name.clear();
                                } else {
                                    follow_target.entity = Some(*entity);
                                    follow_target.name = body.name.clone();
                                }
                            }
                        });

                        // Show body info
                        if let Some(sim) = &sim {
                            if body.sim_index < sim.inner.n_bodies {
                                let pos = sim.inner.positions[body.sim_index];
                                let vel = sim.inner.velocities[body.sim_index];
                                let dist_au =
                                    (pos.x * pos.x + pos.y * pos.y + pos.z * pos.z).sqrt() / AU;
                                let speed_kms =
                                    (vel.x * vel.x + vel.y * vel.y + vel.z * vel.z).sqrt() / 1000.0;

                                ui.indent(body.name.as_str(), |ui| {
                                    ui.label(format!("Dist: {:.3} AU", dist_au));
                                    ui.label(format!("Vel: {:.2} km/s", speed_kms));
                                });
                            }
                        } else if body.body_type == BodyType::Star {
                            ui.indent("sun_info", |ui| {
                                ui.label("Dist: 0.000 AU");
                            });
                        }
                    }
                }
            });
        });
}
