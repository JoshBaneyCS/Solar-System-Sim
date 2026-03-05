#ifndef PHYSICS_CORE_H
#define PHYSICS_CORE_H

#include <stdint.h>

typedef struct PhysicsSim PhysicsSim;

// Create a simulation with n_bodies planets.
// masses: array of n_bodies doubles (kg)
// state: array of n_bodies*6 doubles [px,py,pz,vx,vy,vz] per body
// gr_flags: array of n_bodies uint8 (1 = apply GR correction, 0 = skip)
// planet_gravity: 1 = enable N-body, 0 = sun-only
PhysicsSim* physics_create(uint32_t n_bodies, double sun_mass,
    const double* masses, const double* state,
    const uint8_t* gr_flags, uint8_t planet_gravity);

// Advance simulation by dt seconds (one RK4 step).
void physics_step(PhysicsSim* handle, double dt);

// Read current state into caller-owned buffer.
// out_state must point to n_bodies*6 doubles.
void physics_get_state(const PhysicsSim* handle, double* out_state);

// Update simulation configuration at runtime.
void physics_set_config(PhysicsSim* handle, double sun_mass,
    uint8_t planet_gravity, const uint8_t* gr_flags);

// Free the simulation handle and all owned memory.
void physics_free(PhysicsSim* handle);

// Get the number of bodies.
uint32_t physics_body_count(const PhysicsSim* handle);

#endif
