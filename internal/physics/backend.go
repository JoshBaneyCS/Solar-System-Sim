package physics

import "solar-system-sim/internal/math3d"

// PhysicsBackend defines the interface for swappable physics engines.
type PhysicsBackend interface {
	// Step advances the simulation by dt seconds.
	Step(dt float64)
	// GetState returns current positions and velocities for all bodies.
	GetState() (positions, velocities []math3d.Vec3)
	// SetConfig updates runtime physics configuration.
	SetConfig(sunMass float64, planetGravity bool, relativisticEffects bool)
	// Close releases any resources held by the backend.
	Close()
}
