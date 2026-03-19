use bevy::prelude::*;
use bevy_egui::{egui, EguiContexts};

use crate::launch_core::{
    self, LaunchPlan, ReferenceFrame, Trajectory, DESTINATIONS, EARTH_ORBIT_SMA, VEHICLES,
};
use crate::physics_plugin::{AU, RENDER_SCALE};

// ---------------------------------------------------------------------------
// Resources
// ---------------------------------------------------------------------------

#[derive(Resource)]
pub struct LaunchState {
    pub selected_vehicle: usize,
    pub selected_destination: usize,
    pub plan: Option<LaunchPlan>,
    pub trajectory: Option<Trajectory>,
    pub summary_text: String,
    // Mission playback
    pub playback_active: bool,
    pub playback_time: f64,
    pub playback_speed: f64,
    pub playback_playing: bool,
}

impl Default for LaunchState {
    fn default() -> Self {
        Self {
            selected_vehicle: 0,
            selected_destination: 0,
            plan: None,
            trajectory: None,
            summary_text: String::new(),
            playback_active: false,
            playback_time: 0.0,
            playback_speed: 100.0,
            playback_playing: false,
        }
    }
}

// ---------------------------------------------------------------------------
// Plugin
// ---------------------------------------------------------------------------

pub struct LaunchPlugin;

impl Plugin for LaunchPlugin {
    fn build(&self, app: &mut App) {
        app.insert_resource(LaunchState::default()).add_systems(
            Update,
            (launch_panel, draw_trajectory, update_playback).chain(),
        );
    }
}

// ---------------------------------------------------------------------------
// UI Panel
// ---------------------------------------------------------------------------

fn launch_panel(mut contexts: EguiContexts, mut state: ResMut<LaunchState>) {
    let ctx = contexts.ctx_mut();

    egui::Window::new("Launch Planner")
        .default_width(350.0)
        .default_pos(egui::pos2(300.0, 50.0))
        .resizable(true)
        .show(ctx, |ui| {
            ui.label("Kennedy Space Center");
            ui.add_space(8.0);

            // Destination
            ui.horizontal(|ui| {
                ui.label("Destination:");
                egui::ComboBox::from_id_salt("dest_select")
                    .selected_text(DESTINATIONS[state.selected_destination].name)
                    .show_ui(ui, |ui| {
                        for (i, d) in DESTINATIONS.iter().enumerate() {
                            ui.selectable_value(&mut state.selected_destination, i, d.name);
                        }
                    });
            });

            // Vehicle
            ui.horizontal(|ui| {
                ui.label("Vehicle:");
                egui::ComboBox::from_id_salt("vehicle_select")
                    .selected_text(VEHICLES[state.selected_vehicle].name)
                    .show_ui(ui, |ui| {
                        for (i, v) in VEHICLES.iter().enumerate() {
                            let dv = launch_core::total_vehicle_delta_v(v);
                            let label = format!("{} ({:.1} km/s)", v.name, dv / 1000.0);
                            ui.selectable_value(&mut state.selected_vehicle, i, label);
                        }
                    });
            });

            ui.add_space(8.0);

            ui.horizontal(|ui| {
                if ui.button("Simulate Launch").clicked() {
                    let vehicle = &VEHICLES[state.selected_vehicle];
                    let dest = &DESTINATIONS[state.selected_destination];
                    let plan = launch_core::plan(vehicle, dest);
                    let traj = launch_core::propagate_trajectory(&plan, dest);
                    state.summary_text = launch_core::summary(&plan);
                    state.trajectory = Some(traj);
                    state.plan = Some(plan);
                    state.playback_active = true;
                    state.playback_time = 0.0;
                    state.playback_playing = false;
                }

                if ui.button("Clear").clicked() {
                    state.plan = None;
                    state.trajectory = None;
                    state.summary_text.clear();
                    state.playback_active = false;
                    state.playback_time = 0.0;
                    state.playback_playing = false;
                }
            });

            // Results
            if !state.summary_text.is_empty() {
                ui.add_space(8.0);
                ui.separator();
                egui::ScrollArea::vertical()
                    .max_height(200.0)
                    .show(ui, |ui| {
                        ui.monospace(&state.summary_text);
                    });
            }

            // Mission playback controls
            if state.playback_active {
                ui.add_space(8.0);
                ui.separator();
                ui.strong("Mission Playback");

                ui.horizontal(|ui| {
                    if ui
                        .button(if state.playback_playing {
                            "Pause"
                        } else {
                            "Play"
                        })
                        .clicked()
                    {
                        state.playback_playing = !state.playback_playing;
                    }
                });

                let mut speed_exp = (state.playback_speed.log2()) as f32;
                if ui
                    .add(egui::Slider::new(&mut speed_exp, 0.0..=6.0).text("Speed 2^x"))
                    .changed()
                {
                    state.playback_speed = 2.0_f64.powf(speed_exp as f64);
                }

                // Get trajectory info before mutable borrow
                let traj_info = state.trajectory.as_ref().map(|traj| {
                    let total_time = traj.points.last().map(|p| p.time).unwrap_or(1.0);
                    let playback_time = state.playback_time;
                    let interp = interpolate_trajectory(traj, playback_time);
                    (total_time, interp)
                });

                if let Some((total_time, interp)) = traj_info {
                    let mut progress = (state.playback_time / total_time * 100.0) as f32;
                    if ui
                        .add(egui::Slider::new(&mut progress, 0.0..=100.0).text("Timeline %"))
                        .changed()
                    {
                        state.playback_time = progress as f64 / 100.0 * total_time;
                    }

                    let elapsed_days = state.playback_time / 86400.0;
                    let pct = state.playback_time / total_time * 100.0;

                    if let Some((pos, vel)) = interp {
                        let speed = (vel[0] * vel[0] + vel[1] * vel[1] + vel[2] * vel[2]).sqrt();
                        let dist = (pos[0] * pos[0] + pos[1] * pos[1] + pos[2] * pos[2]).sqrt();

                        ui.label(format!("Elapsed: {:.2} days", elapsed_days));
                        ui.label(format!("Speed: {:.2} km/s", speed / 1000.0));
                        ui.label(format!("Distance: {:.3e} m", dist));
                        ui.label(format!("Progress: {:.1}%", pct));
                    }
                }
            }
        });
}

// ---------------------------------------------------------------------------
// Trajectory rendering
// ---------------------------------------------------------------------------

fn draw_trajectory(state: Res<LaunchState>, mut gizmos: Gizmos) {
    let Some(traj) = &state.trajectory else {
        return;
    };

    let points: Vec<Vec3> = traj
        .points
        .iter()
        .map(|p| trajectory_to_render(p.position, traj.frame))
        .collect();

    let total = points.len();
    if total < 2 {
        return;
    }

    for i in 0..total - 1 {
        let t = i as f32 / total as f32;
        // Color gradient: red -> yellow -> cyan
        let color = if t < 0.5 {
            let t2 = t * 2.0;
            Color::srgb(1.0 - t2 * 0.5, t2, 0.0)
        } else {
            let t2 = (t - 0.5) * 2.0;
            Color::srgb(0.5 - t2 * 0.5, 1.0 - t2, t2)
        };
        gizmos.line(points[i], points[i + 1], color);
    }

    // Draw vehicle marker at playback position
    if state.playback_active {
        if let Some(traj_data) = &state.trajectory {
            if let Some((pos, _)) = interpolate_trajectory(traj_data, state.playback_time) {
                let render_pos = trajectory_to_render(pos, traj.frame);
                gizmos.sphere(
                    Isometry3d::from_translation(render_pos),
                    0.05,
                    Color::srgb(0.0, 1.0, 1.0),
                );
            }
        }
    }
}

fn trajectory_to_render(pos: [f64; 3], frame: ReferenceFrame) -> Vec3 {
    match frame {
        ReferenceFrame::Heliocentric => Vec3::new(
            (pos[0] / AU * RENDER_SCALE) as f32,
            (pos[2] / AU * RENDER_SCALE) as f32,
            (pos[1] / AU * RENDER_SCALE) as f32,
        ),
        ReferenceFrame::EarthCentered => {
            // Scale Earth-centered trajectory relative to Earth's position
            // For now, render at a small scale near the origin
            let scale = 1.0 / EARTH_ORBIT_SMA * RENDER_SCALE;
            Vec3::new(
                (pos[0] * scale) as f32,
                (pos[2] * scale) as f32,
                (pos[1] * scale) as f32,
            )
        }
    }
}

// ---------------------------------------------------------------------------
// Playback
// ---------------------------------------------------------------------------

fn update_playback(time: Res<Time>, mut state: ResMut<LaunchState>) {
    if !state.playback_active || !state.playback_playing {
        return;
    }

    let Some(traj) = &state.trajectory else {
        return;
    };
    let total_time = traj.points.last().map(|p| p.time).unwrap_or(1.0);

    state.playback_time += time.delta_secs_f64() * state.playback_speed;
    if state.playback_time >= total_time {
        state.playback_time = total_time;
        state.playback_playing = false;
    }
}

fn interpolate_trajectory(traj: &Trajectory, time: f64) -> Option<([f64; 3], [f64; 3])> {
    if traj.points.is_empty() {
        return None;
    }
    if traj.points.len() == 1 {
        let p = &traj.points[0];
        return Some((p.position, p.velocity));
    }

    // Find bracketing points
    let mut i = 0;
    while i < traj.points.len() - 1 && traj.points[i + 1].time < time {
        i += 1;
    }

    if i >= traj.points.len() - 1 {
        let p = traj.points.last().unwrap();
        return Some((p.position, p.velocity));
    }

    let p0 = &traj.points[i];
    let p1 = &traj.points[i + 1];
    let dt = p1.time - p0.time;
    if dt <= 0.0 {
        return Some((p0.position, p0.velocity));
    }

    let t = (time - p0.time) / dt;
    let pos = [
        p0.position[0] + (p1.position[0] - p0.position[0]) * t,
        p0.position[1] + (p1.position[1] - p0.position[1]) * t,
        p0.position[2] + (p1.position[2] - p0.position[2]) * t,
    ];
    let vel = [
        p0.velocity[0] + (p1.velocity[0] - p0.velocity[0]) * t,
        p0.velocity[1] + (p1.velocity[1] - p0.velocity[1]) * t,
        p0.velocity[2] + (p1.velocity[2] - p0.velocity[2]) * t,
    ];

    Some((pos, vel))
}
