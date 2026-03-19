use bevy::prelude::*;

use crate::physics_plugin::{
    BlackHoleMarker, BlackHoleRegistry, CelestialBody, SimulationConfig, SimulationState, Sun, AU,
    RENDER_SCALE,
};
use crate::render_plugin::DisplayScale;

// ---------------------------------------------------------------------------
// Plugin
// ---------------------------------------------------------------------------

pub struct SpacetimePlugin;

impl Plugin for SpacetimePlugin {
    fn build(&self, app: &mut App) {
        app.add_systems(Update, draw_spacetime_grid);
    }
}

// ---------------------------------------------------------------------------
// Body info for displacement calculation
// ---------------------------------------------------------------------------

struct BodyWellInfo {
    render_x: f32,
    render_z: f32,
    well_depth: f32,
    well_width: f32,
}

// ---------------------------------------------------------------------------
// System
// ---------------------------------------------------------------------------

/// Draw a deformed grid in the XZ plane representing gravitational potential.
fn draw_spacetime_grid(
    config: Res<SimulationConfig>,
    sim_state: Option<Res<SimulationState>>,
    bh_registry: Res<BlackHoleRegistry>,
    body_query: Query<
        (
            &CelestialBody,
            &Transform,
            Option<&DisplayScale>,
            Option<&BlackHoleMarker>,
        ),
        Without<Sun>,
    >,
    mut gizmos: Gizmos,
) {
    if !config.show_spacetime {
        return;
    }

    let Some(sim) = &sim_state else { return };

    // Collect body well info for the displacement function
    let body_wells: Vec<BodyWellInfo> = body_query
        .iter()
        .map(|(body, tf, ds, bh)| {
            // For black holes, use the (larger) DisplayScale; otherwise use display_radius
            let radius = if bh.is_some() {
                ds.map(|d| d.0).unwrap_or(body.display_radius)
            } else {
                body.display_radius
            };
            BodyWellInfo {
                render_x: tf.translation.x,
                render_z: tf.translation.z,
                well_depth: radius * 3.0,
                well_width: radius * 3.0,
            }
        })
        .collect();

    let grid_size = 40;
    let grid_range = 35.0_f32; // Bevy world units (covers ~3.5 AU)
    let step = grid_range * 2.0 / grid_size as f32;
    let color = Color::srgba(0.0, 0.6, 0.8, 0.15);

    // Draw grid lines along X
    for i in 0..=grid_size {
        let z = -grid_range + i as f32 * step;
        let mut prev: Option<Vec3> = None;
        for j in 0..=grid_size {
            let x = -grid_range + j as f32 * step;
            let y = compute_potential_displacement(x, z, sim, &bh_registry, &body_wells);
            let pos = Vec3::new(x, y, z);
            if let Some(p) = prev {
                gizmos.line(p, pos, color);
            }
            prev = Some(pos);
        }
    }

    // Draw grid lines along Z
    for j in 0..=grid_size {
        let x = -grid_range + j as f32 * step;
        let mut prev: Option<Vec3> = None;
        for i in 0..=grid_size {
            let z = -grid_range + i as f32 * step;
            let y = compute_potential_displacement(x, z, sim, &bh_registry, &body_wells);
            let pos = Vec3::new(x, y, z);
            if let Some(p) = prev {
                gizmos.line(p, pos, color);
            }
            prev = Some(pos);
        }
    }
}

/// Compute Y displacement based on gravitational potential at (x, z) in render coords.
fn compute_potential_displacement(
    x: f32,
    z: f32,
    sim: &SimulationState,
    bh_registry: &BlackHoleRegistry,
    body_wells: &[BodyWellInfo],
) -> f32 {
    let g_constant = 6.674e-11_f64;

    // Convert render coords back to approximate physics coords (AU)
    let px_au = x as f64 / RENDER_SCALE;
    let pz_au = z as f64 / RENDER_SCALE;

    let mut potential = 0.0_f64;

    // Sun contribution (physics-based)
    let r_sun = (px_au * px_au + pz_au * pz_au).sqrt();
    if r_sun > 0.01 {
        potential -= g_constant * sim.inner.sun_mass / (r_sun * AU);
    }

    // Scale Sun potential to visible displacement
    let mut displacement = (potential / 4e8) as f32;

    // Per-body local wells (display-radius-based for visibility)
    // Uses a 1/r shaped well with depth and width proportional to display_radius.
    // This ensures all bodies create visible spacetime curvature.
    for well in body_wells {
        let dx = x - well.render_x;
        let dz = z - well.render_z;
        let r = (dx * dx + dz * dz).sqrt();
        // 1/r well with softening to avoid singularity
        displacement -= well.well_depth * well.well_width / (r + well.well_width);
    }

    // Sun black hole: deepen the central well significantly
    if bh_registry.active.iter().any(|h| h.body_name == "Sun") {
        let r_center = (x * x + z * z).sqrt();
        let bh_depth = 3.0_f32;
        let bh_width = 1.0_f32;
        displacement -= bh_depth * bh_width / (r_center + bh_width);
    }

    displacement.clamp(-5.0, 0.0)
}
