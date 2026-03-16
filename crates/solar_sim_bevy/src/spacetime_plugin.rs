use bevy::prelude::*;

use crate::physics_plugin::{SimulationConfig, SimulationState, AU, RENDER_SCALE};

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
// System
// ---------------------------------------------------------------------------

/// Draw a deformed grid in the XZ plane representing gravitational potential.
fn draw_spacetime_grid(
    config: Res<SimulationConfig>,
    sim_state: Option<Res<SimulationState>>,
    mut gizmos: Gizmos,
) {
    if !config.show_spacetime {
        return;
    }

    let Some(sim) = &sim_state else { return };

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
            let y = compute_potential_displacement(x, z, sim);
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
            let y = compute_potential_displacement(x, z, sim);
            let pos = Vec3::new(x, y, z);
            if let Some(p) = prev {
                gizmos.line(p, pos, color);
            }
            prev = Some(pos);
        }
    }
}

/// Compute Y displacement based on gravitational potential at (x, z) in render coords.
fn compute_potential_displacement(x: f32, z: f32, sim: &SimulationState) -> f32 {
    let g_constant = 6.674e-11_f64;
    let sun_mass = 1.989e30_f64;

    // Convert render coords back to approximate physics coords (AU)
    let px_au = x as f64 / RENDER_SCALE;
    let pz_au = z as f64 / RENDER_SCALE;

    let mut potential = 0.0_f64;

    // Sun contribution
    let r_sun = (px_au * px_au + pz_au * pz_au).sqrt();
    if r_sun > 0.01 {
        potential -= g_constant * sun_mass / (r_sun * AU);
    }

    // Planet contributions
    for i in 0..sim.inner.n_bodies {
        let pos = sim.inner.positions[i];
        let dx = px_au - pos.x / AU;
        let dz = pz_au - pos.y / AU; // Note: physics Y maps to render Z
        let r = (dx * dx + dz * dz).sqrt();
        if r > 0.01 {
            potential -= g_constant * sim.inner.masses[i] / (r * AU);
        }
    }

    // Scale potential to a visible displacement
    // Normalize: potential near Sun ~ -8.8e8 J/kg, we want ~-2 Bevy units displacement
    let displacement = (potential / 4e8) as f32;
    displacement.clamp(-5.0, 0.0)
}
