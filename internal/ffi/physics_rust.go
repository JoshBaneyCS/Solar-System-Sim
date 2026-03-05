//go:build rust_physics

package ffi

/*
#cgo LDFLAGS: -L${SRCDIR}/../../crates/physics_core/target/release -lphysics_core -Wl,-rpath,${SRCDIR}/../../crates/physics_core/target/release
#include "../../crates/physics_core/include/physics_core.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// RustSimulator wraps the Rust physics_core simulation handle.
type RustSimulator struct {
	handle *C.PhysicsSim
	n      int
}

// NewRustSimulator creates a new Rust-backed physics simulation.
func NewRustSimulator(sunMass float64, masses []float64, state []float64, grFlags []uint8, planetGravity bool) *RustSimulator {
	n := len(masses)
	pg := C.uint8_t(0)
	if planetGravity {
		pg = 1
	}

	handle := C.physics_create(
		C.uint32_t(n),
		C.double(sunMass),
		(*C.double)(unsafe.Pointer(&masses[0])),
		(*C.double)(unsafe.Pointer(&state[0])),
		(*C.uint8_t)(unsafe.Pointer(&grFlags[0])),
		pg,
	)

	return &RustSimulator{handle: handle, n: n}
}

// Step advances the simulation by dt seconds.
func (rs *RustSimulator) Step(dt float64) {
	C.physics_step(rs.handle, C.double(dt))
}

// GetState reads current positions and velocities into a flat float64 slice.
// Layout: [px0, py0, pz0, vx0, vy0, vz0, px1, py1, ...] (n*6 elements)
func (rs *RustSimulator) GetState() []float64 {
	buf := make([]float64, rs.n*6)
	C.physics_get_state(rs.handle, (*C.double)(unsafe.Pointer(&buf[0])))
	return buf
}

// SetConfig updates runtime configuration.
func (rs *RustSimulator) SetConfig(sunMass float64, planetGravity bool, grFlags []uint8) {
	pg := C.uint8_t(0)
	if planetGravity {
		pg = 1
	}
	C.physics_set_config(
		rs.handle,
		C.double(sunMass),
		pg,
		(*C.uint8_t)(unsafe.Pointer(&grFlags[0])),
	)
}

// Free releases the Rust simulation handle.
func (rs *RustSimulator) Free() {
	if rs.handle != nil {
		C.physics_free(rs.handle)
		rs.handle = nil
	}
}

// BodyCount returns the number of bodies in the simulation.
func (rs *RustSimulator) BodyCount() int {
	return int(C.physics_body_count(rs.handle))
}
