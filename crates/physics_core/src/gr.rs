use crate::constants::{C, G};
use crate::vec3::Vec3;

/// Computes the first post-Newtonian (1PN) correction to acceleration.
///
/// Standard 1PN expression for a test particle in a Schwarzschild field:
///   a_GR = (GM/(c²r³)) * [(4GM/r - v²)r + 4(r·v)v]
///
/// This produces Mercury's perihelion precession of ~43 arcsec/century.
#[inline]
pub fn calculate_gr_correction(
    body_pos: Vec3,
    body_vel: Vec3,
    sun_mass: f64,
    distance_to_sun: f64,
) -> Vec3 {
    let gm = G * sun_mass;
    let r = distance_to_sun;
    let v2 = body_vel.dot(body_vel);
    let rdotv = body_pos.dot(body_vel);

    let coeff = gm / (C * C * r * r * r);

    // (4GM/r - v²) * r_vec + 4(r·v) * v_vec
    let term1 = body_pos.mul(4.0 * gm / r - v2);
    let term2 = body_vel.mul(4.0 * rdotv);

    term1.add(term2).mul(coeff)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_gr_correction_nonzero_for_mercury() {
        let pos = Vec3::new(5.791e10, 0.0, 0.0);
        let vel = Vec3::new(0.0, 47362.0, 0.0);
        let sun_mass = 1.989e30;
        let dist = pos.magnitude();

        let corr = calculate_gr_correction(pos, vel, sun_mass, dist);
        assert!(
            corr.magnitude() > 0.0,
            "GR correction should be non-zero for Mercury"
        );
    }

    #[test]
    fn test_gr_correction_zero_velocity() {
        // Body at rest — rdotv = 0, v² = 0, so a_GR = (GM/(c²r³)) * (4GM/r) * r_vec
        let pos = Vec3::new(1e11, 0.0, 0.0);
        let vel = Vec3::new(0.0, 0.0, 0.0);
        let sun_mass = 1.989e30;
        let dist = pos.magnitude();

        let corr = calculate_gr_correction(pos, vel, sun_mass, dist);
        // Should point in +x direction (same as position vector)
        assert!(
            corr.x > 0.0,
            "GR correction should be positive along position"
        );
        assert!(corr.y.abs() < 1e-30);
        assert!(corr.z.abs() < 1e-30);
    }

    #[test]
    fn test_gr_correction_formula_1pn() {
        let pos = Vec3::new(5.791e10, 0.0, 0.0);
        let vel = Vec3::new(0.0, 47362.0, 0.0);
        let sun_mass = 1.989e30;
        let dist = pos.magnitude();

        let corr = calculate_gr_correction(pos, vel, sun_mass, dist);

        // Manual calculation using 1PN formula
        let gm = G * sun_mass;
        let r = dist;
        let v2 = vel.dot(vel);
        let rdotv = pos.dot(vel);
        let coeff = gm / (C * C * r * r * r);
        let term1 = pos.mul(4.0 * gm / r - v2);
        let term2 = vel.mul(4.0 * rdotv);
        let expected = term1.add(term2).mul(coeff);

        let rel_err = (corr.magnitude() - expected.magnitude()).abs() / expected.magnitude();
        assert!(rel_err < 1e-15, "formula mismatch: rel_err = {}", rel_err);
    }
}
