use crate::sim::Simulation;
use crate::vec3::Vec3;
use std::slice;

#[no_mangle]
pub unsafe extern "C" fn physics_create(
    n_bodies: u32,
    sun_mass: f64,
    masses: *const f64,
    state: *const f64,
    gr_flags: *const u8,
    planet_gravity: u8,
) -> *mut Simulation {
    let n = n_bodies as usize;
    let masses_slice = slice::from_raw_parts(masses, n);
    let state_slice = slice::from_raw_parts(state, n * 6);
    let gr_slice = slice::from_raw_parts(gr_flags, n);

    let mut positions = Vec::with_capacity(n);
    let mut velocities = Vec::with_capacity(n);

    for i in 0..n {
        let base = i * 6;
        positions.push(Vec3::new(
            state_slice[base],
            state_slice[base + 1],
            state_slice[base + 2],
        ));
        velocities.push(Vec3::new(
            state_slice[base + 3],
            state_slice[base + 4],
            state_slice[base + 5],
        ));
    }

    let sim = Simulation::new(
        n,
        sun_mass,
        masses_slice.to_vec(),
        positions,
        velocities,
        gr_slice.iter().map(|&f| f != 0).collect(),
        planet_gravity != 0,
    );

    Box::into_raw(Box::new(sim))
}

#[no_mangle]
pub unsafe extern "C" fn physics_step(handle: *mut Simulation, dt: f64) {
    if handle.is_null() {
        return;
    }
    let sim = &mut *handle;
    sim.step(dt);
}

#[no_mangle]
pub unsafe extern "C" fn physics_get_state(handle: *const Simulation, out_state: *mut f64) {
    if handle.is_null() || out_state.is_null() {
        return;
    }
    let sim = &*handle;
    let out = slice::from_raw_parts_mut(out_state, sim.n_bodies * 6);

    for i in 0..sim.n_bodies {
        let base = i * 6;
        out[base] = sim.positions[i].x;
        out[base + 1] = sim.positions[i].y;
        out[base + 2] = sim.positions[i].z;
        out[base + 3] = sim.velocities[i].x;
        out[base + 4] = sim.velocities[i].y;
        out[base + 5] = sim.velocities[i].z;
    }
}

#[no_mangle]
pub unsafe extern "C" fn physics_set_config(
    handle: *mut Simulation,
    sun_mass: f64,
    planet_gravity: u8,
    gr_flags: *const u8,
) {
    if handle.is_null() {
        return;
    }
    let sim = &mut *handle;
    sim.sun_mass = sun_mass;
    sim.planet_gravity = planet_gravity != 0;

    if !gr_flags.is_null() {
        let flags = slice::from_raw_parts(gr_flags, sim.n_bodies);
        for i in 0..sim.n_bodies {
            sim.gr_flags[i] = flags[i] != 0;
        }
    }
}

#[no_mangle]
pub unsafe extern "C" fn physics_free(handle: *mut Simulation) {
    if !handle.is_null() {
        drop(Box::from_raw(handle));
    }
}

#[no_mangle]
pub unsafe extern "C" fn physics_body_count(handle: *const Simulation) -> u32 {
    if handle.is_null() {
        return 0;
    }
    (*handle).n_bodies as u32
}
