const AU: f64 = 1.496e11;
const DEFAULT_DISPLAY_SCALE: f64 = 100.0;

pub struct Camera {
    pub zoom: f64,
    pub pan_x: f64,
    pub pan_y: f64,
    pub rotation_x: f64,
    pub rotation_y: f64,
    pub rotation_z: f64,
    pub use_3d: bool,
    pub follow_x: f64,
    pub follow_y: f64,
    pub follow_z: f64,
    pub width: u32,
    pub height: u32,
}

impl Camera {
    pub fn new(width: u32, height: u32) -> Self {
        Self {
            zoom: 1.0,
            pan_x: 0.0,
            pan_y: 0.0,
            rotation_x: 0.0,
            rotation_y: 0.0,
            rotation_z: 0.0,
            use_3d: false,
            follow_x: 0.0,
            follow_y: 0.0,
            follow_z: 0.0,
            width,
            height,
        }
    }

    #[inline]
    pub fn display_scale(&self) -> f64 {
        DEFAULT_DISPLAY_SCALE * self.zoom
    }

    /// Port of viewport.go WorldToScreen — converts 3D world position (meters) to screen pixels.
    #[inline]
    pub fn world_to_screen(&self, wx: f64, wy: f64, wz: f64) -> (f32, f32) {
        let mut x = wx;
        let mut y = wy;
        let mut z = wz;

        if self.use_3d {
            if self.rotation_x != 0.0 {
                let cos_x = self.rotation_x.cos();
                let sin_x = self.rotation_x.sin();
                let new_y = y * cos_x - z * sin_x;
                let new_z = y * sin_x + z * cos_x;
                y = new_y;
                z = new_z;
            }
            if self.rotation_y != 0.0 {
                let cos_y = self.rotation_y.cos();
                let sin_y = self.rotation_y.sin();
                let new_x = x * cos_y + z * sin_y;
                let new_z = -x * sin_y + z * cos_y;
                x = new_x;
                z = new_z;
            }
            if self.rotation_z != 0.0 {
                let cos_z = self.rotation_z.cos();
                let sin_z = self.rotation_z.sin();
                let new_x = x * cos_z - y * sin_z;
                let new_y = x * sin_z + y * cos_z;
                x = new_x;
                y = new_y;
            }
        }

        let ds = self.display_scale();
        let cw = self.width as f64;
        let ch = self.height as f64;

        let mut sx = ((x - self.follow_x) / AU * ds - self.pan_x * ds + cw / 2.0) as f32;
        let mut sy = ((y - self.follow_y) / AU * ds - self.pan_y * ds + ch / 2.0) as f32;

        if self.use_3d {
            sx -= (z / AU * ds * 0.5) as f32;
            sy -= (z / AU * ds * 0.8) as f32;
        }

        (sx, sy)
    }

    /// Orthographic projection matrix mapping pixel coords [0,w]x[0,h] to NDC [-1,1].
    /// wgpu NDC: x[-1,1], y[-1,1] (y up), z[0,1].
    /// We flip y so that y=0 is top of screen (matching Fyne convention).
    pub fn ortho_matrix(&self) -> [[f32; 4]; 4] {
        let w = self.width as f32;
        let h = self.height as f32;
        [
            [2.0 / w, 0.0, 0.0, 0.0],
            [0.0, -2.0 / h, 0.0, 0.0],
            [0.0, 0.0, 1.0, 0.0],
            [-1.0, 1.0, 0.0, 1.0],
        ]
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_world_to_screen_origin() {
        let cam = Camera::new(800, 600);
        let (sx, sy) = cam.world_to_screen(0.0, 0.0, 0.0);
        assert!((sx - 400.0).abs() < 0.01);
        assert!((sy - 300.0).abs() < 0.01);
    }

    #[test]
    fn test_world_to_screen_1au() {
        let cam = Camera::new(800, 600);
        // 1 AU on x-axis should be at center + display_scale pixels
        let (sx, _sy) = cam.world_to_screen(AU, 0.0, 0.0);
        assert!((sx - 500.0).abs() < 0.01); // 400 + 100
    }
}
