use crate::constants::{G, C};
use crate::vec3::Vec3;

/// Computes the general relativistic perihelion precession correction.
/// Formula: a_GR = (3G²M²)/(c²r³L) * (L × r)
pub fn calculate_gr_correction(body_pos: Vec3, body_vel: Vec3, sun_mass: f64, distance_to_sun: f64) -> Vec3 {
    let l_vec = body_pos.cross(body_vel);
    let l_mag = l_vec.magnitude();

    if l_mag < 1e-10 {
        return Vec3::default();
    }

    let factor = 3.0 * G * G * sun_mass * sun_mass
        / (C * C * distance_to_sun * distance_to_sun * distance_to_sun * l_mag);

    l_vec.cross(body_pos).mul(factor)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_gr_correction_nonzero_for_mercury() {
        // Mercury-like initial conditions
        let pos = Vec3::new(5.791e10, 0.0, 0.0);
        let vel = Vec3::new(0.0, 47362.0, 0.0);
        let sun_mass = 1.989e30;
        let dist = pos.magnitude();

        let corr = calculate_gr_correction(pos, vel, sun_mass, dist);
        assert!(corr.magnitude() > 0.0, "GR correction should be non-zero for Mercury");
    }

    #[test]
    fn test_gr_correction_zero_angular_momentum() {
        // Radial motion — L = 0
        let pos = Vec3::new(1e11, 0.0, 0.0);
        let vel = Vec3::new(1e4, 0.0, 0.0);
        let sun_mass = 1.989e30;
        let dist = pos.magnitude();

        let corr = calculate_gr_correction(pos, vel, sun_mass, dist);
        assert_eq!(corr.magnitude(), 0.0);
    }

    #[test]
    fn test_gr_correction_formula() {
        let pos = Vec3::new(5.791e10, 0.0, 0.0);
        let vel = Vec3::new(0.0, 47362.0, 0.0);
        let sun_mass = 1.989e30;
        let dist = pos.magnitude();

        let corr = calculate_gr_correction(pos, vel, sun_mass, dist);

        // Manual calculation
        let l_vec = pos.cross(vel);
        let l_mag = l_vec.magnitude();
        let factor = 3.0 * G * G * sun_mass * sun_mass
            / (C * C * dist * dist * dist * l_mag);
        let expected = l_vec.cross(pos).mul(factor);

        let rel_err = (corr.magnitude() - expected.magnitude()).abs() / expected.magnitude();
        assert!(rel_err < 1e-15, "formula mismatch: rel_err = {}", rel_err);
    }
}
