package main

import (
	"math"
	"testing"
)

func TestGRCorrectionNonZeroForMercury(t *testing.T) {
	sim := NewSimulator()

	states := make([]BodyState, len(sim.Planets))
	for i, p := range sim.Planets {
		states[i] = BodyState{Position: p.Position, Velocity: p.Velocity}
	}

	// Mercury is planet index 0
	sim.RelativisticEffects = true
	accelGR := sim.calculateAccelerationWithSnapshot(0, sim.Planets[0].Position, sim.Planets[0].Velocity,
		sim.Planets[0].Mass, "Mercury", states)

	sim.RelativisticEffects = false
	accelNewton := sim.calculateAccelerationWithSnapshot(0, sim.Planets[0].Position, sim.Planets[0].Velocity,
		sim.Planets[0].Mass, "Mercury", states)

	grCorrection := accelGR.Sub(accelNewton)

	if grCorrection.Magnitude() < 1e-20 {
		t.Fatal("GR correction should be non-zero for Mercury")
	}

	// Verify the GR correction is non-trivial
	t.Logf("GR correction magnitude: %e m/s²", grCorrection.Magnitude())
	t.Logf("Newtonian acceleration:  %e m/s²", accelNewton.Magnitude())
	t.Logf("GR/Newtonian ratio: %e", grCorrection.Magnitude()/accelNewton.Magnitude())
}

func TestGRCorrectionZeroForOtherPlanets(t *testing.T) {
	sim := NewSimulator()

	states := make([]BodyState, len(sim.Planets))
	for i, p := range sim.Planets {
		states[i] = BodyState{Position: p.Position, Velocity: p.Velocity}
	}

	// Test Venus (index 1), Earth (index 2), Mars (index 3)
	for _, idx := range []int{1, 2, 3} {
		name := sim.Planets[idx].Name
		t.Run(name, func(t *testing.T) {
			sim.RelativisticEffects = true
			accelGR := sim.calculateAccelerationWithSnapshot(idx, sim.Planets[idx].Position, sim.Planets[idx].Velocity,
				sim.Planets[idx].Mass, name, states)

			sim.RelativisticEffects = false
			accelNewton := sim.calculateAccelerationWithSnapshot(idx, sim.Planets[idx].Position, sim.Planets[idx].Velocity,
				sim.Planets[idx].Mass, name, states)

			diff := accelGR.Sub(accelNewton)
			if diff.Magnitude() > 1e-30 {
				t.Errorf("GR correction should be zero for %s, got magnitude %e", name, diff.Magnitude())
			}
		})
	}
}

func TestGRCorrectionFormula(t *testing.T) {
	sim := NewSimulator()

	// Use Mercury's position and velocity
	mercury := sim.Planets[0]
	pos := mercury.Position
	vel := mercury.Velocity

	distSun := pos.Magnitude()
	c := 299792458.0

	// Manually compute GR correction: 3G²M²/(c²r³L) · (L × r)
	LVec := pos.Cross(vel)
	LMag := LVec.Magnitude()
	grAccel := LVec.Cross(pos).Mul(3 * G * G * sim.SunMass * sim.SunMass /
		(c * c * distSun * distSun * distSun * LMag))

	// Get correction from simulator
	states := make([]BodyState, len(sim.Planets))
	for i, p := range sim.Planets {
		states[i] = BodyState{Position: p.Position, Velocity: p.Velocity}
	}

	sim.RelativisticEffects = true
	sim.PlanetGravityEnabled = false
	accelGR := sim.calculateAccelerationWithSnapshot(0, pos, vel, mercury.Mass, "Mercury", states)

	sim.RelativisticEffects = false
	accelNewton := sim.calculateAccelerationWithSnapshot(0, pos, vel, mercury.Mass, "Mercury", states)

	codeGR := accelGR.Sub(accelNewton)

	// Compare manual calculation to code's result
	assertRelativeError(t, codeGR.X, grAccel.X, 1e-8, "GR X component")
	assertRelativeError(t, codeGR.Y, grAccel.Y, 1e-8, "GR Y component")

	// Z component may be very small, use absolute comparison
	assertFloat64Near(t, codeGR.Z, grAccel.Z, grAccel.Magnitude()*1e-8, "GR Z component")

	t.Logf("GR correction magnitude: %e m/s²", grAccel.Magnitude())
	t.Logf("Newtonian acceleration:  %e m/s²", accelNewton.Magnitude())
	t.Logf("Ratio: %e", grAccel.Magnitude()/accelNewton.Magnitude())
}

func TestGRCorrectionPerpendicular(t *testing.T) {
	sim := NewSimulator()

	mercury := sim.Planets[0]
	pos := mercury.Position
	vel := mercury.Velocity

	states := make([]BodyState, len(sim.Planets))
	for i, p := range sim.Planets {
		states[i] = BodyState{Position: p.Position, Velocity: p.Velocity}
	}

	sim.RelativisticEffects = true
	sim.PlanetGravityEnabled = false
	accelGR := sim.calculateAccelerationWithSnapshot(0, pos, vel, mercury.Mass, "Mercury", states)

	sim.RelativisticEffects = false
	accelNewton := sim.calculateAccelerationWithSnapshot(0, pos, vel, mercury.Mass, "Mercury", states)

	grCorrection := accelGR.Sub(accelNewton)

	// The GR correction is (L × r) direction, which is perpendicular to L
	// L = r × v, so (L × r) is in the orbital plane but perpendicular to r
	L := pos.Cross(vel)
	dot := grCorrection.Dot(L)
	if math.Abs(dot) > grCorrection.Magnitude()*L.Magnitude()*1e-8 {
		t.Errorf("GR correction should be perpendicular to angular momentum vector, dot = %e", dot)
	}
}
