//go:build rust_physics

package physics

import (
	"solar-system-sim/internal/ffi"
	"solar-system-sim/internal/math3d"
)

// RustBackend wraps the Rust physics_core FFI for use as a PhysicsBackend.
type RustBackend struct {
	sim     *ffi.RustSimulator
	n       int
	grFlags []uint8
}

// NewRustBackend creates a Rust physics backend from the current simulator state.
func NewRustBackend(s *Simulator) *RustBackend {
	n := len(s.Planets)
	masses := make([]float64, n)
	state := make([]float64, n*6)
	grFlags := make([]uint8, n)

	for i, p := range s.Planets {
		masses[i] = p.Mass
		base := i * 6
		state[base] = p.Position.X
		state[base+1] = p.Position.Y
		state[base+2] = p.Position.Z
		state[base+3] = p.Velocity.X
		state[base+4] = p.Velocity.Y
		state[base+5] = p.Velocity.Z

		// GR is applied only to Mercury (index 0) when relativistic effects are on
		if s.RelativisticEffects && p.Name == "Mercury" {
			grFlags[i] = 1
		}
	}

	rs := ffi.NewRustSimulator(s.SunMass, masses, state, grFlags, s.PlanetGravityEnabled)
	return &RustBackend{sim: rs, n: n, grFlags: grFlags}
}

func (rb *RustBackend) Step(dt float64) {
	rb.sim.Step(dt)
}

func (rb *RustBackend) GetState() (positions, velocities []math3d.Vec3) {
	state := rb.sim.GetState()
	positions = make([]math3d.Vec3, rb.n)
	velocities = make([]math3d.Vec3, rb.n)

	for i := 0; i < rb.n; i++ {
		base := i * 6
		positions[i] = math3d.Vec3{X: state[base], Y: state[base+1], Z: state[base+2]}
		velocities[i] = math3d.Vec3{X: state[base+3], Y: state[base+4], Z: state[base+5]}
	}
	return
}

func (rb *RustBackend) SetConfig(sunMass float64, planetGravity bool, relativisticEffects bool) {
	// Update GR flags based on relativistic effects toggle
	for i := range rb.grFlags {
		rb.grFlags[i] = 0
	}
	if relativisticEffects && rb.n > 0 {
		rb.grFlags[0] = 1 // Mercury is index 0
	}
	rb.sim.SetConfig(sunMass, planetGravity, rb.grFlags)
}

func (rb *RustBackend) Close() {
	rb.sim.Free()
}
