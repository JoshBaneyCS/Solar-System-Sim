const G: f64 = 6.67430e-11;
const C: f64 = 299_792_458.0;
const AU: f64 = 1.496e11;

const DEFAULT_GRID_RESOLUTION: usize = 80;
const DISPLACEMENT_FACTOR: f64 = 50.0;
const MAX_INFLUENCE_DISTANCE: f64 = 10.0 * AU;
const CURVATURE_SCALE_FACTOR: f64 = 1e11;
const MIN_GRID_RESOLUTION: usize = 40;
const MAX_GRID_RESOLUTION: usize = 120;

use crate::shapes::LineVertex;

/// Body data needed for spacetime calculation (position + mass).
pub struct SpacetimeBody {
    pub x: f64,
    pub y: f64,
    pub mass: f64,
}

fn adaptive_resolution(zoom: f64) -> usize {
    if zoom < 0.5 {
        MIN_GRID_RESOLUTION
    } else if zoom < 2.0 {
        80
    } else if zoom < 10.0 {
        100
    } else {
        MAX_GRID_RESOLUTION
    }
}

/// Interpolate normalized potential (0-1) to RGBA color.
/// Gradient: dark blue → magenta → red → yellow, alpha 180/255.
fn interpolate_color(normalized_potential: f64) -> [f32; 4] {
    let t = (1.0 - normalized_potential).clamp(0.0, 1.0);
    let alpha = 180.0 / 255.0;

    if t < 0.33 {
        let ratio = t / 0.33;
        [
            (100.0 * ratio) / 255.0,
            0.0,
            (50.0 + 100.0 * ratio) / 255.0,
            alpha,
        ]
    } else if t < 0.66 {
        let ratio = (t - 0.33) / 0.33;
        [
            (100.0 + 100.0 * ratio) / 255.0,
            0.0,
            (150.0 - 150.0 * ratio) / 255.0,
            alpha,
        ]
    } else {
        let ratio = (t - 0.66) / 0.34;
        [
            (200.0 + 55.0 * ratio) / 255.0,
            (100.0 * ratio) / 255.0,
            0.0,
            alpha,
        ]
    }
}

/// Generate spacetime grid line vertices from body positions and masses.
/// Returns line vertices ready for the line pipeline.
pub fn generate_grid(
    width: f64,
    height: f64,
    zoom: f64,
    pan_x: f64,
    pan_y: f64,
    bodies: &[SpacetimeBody],
) -> Vec<LineVertex> {
    let resolution = adaptive_resolution(zoom);
    let display_scale = 100.0 * zoom;

    let world_width = width / display_scale * AU;
    let world_height = height / display_scale * AU;

    let viewport_center_x = pan_x * AU;
    let viewport_center_y = pan_y * AU;

    let grid_spacing_x = world_width / (resolution as f64 - 1.0);
    let grid_spacing_y = world_height / (resolution as f64 - 1.0);

    let effective_influence = if zoom > 2.0 {
        MAX_INFLUENCE_DISTANCE * zoom
    } else {
        MAX_INFLUENCE_DISTANCE
    };

    // Calculate potential field
    let mut potentials = vec![vec![0.0f64; resolution]; resolution];

    for i in 0..resolution {
        for j in 0..resolution {
            let world_x = viewport_center_x - world_width / 2.0 + i as f64 * grid_spacing_x;
            let world_y = viewport_center_y - world_height / 2.0 + j as f64 * grid_spacing_y;

            let mut curvature = 0.0;

            for body in bodies {
                let rx = body.x - world_x;
                let ry = body.y - world_y;
                let r = (rx * rx + ry * ry).sqrt();

                if r > 1e6 && r < effective_influence {
                    let h00 = 2.0 * G * body.mass / (C * C * r);
                    curvature += h00;
                }
            }

            potentials[i][j] = curvature * CURVATURE_SCALE_FACTOR;
        }
    }

    // Normalize potentials to 0-1
    let mut min_p = f64::MAX;
    let mut max_p = f64::MIN;
    for row in &potentials {
        for &v in row {
            if v < min_p { min_p = v; }
            if v > max_p { max_p = v; }
        }
    }
    let range = max_p - min_p;
    if range > 0.0 {
        for row in &mut potentials {
            for v in row {
                *v = (*v - min_p) / range;
            }
        }
    } else {
        for row in &mut potentials {
            for v in row {
                *v = 0.0;
            }
        }
    }

    let effective_displacement = DISPLACEMENT_FACTOR * (1.0 + zoom / 2.0);

    let screen_spacing_x = width as f32 / (resolution as f32 - 1.0);
    let screen_spacing_y = height as f32 / (resolution as f32 - 1.0);

    // Estimate: 2 vertices per segment, (res-1)*res horizontal + res*(res-1) vertical
    let n_segments = 2 * resolution * (resolution - 1);
    let mut vertices = Vec::with_capacity(n_segments * 2);

    // Horizontal lines
    for j in 0..resolution {
        for i in 0..(resolution - 1) {
            let x1 = i as f32 * screen_spacing_x;
            let y1 = j as f32 * screen_spacing_y + (potentials[i][j] * effective_displacement) as f32;
            let x2 = (i + 1) as f32 * screen_spacing_x;
            let y2 = j as f32 * screen_spacing_y + (potentials[i + 1][j] * effective_displacement) as f32;

            let avg = (potentials[i][j] + potentials[i + 1][j]) / 2.0;
            let color = interpolate_color(avg);

            vertices.push(LineVertex { position: [x1, y1], color });
            vertices.push(LineVertex { position: [x2, y2], color });
        }
    }

    // Vertical lines
    for i in 0..resolution {
        for j in 0..(resolution - 1) {
            let x1 = i as f32 * screen_spacing_x;
            let y1 = j as f32 * screen_spacing_y + (potentials[i][j] * effective_displacement) as f32;
            let x2 = i as f32 * screen_spacing_x;
            let y2 = (j + 1) as f32 * screen_spacing_y + (potentials[i][j + 1] * effective_displacement) as f32;

            let avg = (potentials[i][j] + potentials[i][j + 1]) / 2.0;
            let color = interpolate_color(avg);

            vertices.push(LineVertex { position: [x1, y1], color });
            vertices.push(LineVertex { position: [x2, y2], color });
        }
    }

    vertices
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_interpolate_color_bounds() {
        let c0 = interpolate_color(0.0);
        let c1 = interpolate_color(1.0);
        let c_mid = interpolate_color(0.5);
        for c in [c0, c1, c_mid] {
            for &v in &c {
                assert!(v >= 0.0 && v <= 1.0, "color component out of range: {v}");
            }
        }
    }

    #[test]
    fn test_generate_grid_empty_bodies() {
        let verts = generate_grid(800.0, 600.0, 1.0, 0.0, 0.0, &[]);
        // With no bodies, potentials are all zero, normalized to 0, still produces grid vertices
        assert!(!verts.is_empty());
    }

    #[test]
    fn test_generate_grid_with_sun() {
        let bodies = vec![SpacetimeBody { x: 0.0, y: 0.0, mass: 1.989e30 }];
        let verts = generate_grid(800.0, 600.0, 1.0, 0.0, 0.0, &bodies);
        assert!(!verts.is_empty());
        // Should have line vertices for horizontal + vertical grid
        let res = 80usize; // default for zoom=1.0
        let expected_segments = 2 * res * (res - 1);
        assert_eq!(verts.len(), expected_segments * 2);
    }

    #[test]
    fn test_adaptive_resolution() {
        assert_eq!(adaptive_resolution(0.3), MIN_GRID_RESOLUTION);
        assert_eq!(adaptive_resolution(1.0), 80);
        assert_eq!(adaptive_resolution(5.0), 100);
        assert_eq!(adaptive_resolution(15.0), MAX_GRID_RESOLUTION);
    }
}
