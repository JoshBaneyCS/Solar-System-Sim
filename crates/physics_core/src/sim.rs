use crate::constants::G;
use crate::gr;
use crate::vec3::Vec3;

pub struct Simulation {
    pub n_bodies: usize,
    pub sun_mass: f64,
    pub masses: Vec<f64>,
    pub positions: Vec<Vec3>,
    pub velocities: Vec<Vec3>,
    pub gr_flags: Vec<bool>,
    pub planet_gravity: bool,
}

impl Simulation {
    pub fn new(
        n_bodies: usize,
        sun_mass: f64,
        masses: Vec<f64>,
        positions: Vec<Vec3>,
        velocities: Vec<Vec3>,
        gr_flags: Vec<bool>,
        planet_gravity: bool,
    ) -> Self {
        Self {
            n_bodies,
            sun_mass,
            masses,
            positions,
            velocities,
            gr_flags,
            planet_gravity,
        }
    }

    fn calculate_acceleration(
        &self,
        body_index: usize,
        body_pos: Vec3,
        body_vel: Vec3,
        snapshot_positions: &[Vec3],
    ) -> Vec3 {
        let sun_pos = Vec3::default(); // Sun at origin
        let mut total_accel = Vec3::default();

        let r_sun = sun_pos.sub(body_pos);
        let distance_sun = r_sun.magnitude();

        if distance_sun > 1e6 {
            let r_hat_sun = r_sun.normalize();
            let accel_mag_sun = G * self.sun_mass / (distance_sun * distance_sun);
            let mut accel_sun = r_hat_sun.mul(accel_mag_sun);

            if self.gr_flags[body_index] {
                let rel_accel = gr::calculate_gr_correction(body_pos, body_vel, self.sun_mass, distance_sun);
                accel_sun = accel_sun.add(rel_accel);
            }

            total_accel = total_accel.add(accel_sun);
        }

        if self.planet_gravity {
            for i in 0..self.n_bodies {
                if i == body_index {
                    continue;
                }

                let other_pos = snapshot_positions[i];
                let other_mass = self.masses[i];

                let r_planet = other_pos.sub(body_pos);
                let distance_planet = r_planet.magnitude();

                if distance_planet > 1e6 {
                    let r_hat_planet = r_planet.normalize();
                    let accel_mag_planet = G * other_mass / (distance_planet * distance_planet);
                    let accel_planet = r_hat_planet.mul(accel_mag_planet);
                    total_accel = total_accel.add(accel_planet);
                }
            }
        }

        total_accel
    }

    pub fn step(&mut self, dt: f64) {
        let n = self.n_bodies;

        let pos0: Vec<Vec3> = self.positions.clone();
        let vel0: Vec<Vec3> = self.velocities.clone();

        // k1
        let mut k1p = vec![Vec3::default(); n];
        let mut k1v = vec![Vec3::default(); n];
        for i in 0..n {
            k1p[i] = vel0[i];
            k1v[i] = self.calculate_acceleration(i, pos0[i], vel0[i], &pos0);
        }

        // k2
        let mut pos2 = vec![Vec3::default(); n];
        let mut vel2 = vec![Vec3::default(); n];
        let mut k2p = vec![Vec3::default(); n];
        let mut k2v = vec![Vec3::default(); n];
        for i in 0..n {
            pos2[i] = pos0[i].add(k1p[i].mul(dt / 2.0));
            vel2[i] = vel0[i].add(k1v[i].mul(dt / 2.0));
        }
        for i in 0..n {
            k2p[i] = vel2[i];
            k2v[i] = self.calculate_acceleration(i, pos2[i], vel2[i], &pos2);
        }

        // k3
        let mut pos3 = vec![Vec3::default(); n];
        let mut vel3 = vec![Vec3::default(); n];
        let mut k3p = vec![Vec3::default(); n];
        let mut k3v = vec![Vec3::default(); n];
        for i in 0..n {
            pos3[i] = pos0[i].add(k2p[i].mul(dt / 2.0));
            vel3[i] = vel0[i].add(k2v[i].mul(dt / 2.0));
        }
        for i in 0..n {
            k3p[i] = vel3[i];
            k3v[i] = self.calculate_acceleration(i, pos3[i], vel3[i], &pos3);
        }

        // k4
        let mut pos4 = vec![Vec3::default(); n];
        let mut vel4 = vec![Vec3::default(); n];
        let mut k4p = vec![Vec3::default(); n];
        let mut k4v = vec![Vec3::default(); n];
        for i in 0..n {
            pos4[i] = pos0[i].add(k3p[i].mul(dt));
            vel4[i] = vel0[i].add(k3v[i].mul(dt));
        }
        for i in 0..n {
            k4p[i] = vel4[i];
            k4v[i] = self.calculate_acceleration(i, pos4[i], vel4[i], &pos4);
        }

        // Final combination
        for i in 0..n {
            self.positions[i] = pos0[i].add(
                k1p[i]
                    .add(k2p[i].mul(2.0))
                    .add(k3p[i].mul(2.0))
                    .add(k4p[i])
                    .mul(dt / 6.0),
            );
            self.velocities[i] = vel0[i].add(
                k1v[i]
                    .add(k2v[i].mul(2.0))
                    .add(k3v[i].mul(2.0))
                    .add(k4v[i])
                    .mul(dt / 6.0),
            );
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn make_test_sim() -> Simulation {
        // Use the exact initial state from Go's NewSimulator()
        // Mercury only, with GR enabled, planet gravity disabled
        Simulation::new(
            1,
            1.989e30,
            vec![3.3011e23],
            vec![Vec3::new(4.600e10, 0.0, 0.0)], // ~perihelion
            vec![Vec3::new(0.0, 58980.0, 0.0)],   // ~perihelion velocity
            vec![true],
            false,
        )
    }

    #[test]
    fn test_sun_only_gravity() {
        // Test without GR to isolate Newtonian gravity
        let sim = Simulation::new(
            1,
            1.989e30,
            vec![3.3011e23],
            vec![Vec3::new(4.600e10, 0.0, 0.0)],
            vec![Vec3::new(0.0, 58980.0, 0.0)],
            vec![false], // no GR
            false,
        );
        let accel = sim.calculate_acceleration(
            0,
            sim.positions[0],
            sim.velocities[0],
            &sim.positions,
        );
        // Should point toward Sun (negative x)
        assert!(accel.x < 0.0, "acceleration should point toward Sun");
        assert!(accel.y.abs() < 1e-10, "y acceleration should be ~0 without GR");

        // Check magnitude: GM/r^2
        let r = sim.positions[0].magnitude();
        let expected_mag = G * sim.sun_mass / (r * r);
        let rel_err = (accel.magnitude() - expected_mag).abs() / expected_mag;
        assert!(rel_err < 1e-10, "acceleration magnitude mismatch: rel_err = {}", rel_err);
    }

    #[test]
    fn test_inverse_square_law() {
        let sim = Simulation::new(
            1,
            1.989e30,
            vec![5.972e24],
            vec![Vec3::new(1.496e11, 0.0, 0.0)], // 1 AU
            vec![Vec3::new(0.0, 29780.0, 0.0)],
            vec![false],
            false,
        );
        let accel_1au = sim.calculate_acceleration(0, sim.positions[0], sim.velocities[0], &sim.positions);

        let sim2 = Simulation::new(
            1,
            1.989e30,
            vec![5.972e24],
            vec![Vec3::new(2.0 * 1.496e11, 0.0, 0.0)], // 2 AU
            vec![Vec3::new(0.0, 29780.0, 0.0)],
            vec![false],
            false,
        );
        let accel_2au = sim2.calculate_acceleration(0, sim2.positions[0], sim2.velocities[0], &sim2.positions);

        let ratio = accel_1au.magnitude() / accel_2au.magnitude();
        let rel_err = (ratio - 4.0).abs() / 4.0;
        assert!(rel_err < 1e-10, "inverse square ratio should be 4, got {}", ratio);
    }

    #[test]
    fn test_step_changes_state() {
        let mut sim = make_test_sim();
        let pos_before = sim.positions[0];
        sim.step(7200.0);
        let pos_after = sim.positions[0];
        assert!(
            (pos_after.x - pos_before.x).abs() > 0.0 || (pos_after.y - pos_before.y).abs() > 0.0,
            "step should change position"
        );
    }

    #[test]
    fn test_energy_conservation() {
        // Sun-only, no GR, single planet
        let mut sim = Simulation::new(
            1,
            1.989e30,
            vec![5.972e24],
            vec![Vec3::new(1.496e11, 0.0, 0.0)],
            vec![Vec3::new(0.0, 29780.0, 0.0)],
            vec![false],
            false,
        );

        let compute_energy = |s: &Simulation| -> f64 {
            let v = s.velocities[0].magnitude();
            let r = s.positions[0].magnitude();
            0.5 * s.masses[0] * v * v - G * s.sun_mass * s.masses[0] / r
        };

        let e0 = compute_energy(&sim);
        for _ in 0..1000 {
            sim.step(7200.0);
        }
        let e1 = compute_energy(&sim);

        let rel_drift = ((e1 - e0) / e0).abs();
        assert!(
            rel_drift < 1e-6,
            "energy conservation violated: rel_drift = {}",
            rel_drift
        );
    }
}
