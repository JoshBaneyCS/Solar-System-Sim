use bevy::prelude::*;
use bevy_egui::{egui, EguiContexts};

#[derive(Resource, Default)]
pub struct AboutWindowOpen(pub bool);

pub fn about_dialog(mut contexts: EguiContexts, mut open: ResMut<AboutWindowOpen>) {
    if !open.0 {
        return;
    }

    let ctx = contexts.ctx_mut();

    egui::Window::new("About Solar System Simulator")
        .collapsible(false)
        .resizable(false)
        .default_width(380.0)
        .open(&mut open.0)
        .show(ctx, |ui| {
            ui.vertical_centered(|ui| {
                ui.heading("Solar System Simulator");
                ui.label("Version 0.2.0 (Bevy)");
                ui.add_space(8.0);
                ui.label("by Joshua Baney");
                ui.add_space(4.0);
                ui.hyperlink_to(
                    "GitHub Repository",
                    "https://github.com/joshbaney/solar-system-simulator",
                );
            });

            ui.add_space(12.0);
            ui.separator();

            ui.strong("System Information");
            ui.label(format!("OS: {}", std::env::consts::OS));
            ui.label(format!("Arch: {}", std::env::consts::ARCH));

            ui.add_space(12.0);
            ui.separator();

            ui.strong("Credits");
            ui.label("- Bevy Engine (bevy.rs)");
            ui.label("- egui (emilk/egui)");
            ui.label("- Planet textures: Solar System Scope (CC BY 4.0)");
            ui.label("- Skybox: NASA/ESA/Hubble");
            ui.label("- Physics: NASA/JPL orbital elements");
            ui.label("- Built with Rust");

            ui.add_space(8.0);
            ui.label("License: MIT");
        });
}
