use bevy::prelude::*;
use physics_core::constants::G;

use crate::physics_plugin::{
    BlackHoleMarker, BlackHoleRegistry, BodyType, CelestialBody, SimulationState, AU, C_LIGHT,
    RENDER_SCALE, SUN_MASS,
};
use crate::render_plugin::{DisplayScale, RenderInitialized};

// ---------------------------------------------------------------------------
// Plugin
// ---------------------------------------------------------------------------

pub struct BlackHolePlugin;

impl Plugin for BlackHolePlugin {
    fn build(&self, app: &mut App) {
        app.add_systems(Update, (sync_black_hole_visuals, draw_accretion_disks))
            .add_systems(FixedUpdate, check_event_horizon_capture);
    }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Schwarzschild radius: r_s = 2GM / c^2
fn schwarzschild_radius(mass_kg: f64) -> f64 {
    2.0 * G * mass_kg / (C_LIGHT * C_LIGHT)
}

/// Convert Schwarzschild radius (meters) to render-space radius.
fn schwarzschild_render_radius(mass_kg: f64) -> f32 {
    let r_s = schwarzschild_radius(mass_kg);
    let r_render = (r_s / AU * RENDER_SCALE) as f32;
    r_render.max(0.3) // minimum visual radius for visibility
}

// ---------------------------------------------------------------------------
// Systems
// ---------------------------------------------------------------------------

/// Add / remove BlackHoleMarker and update display scale based on registry.
fn sync_black_hole_visuals(
    bh_registry: Res<BlackHoleRegistry>,
    mut commands: Commands,
    mut query: Query<(
        Entity,
        &CelestialBody,
        Option<&mut DisplayScale>,
        Option<&BlackHoleMarker>,
    )>,
) {
    for (entity, body, display_scale, bh_marker) in &mut query {
        let active_hole = bh_registry.active.iter().find(|h| h.body_name == body.name);

        if let Some(hole) = active_hole {
            let mass_kg = hole.mass_solar * SUN_MASS;
            let visual_r = schwarzschild_render_radius(mass_kg);

            if bh_marker.is_none() {
                // Newly converted: add marker, force re-render
                commands
                    .entity(entity)
                    .insert(BlackHoleMarker)
                    .remove::<RenderInitialized>();
            }

            // Update visual radius
            if let Some(mut ds) = display_scale {
                ds.0 = visual_r;
            }
        } else if bh_marker.is_some() {
            // No longer a black hole: restore
            let original_radius = bh_registry
                .active
                .iter()
                .find(|h| h.body_name == body.name)
                .map(|h| h.original_display_radius)
                .unwrap_or(body.display_radius);

            commands
                .entity(entity)
                .remove::<BlackHoleMarker>()
                .remove::<RenderInitialized>();

            if let Some(mut ds) = display_scale {
                ds.0 = original_radius;
            }
        }
    }
}

/// Draw accretion disk gizmo rings and event horizon boundary around black holes.
fn draw_accretion_disks(
    bh_registry: Res<BlackHoleRegistry>,
    bh_query: Query<(&Transform, &CelestialBody), With<BlackHoleMarker>>,
    mut gizmos: Gizmos,
) {
    for (transform, body) in &bh_query {
        let Some(hole) = bh_registry.active.iter().find(|h| h.body_name == body.name) else {
            continue;
        };

        let mass_kg = hole.mass_solar * SUN_MASS;
        let visual_r = schwarzschild_render_radius(mass_kg);
        let center = transform.translation;

        let isco = visual_r * 3.0; // innermost stable circular orbit
        let outer = visual_r * 5.0;

        // Concentric accretion disk rings
        let num_rings: usize = 8;
        let segments: usize = 64;

        for i in 0..num_rings {
            let t = i as f32 / (num_rings - 1).max(1) as f32;
            let radius = isco + t * (outer - isco);

            // Orange-inner to dim-red-outer gradient
            let r_color = 1.0 - t * 0.3;
            let g_color = 0.6 - t * 0.4;
            let b_color = 0.1 - t * 0.08;
            let alpha = 0.8 - t * 0.5;
            let color = Color::srgba(r_color, g_color, b_color, alpha);

            draw_circle_gizmo(&mut gizmos, center, radius, segments, color);
        }

        // Event horizon boundary (dark red)
        draw_circle_gizmo(
            &mut gizmos,
            center,
            visual_r * 1.05,
            segments,
            Color::srgba(0.5, 0.0, 0.0, 0.9),
        );
    }
}

/// Draw a circle in the XZ plane using line segments.
fn draw_circle_gizmo(
    gizmos: &mut Gizmos,
    center: Vec3,
    radius: f32,
    segments: usize,
    color: Color,
) {
    for s in 0..segments {
        let a1 = (s as f32 / segments as f32) * std::f32::consts::TAU;
        let a2 = ((s + 1) as f32 / segments as f32) * std::f32::consts::TAU;

        let p1 = center + Vec3::new(radius * a1.cos(), 0.0, radius * a1.sin());
        let p2 = center + Vec3::new(radius * a2.cos(), 0.0, radius * a2.sin());

        gizmos.line(p1, p2, color);
    }
}

/// Capture (despawn) bodies that cross a black hole's event horizon.
fn check_event_horizon_capture(
    mut commands: Commands,
    bh_registry: Res<BlackHoleRegistry>,
    sim_state: Option<ResMut<SimulationState>>,
    bh_query: Query<(&Transform, &CelestialBody), With<BlackHoleMarker>>,
    body_query: Query<(Entity, &CelestialBody, &Transform), Without<BlackHoleMarker>>,
) {
    let Some(mut sim) = sim_state else { return };

    for (bh_tf, bh_body) in &bh_query {
        let Some(hole) = bh_registry
            .active
            .iter()
            .find(|h| h.body_name == bh_body.name)
        else {
            continue;
        };

        let mass_kg = hole.mass_solar * SUN_MASS;
        let visual_r = schwarzschild_render_radius(mass_kg);
        let bh_pos = bh_tf.translation;

        for (entity, body, body_tf) in &body_query {
            // Don't capture the Sun (gravity source) unless it's a different black hole
            if body.body_type == BodyType::Star {
                continue;
            }

            let dist = (body_tf.translation - bh_pos).length();
            if dist < visual_r {
                // Captured: zero mass (avoids index shifting), despawn entity
                if body.sim_index < sim.inner.n_bodies {
                    sim.inner.masses[body.sim_index] = 0.0;
                }
                commands.entity(entity).despawn_recursive();
            }
        }
    }
}
