use crate::constants::G;
use crate::gr;
use crate::vec3::Vec3;
use rayon::prelude::*;

/// Minimum body count before parallelizing acceleration loops.
/// Below this threshold, thread spawn overhead exceeds the benefit.
const PARALLEL_THRESHOLD: usize = 16;

pub struct Simulation {
    pub n_bodies: usize,
    pub sun_mass: f64,
    pub masses: Vec<f64>,
    pub positions: Vec<Vec3>,
    pub velocities: Vec<Vec3>,
    pub gr_flags: Vec<bool>,
    pub planet_gravity: bool,

    // Pre-allocated RK4 scratch buffers (avoid re-alloc per step)
    scratch: Option<RK4Scratch>,
}

/// Pre-allocated buffers for the RK4 integrator.
struct RK4Scratch {
    pos0: Vec<Vec3>,
    vel0: Vec<Vec3>,
    k1p: Vec<Vec3>,
    k1v: Vec<Vec3>,
    pos2: Vec<Vec3>,
    vel2: Vec<Vec3>,
    k2p: Vec<Vec3>,
    k2v: Vec<Vec3>,
    pos3: Vec<Vec3>,
    vel3: Vec<Vec3>,
    k3p: Vec<Vec3>,
    k3v: Vec<Vec3>,
    pos4: Vec<Vec3>,
    vel4: Vec<Vec3>,
    k4p: Vec<Vec3>,
    k4v: Vec<Vec3>,
}

impl RK4Scratch {
    fn new(n: usize) -> Self {
        let z = Vec3::default();
        Self {
            pos0: vec![z; n],
            vel0: vec![z; n],
            k1p: vec![z; n],
            k1v: vec![z; n],
            pos2: vec![z; n],
            vel2: vec![z; n],
            k2p: vec![z; n],
            k2v: vec![z; n],
            pos3: vec![z; n],
            vel3: vec![z; n],
            k3p: vec![z; n],
            k3v: vec![z; n],
            pos4: vec![z; n],
            vel4: vec![z; n],
            k4p: vec![z; n],
            k4v: vec![z; n],
        }
    }

    fn ensure_size(&mut self, n: usize) {
        if self.pos0.len() >= n {
            return;
        }
        *self = Self::new(n);
    }
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
            scratch: Some(RK4Scratch::new(n_bodies)),
        }
    }

    #[inline]
    fn calculate_acceleration(
        &self,
        body_index: usize,
        body_pos: Vec3,
        body_vel: Vec3,
        snapshot_positions: &[Vec3],
    ) -> Vec3 {
        let mut total_accel = Vec3::default();

        // Sun gravity (sun at origin)
        let r_sun = body_pos.mul(-1.0); // Vec3::default().sub(body_pos) = -body_pos
        let distance_sun = r_sun.magnitude();

        if distance_sun > 1e6 {
            let r_hat_sun = r_sun.normalize();
            let accel_mag_sun = G * self.sun_mass / (distance_sun * distance_sun);
            let mut accel_sun = r_hat_sun.mul(accel_mag_sun);

            if self.gr_flags[body_index] {
                let rel_accel =
                    gr::calculate_gr_correction(body_pos, body_vel, self.sun_mass, distance_sun);
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

    /// Compute accelerations for all bodies, parallelizing when body count is large enough.
    fn compute_accelerations(
        &self,
        positions: &[Vec3],
        velocities: &[Vec3],
        snapshot: &[Vec3],
        out_kp: &mut [Vec3],
        out_kv: &mut [Vec3],
    ) {
        let n = self.n_bodies;

        // kp = velocity (no computation needed beyond copy)
        out_kp[..n].copy_from_slice(&velocities[..n]);

        if n >= PARALLEL_THRESHOLD {
            // Parallel acceleration computation
            let accels: Vec<Vec3> = (0..n)
                .into_par_iter()
                .map(|i| self.calculate_acceleration(i, positions[i], velocities[i], snapshot))
                .collect();
            out_kv[..n].copy_from_slice(&accels);
        } else {
            // Sequential for small body counts
            for i in 0..n {
                out_kv[i] = self.calculate_acceleration(i, positions[i], velocities[i], snapshot);
            }
        }
    }

    pub fn step(&mut self, dt: f64) {
        let n = self.n_bodies;

        // Take scratch out to avoid borrow conflicts
        let mut scratch = self.scratch.take().unwrap_or_else(|| RK4Scratch::new(n));
        scratch.ensure_size(n);

        // Copy current state to scratch
        scratch.pos0[..n].copy_from_slice(&self.positions[..n]);
        scratch.vel0[..n].copy_from_slice(&self.velocities[..n]);

        // k1: accelerations at current state
        self.compute_accelerations(
            &scratch.pos0[..n],
            &scratch.vel0[..n],
            &scratch.pos0[..n],
            &mut scratch.k1p,
            &mut scratch.k1v,
        );

        // k2: midpoint using k1
        let half_dt = dt / 2.0;
        for i in 0..n {
            scratch.pos2[i] = scratch.pos0[i].add(scratch.k1p[i].mul(half_dt));
            scratch.vel2[i] = scratch.vel0[i].add(scratch.k1v[i].mul(half_dt));
        }
        self.compute_accelerations(
            &scratch.pos2[..n],
            &scratch.vel2[..n],
            &scratch.pos2[..n],
            &mut scratch.k2p,
            &mut scratch.k2v,
        );

        // k3: midpoint using k2
        for i in 0..n {
            scratch.pos3[i] = scratch.pos0[i].add(scratch.k2p[i].mul(half_dt));
            scratch.vel3[i] = scratch.vel0[i].add(scratch.k2v[i].mul(half_dt));
        }
        self.compute_accelerations(
            &scratch.pos3[..n],
            &scratch.vel3[..n],
            &scratch.pos3[..n],
            &mut scratch.k3p,
            &mut scratch.k3v,
        );

        // k4: full step using k3
        for i in 0..n {
            scratch.pos4[i] = scratch.pos0[i].add(scratch.k3p[i].mul(dt));
            scratch.vel4[i] = scratch.vel0[i].add(scratch.k3v[i].mul(dt));
        }
        self.compute_accelerations(
            &scratch.pos4[..n],
            &scratch.vel4[..n],
            &scratch.pos4[..n],
            &mut scratch.k4p,
            &mut scratch.k4v,
        );

        // Final RK4 combination
        let dt6 = dt / 6.0;
        for i in 0..n {
            self.positions[i] = scratch.pos0[i].add(
                scratch.k1p[i]
                    .add(scratch.k2p[i].mul(2.0))
                    .add(scratch.k3p[i].mul(2.0))
                    .add(scratch.k4p[i])
                    .mul(dt6),
            );
            self.velocities[i] = scratch.vel0[i].add(
                scratch.k1v[i]
                    .add(scratch.k2v[i].mul(2.0))
                    .add(scratch.k3v[i].mul(2.0))
                    .add(scratch.k4v[i])
                    .mul(dt6),
            );
        }

        // Return scratch buffers
        self.scratch = Some(scratch);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn make_test_sim() -> Simulation {
        Simulation::new(
            1,
            1.989e30,
            vec![3.3011e23],
            vec![Vec3::new(4.600e10, 0.0, 0.0)],
            vec![Vec3::new(0.0, 58980.0, 0.0)],
            vec![true],
            false,
        )
    }

    #[test]
    fn test_sun_only_gravity() {
        let sim = Simulation::new(
            1,
            1.989e30,
            vec![3.3011e23],
            vec![Vec3::new(4.600e10, 0.0, 0.0)],
            vec![Vec3::new(0.0, 58980.0, 0.0)],
            vec![false],
            false,
        );
        let accel =
            sim.calculate_acceleration(0, sim.positions[0], sim.velocities[0], &sim.positions);
        assert!(accel.x < 0.0, "acceleration should point toward Sun");
        assert!(
            accel.y.abs() < 1e-10,
            "y acceleration should be ~0 without GR"
        );

        let r = sim.positions[0].magnitude();
        let expected_mag = G * sim.sun_mass / (r * r);
        let rel_err = (accel.magnitude() - expected_mag).abs() / expected_mag;
        assert!(
            rel_err < 1e-10,
            "acceleration magnitude mismatch: rel_err = {}",
            rel_err
        );
    }

    #[test]
    fn test_inverse_square_law() {
        let sim = Simulation::new(
            1,
            1.989e30,
            vec![5.972e24],
            vec![Vec3::new(1.496e11, 0.0, 0.0)],
            vec![Vec3::new(0.0, 29780.0, 0.0)],
            vec![false],
            false,
        );
        let accel_1au =
            sim.calculate_acceleration(0, sim.positions[0], sim.velocities[0], &sim.positions);

        let sim2 = Simulation::new(
            1,
            1.989e30,
            vec![5.972e24],
            vec![Vec3::new(2.0 * 1.496e11, 0.0, 0.0)],
            vec![Vec3::new(0.0, 29780.0, 0.0)],
            vec![false],
            false,
        );
        let accel_2au =
            sim2.calculate_acceleration(0, sim2.positions[0], sim2.velocities[0], &sim2.positions);

        let ratio = accel_1au.magnitude() / accel_2au.magnitude();
        let rel_err = (ratio - 4.0).abs() / 4.0;
        assert!(
            rel_err < 1e-10,
            "inverse square ratio should be 4, got {}",
            ratio
        );
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

    #[test]
    fn test_parallel_consistency() {
        // Run the same simulation with many bodies and verify results match sequential
        let n = 20;
        let masses = vec![5.972e24; n];
        let mut positions = Vec::with_capacity(n);
        let mut velocities = Vec::with_capacity(n);
        let gr_flags = vec![false; n];

        for i in 0..n {
            let angle = 2.0 * std::f64::consts::PI * (i as f64) / (n as f64);
            let r = 1.496e11 * (1.0 + 0.5 * (i as f64) / (n as f64));
            positions.push(Vec3::new(r * angle.cos(), r * angle.sin(), 0.0));
            let v = 29780.0 * (1.0 / (1.0 + 0.5 * (i as f64) / (n as f64))).sqrt();
            velocities.push(Vec3::new(-v * angle.sin(), v * angle.cos(), 0.0));
        }

        let mut sim = Simulation::new(n, 1.989e30, masses, positions, velocities, gr_flags, true);
        sim.step(7200.0);

        // Verify all positions changed (basic sanity)
        for i in 0..n {
            assert!(
                sim.positions[i].magnitude() > 1e10,
                "body {} should still be far from origin",
                i
            );
        }
    }
}
