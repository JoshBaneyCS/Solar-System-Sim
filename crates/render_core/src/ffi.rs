use crate::renderer::{BodyData, DistanceLine, Renderer, TrailData};
use crate::shapes::LineVertex;
use crate::spacetime::SpacetimeBody;
use std::slice;

/// Create a new GPU renderer. Returns null on failure.
#[unsafe(no_mangle)]
pub extern "C" fn render_create(width: u32, height: u32) -> *mut Renderer {
    match Renderer::new(width, height) {
        Some(r) => Box::into_raw(Box::new(r)),
        None => std::ptr::null_mut(),
    }
}

/// Set camera parameters.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn render_set_camera(
    h: *mut Renderer,
    zoom: f64,
    pan_x: f64,
    pan_y: f64,
    rot_x: f64,
    rot_y: f64,
    rot_z: f64,
    use_3d: u8,
    follow_x: f64,
    follow_y: f64,
    follow_z: f64,
) {
    let r = unsafe { &mut *h };
    r.camera.zoom = zoom;
    r.camera.pan_x = pan_x;
    r.camera.pan_y = pan_y;
    r.camera.rotation_x = rot_x;
    r.camera.rotation_y = rot_y;
    r.camera.rotation_z = rot_z;
    r.camera.use_3d = use_3d != 0;
    r.camera.follow_x = follow_x;
    r.camera.follow_y = follow_y;
    r.camera.follow_z = follow_z;
}

/// Set body (planet) data. Positions are world coords (meters), converted to screen internally.
/// `positions`: flat array [x0,y0,z0, x1,y1,z1, ...]
/// `colors`: flat array [r0,g0,b0,a0, r1,g1,b1,a1, ...] (0-1 range)
/// `radii`: display radius in pixels for each body
/// `sun_pos`: [x,y,z] world position of sun
/// `sun_color`: [r,g,b,a] (0-1 range)
/// `sun_radius`: display radius in pixels
#[unsafe(no_mangle)]
pub unsafe extern "C" fn render_set_bodies(
    h: *mut Renderer,
    n_bodies: u32,
    positions: *const f64,
    colors: *const f64,
    radii: *const f64,
    sun_pos: *const f64,
    sun_color: *const f64,
    sun_radius: f64,
) {
    let r = unsafe { &mut *h };
    let n = n_bodies as usize;

    let pos = unsafe { slice::from_raw_parts(positions, n * 3) };
    let col = unsafe { slice::from_raw_parts(colors, n * 4) };
    let rad = unsafe { slice::from_raw_parts(radii, n) };

    r.bodies.clear();
    for i in 0..n {
        let wx = pos[i * 3];
        let wy = pos[i * 3 + 1];
        let wz = pos[i * 3 + 2];
        let (sx, sy) = r.camera.world_to_screen(wx, wy, wz);
        r.bodies.push(BodyData {
            screen_x: sx,
            screen_y: sy,
            radius: rad[i] as f32,
            color: [col[i * 4] as f32, col[i * 4 + 1] as f32, col[i * 4 + 2] as f32, col[i * 4 + 3] as f32],
        });
    }

    // Sun
    let sp = unsafe { slice::from_raw_parts(sun_pos, 3) };
    let sc = unsafe { slice::from_raw_parts(sun_color, 4) };
    let (sx, sy) = r.camera.world_to_screen(sp[0], sp[1], sp[2]);
    r.sun = Some(BodyData {
        screen_x: sx,
        screen_y: sy,
        radius: sun_radius as f32,
        color: [sc[0] as f32, sc[1] as f32, sc[2] as f32, sc[3] as f32],
    });
}

/// Set trail data.
/// `trail_lengths`: number of trail points per body
/// `trail_positions`: flat [x0,y0,z0, x1,y1,z1, ...] for all trail points concatenated
/// `trail_colors`: flat [r,g,b,a] per body (not per point — alpha is computed per segment)
/// `show_trails`: whether to render trails
#[unsafe(no_mangle)]
pub unsafe extern "C" fn render_set_trails(
    h: *mut Renderer,
    n_bodies: u32,
    trail_lengths: *const u32,
    trail_positions: *const f64,
    trail_colors: *const f64,
    show_trails: u8,
) {
    let r = unsafe { &mut *h };
    r.show_trails = show_trails != 0;

    if !r.show_trails {
        r.trails.clear();
        return;
    }

    let n = n_bodies as usize;
    let lengths = unsafe { slice::from_raw_parts(trail_lengths, n) };
    let cols = unsafe { slice::from_raw_parts(trail_colors, n * 4) };

    // Calculate total trail points
    let total_points: u32 = lengths.iter().sum();
    let positions = unsafe { slice::from_raw_parts(trail_positions, total_points as usize * 3) };

    r.trails.clear();
    let mut offset = 0usize;

    for i in 0..n {
        let len = lengths[i] as usize;
        let base_color = [
            cols[i * 4] as f32,
            cols[i * 4 + 1] as f32,
            cols[i * 4 + 2] as f32,
            1.0f32,
        ];

        let mut vertices = Vec::new();

        if len > 1 {
            // Downsample if too many points
            let step = if len > 200 { len / 200 } else { 1 };

            let mut j = 0;
            while j + step < len {
                let idx1 = offset + j;
                let idx2 = offset + j + step;

                let (x1, y1) = r.camera.world_to_screen(
                    positions[idx1 * 3],
                    positions[idx1 * 3 + 1],
                    positions[idx1 * 3 + 2],
                );
                let (x2, y2) = r.camera.world_to_screen(
                    positions[idx2 * 3],
                    positions[idx2 * 3 + 1],
                    positions[idx2 * 3 + 2],
                );

                // Alpha fade: older segments more transparent
                let alpha = j as f32 / len as f32;
                let color = [base_color[0], base_color[1], base_color[2], alpha];

                vertices.push(LineVertex { position: [x1, y1], color });
                vertices.push(LineVertex { position: [x2, y2], color });

                j += step;
            }
        }

        r.trails.push(TrailData { vertices });
        offset += len;
    }
}

/// Set spacetime visualization parameters.
/// `masses`: mass per body (including sun as first element)
/// `positions`: flat [x,y,z,...] for each body
#[unsafe(no_mangle)]
pub unsafe extern "C" fn render_set_spacetime(
    h: *mut Renderer,
    show_spacetime: u8,
    masses: *const f64,
    positions: *const f64,
    n_bodies: u32,
) {
    let r = unsafe { &mut *h };
    r.show_spacetime = show_spacetime != 0;

    if !r.show_spacetime {
        r.spacetime_bodies.clear();
        return;
    }

    let n = n_bodies as usize;
    let m = unsafe { slice::from_raw_parts(masses, n) };
    let p = unsafe { slice::from_raw_parts(positions, n * 3) };

    r.spacetime_bodies.clear();
    for i in 0..n {
        r.spacetime_bodies.push(SpacetimeBody {
            x: p[i * 3],
            y: p[i * 3 + 1],
            mass: m[i],
        });
    }
}

/// Set distance measurement line between two selected bodies.
/// Pass null pointers to clear.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn render_set_distance_line(
    h: *mut Renderer,
    has_line: u8,
    x1: f64, y1: f64, z1: f64,
    x2: f64, y2: f64, z2: f64,
) {
    let r = unsafe { &mut *h };
    if has_line == 0 {
        r.distance_line = None;
    } else {
        let (sx1, sy1) = r.camera.world_to_screen(x1, y1, z1);
        let (sx2, sy2) = r.camera.world_to_screen(x2, y2, z2);
        r.distance_line = Some(DistanceLine {
            x1: sx1, y1: sy1,
            x2: sx2, y2: sy2,
        });
    }
}

/// Enable or disable ray tracing mode.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn render_set_rt_mode(h: *mut Renderer, enabled: u8) {
    let r = unsafe { &mut *h };
    r.rt_enabled = enabled != 0;
    r.rt_frame_count = 0;
}

/// Set ray tracing quality parameters.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn render_set_rt_quality(
    h: *mut Renderer,
    samples_per_frame: u32,
    max_bounces: u32,
) {
    let r = unsafe { &mut *h };
    r.rt_samples_per_frame = samples_per_frame;
    r.rt_max_bounces = max_bounces;
    r.rt_frame_count = 0;
}

/// Render a frame and return pointer to RGBA pixel data.
/// The returned pointer is valid until the next call to render_frame or render_free.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn render_frame(h: *mut Renderer) -> *const u8 {
    let r = unsafe { &mut *h };
    let data = r.render_frame();
    data.as_ptr()
}

/// Resize the render target.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn render_resize(h: *mut Renderer, width: u32, height: u32) {
    let r = unsafe { &mut *h };
    r.resize(width, height);
}

/// Free the renderer.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn render_free(h: *mut Renderer) {
    if !h.is_null() {
        let _ = unsafe { Box::from_raw(h) };
    }
}
