package main

import (
	"testing"
)

func TestSunOnlyGravity(t *testing.T) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = false
	sim.RelativisticEffects = false

	// Place a test body at 1 AU on the x-axis
	bodyPos := Vec3{AU, 0, 0}
	bodyVel := Vec3{0, 29784, 0} // approximate Earth orbital velocity
	bodyMass := 5.972e24

	states := make([]BodyState, len(sim.Planets))
	for i, p := range sim.Planets {
		states[i] = BodyState{Position: p.Position, Velocity: p.Velocity}
	}

	accel := sim.calculateAccelerationWithSnapshot(0, bodyPos, bodyVel, bodyMass, "TestBody", states)

	// Expected: GM/r² pointing toward Sun (-x direction)
	expectedMag := G * sim.SunMass / (AU * AU)
	gotMag := accel.Magnitude()

	assertRelativeError(t, gotMag, expectedMag, 1e-8, "acceleration magnitude")

	// Should point in -x direction
	if accel.X >= 0 {
		t.Errorf("expected negative X acceleration (toward Sun), got %e", accel.X)
	}
	assertFloat64Near(t, accel.Y, 0, expectedMag*1e-10, "Y component should be ~0")
	assertFloat64Near(t, accel.Z, 0, expectedMag*1e-10, "Z component should be ~0")
}

func TestInverseSquareLaw(t *testing.T) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = false
	sim.RelativisticEffects = false

	states := make([]BodyState, len(sim.Planets))
	for i, p := range sim.Planets {
		states[i] = BodyState{Position: p.Position, Velocity: p.Velocity}
	}

	// Acceleration at 1 AU
	accel1 := sim.calculateAccelerationWithSnapshot(0, Vec3{AU, 0, 0}, Vec3{0, 0, 0}, 1e24, "Test", states)
	// Acceleration at 2 AU
	accel2 := sim.calculateAccelerationWithSnapshot(0, Vec3{2 * AU, 0, 0}, Vec3{0, 0, 0}, 1e24, "Test", states)

	mag1 := accel1.Magnitude()
	mag2 := accel2.Magnitude()

	// At 2x distance, acceleration should be 1/4
	ratio := mag1 / mag2
	assertRelativeError(t, ratio, 4.0, 1e-8, "inverse square ratio")
}

func TestNBodyGravity(t *testing.T) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = true
	sim.RelativisticEffects = false

	states := make([]BodyState, len(sim.Planets))
	for i, p := range sim.Planets {
		states[i] = BodyState{Position: p.Position, Velocity: p.Velocity}
	}

	// Get acceleration with N-body enabled
	accelNBody := sim.calculateAccelerationWithSnapshot(0, sim.Planets[0].Position, sim.Planets[0].Velocity,
		sim.Planets[0].Mass, sim.Planets[0].Name, states)

	// Get acceleration with N-body disabled
	sim.PlanetGravityEnabled = false
	accelSunOnly := sim.calculateAccelerationWithSnapshot(0, sim.Planets[0].Position, sim.Planets[0].Velocity,
		sim.Planets[0].Mass, sim.Planets[0].Name, states)

	// N-body acceleration should differ from Sun-only
	diff := accelNBody.Sub(accelSunOnly)
	if diff.Magnitude() < 1e-20 {
		t.Error("N-body contribution should be non-zero")
	}

	// But Sun dominates: N-body perturbation should be small relative to Sun gravity
	ratio := diff.Magnitude() / accelSunOnly.Magnitude()
	if ratio > 0.01 {
		t.Errorf("N-body perturbation too large relative to Sun: ratio = %e", ratio)
	}
}

func TestGravitySymmetry(t *testing.T) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = false
	sim.RelativisticEffects = false

	states := make([]BodyState, len(sim.Planets))
	for i, p := range sim.Planets {
		states[i] = BodyState{Position: p.Position, Velocity: p.Velocity}
	}

	// Acceleration at (AU, 0, 0) and (-AU, 0, 0) should have same magnitude
	accelPos := sim.calculateAccelerationWithSnapshot(0, Vec3{AU, 0, 0}, Vec3{}, 1e24, "Test", states)
	accelNeg := sim.calculateAccelerationWithSnapshot(0, Vec3{-AU, 0, 0}, Vec3{}, 1e24, "Test", states)

	assertRelativeError(t, accelPos.Magnitude(), accelNeg.Magnitude(), 1e-10, "symmetric magnitude")
}
