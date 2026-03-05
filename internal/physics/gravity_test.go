package physics

import (
	"testing"

	"solar-system-sim/internal/math3d"
	"solar-system-sim/pkg/constants"
)

func TestSunOnlyGravity(t *testing.T) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = false
	sim.RelativisticEffects = false

	bodyPos := math3d.Vec3{X: constants.AU, Y: 0, Z: 0}
	bodyVel := math3d.Vec3{X: 0, Y: 29784, Z: 0}
	bodyMass := 5.972e24

	states := make([]BodyState, len(sim.Planets))
	for i, p := range sim.Planets {
		states[i] = BodyState{Position: p.Position, Velocity: p.Velocity}
	}

	accel := sim.CalculateAccelerationWithSnapshot(0, bodyPos, bodyVel, bodyMass, "TestBody", states)

	expectedMag := constants.G * sim.SunMass / (constants.AU * constants.AU)
	gotMag := accel.Magnitude()

	assertRelativeError(t, gotMag, expectedMag, 1e-8, "acceleration magnitude")

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

	accel1 := sim.CalculateAccelerationWithSnapshot(0, math3d.Vec3{X: constants.AU}, math3d.Vec3{}, 1e24, "Test", states)
	accel2 := sim.CalculateAccelerationWithSnapshot(0, math3d.Vec3{X: 2 * constants.AU}, math3d.Vec3{}, 1e24, "Test", states)

	mag1 := accel1.Magnitude()
	mag2 := accel2.Magnitude()

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

	accelNBody := sim.CalculateAccelerationWithSnapshot(0, sim.Planets[0].Position, sim.Planets[0].Velocity,
		sim.Planets[0].Mass, sim.Planets[0].Name, states)

	sim.PlanetGravityEnabled = false
	accelSunOnly := sim.CalculateAccelerationWithSnapshot(0, sim.Planets[0].Position, sim.Planets[0].Velocity,
		sim.Planets[0].Mass, sim.Planets[0].Name, states)

	diff := accelNBody.Sub(accelSunOnly)
	if diff.Magnitude() < 1e-20 {
		t.Error("N-body contribution should be non-zero")
	}

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

	accelPos := sim.CalculateAccelerationWithSnapshot(0, math3d.Vec3{X: constants.AU}, math3d.Vec3{}, 1e24, "Test", states)
	accelNeg := sim.CalculateAccelerationWithSnapshot(0, math3d.Vec3{X: -constants.AU}, math3d.Vec3{}, 1e24, "Test", states)

	assertRelativeError(t, accelPos.Magnitude(), accelNeg.Magnitude(), 1e-10, "symmetric magnitude")
}
