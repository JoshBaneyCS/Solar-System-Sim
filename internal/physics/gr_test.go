package physics

import (
	"math"
	"testing"

	"solar-system-sim/pkg/constants"
)

func TestGRCorrectionNonZeroForMercury(t *testing.T) {
	sim := NewSimulator()

	states := make([]BodyState, len(sim.Planets))
	for i, p := range sim.Planets {
		states[i] = BodyState{Position: p.Position, Velocity: p.Velocity}
	}

	sim.RelativisticEffects = true
	accelGR := sim.CalculateAccelerationWithSnapshot(0, sim.Planets[0].Position, sim.Planets[0].Velocity,
		sim.Planets[0].Mass, "Mercury", states)

	sim.RelativisticEffects = false
	accelNewton := sim.CalculateAccelerationWithSnapshot(0, sim.Planets[0].Position, sim.Planets[0].Velocity,
		sim.Planets[0].Mass, "Mercury", states)

	grCorrection := accelGR.Sub(accelNewton)

	if grCorrection.Magnitude() < 1e-20 {
		t.Fatal("GR correction should be non-zero for Mercury")
	}

	t.Logf("GR correction magnitude: %e m/s²", grCorrection.Magnitude())
	t.Logf("Newtonian acceleration:  %e m/s²", accelNewton.Magnitude())

	ratio := grCorrection.Magnitude() / accelNewton.Magnitude()
	t.Logf("GR/Newtonian ratio: %e", ratio)

	// 1PN correction for Mercury should be ~1e-7 of Newtonian
	if ratio > 1e-5 || ratio < 1e-10 {
		t.Errorf("GR/Newtonian ratio %e out of expected range [1e-10, 1e-5]", ratio)
	}
}

func TestGRCorrectionZeroForOtherPlanets(t *testing.T) {
	sim := NewSimulator()

	states := make([]BodyState, len(sim.Planets))
	for i, p := range sim.Planets {
		states[i] = BodyState{Position: p.Position, Velocity: p.Velocity}
	}

	for _, idx := range []int{1, 2, 3} {
		name := sim.Planets[idx].Name
		t.Run(name, func(t *testing.T) {
			sim.RelativisticEffects = true
			accelGR := sim.CalculateAccelerationWithSnapshot(idx, sim.Planets[idx].Position, sim.Planets[idx].Velocity,
				sim.Planets[idx].Mass, name, states)

			sim.RelativisticEffects = false
			accelNewton := sim.CalculateAccelerationWithSnapshot(idx, sim.Planets[idx].Position, sim.Planets[idx].Velocity,
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

	mercury := sim.Planets[0]
	pos := mercury.Position
	vel := mercury.Velocity

	distSun := pos.Magnitude()
	c := constants.C
	GM := constants.G * sim.SunMass

	// Compute expected 1PN correction manually:
	// a_GR = GM/(c²r³) * [(4GM/r - v²)r + 4(r·v)v]
	v2 := vel.Dot(vel)
	rdotv := pos.Dot(vel)
	coeff := GM / (c * c * distSun * distSun * distSun)
	grAccel := pos.Mul(4*GM/distSun - v2).Add(vel.Mul(4 * rdotv)).Mul(coeff)

	states := make([]BodyState, len(sim.Planets))
	for i, p := range sim.Planets {
		states[i] = BodyState{Position: p.Position, Velocity: p.Velocity}
	}

	sim.RelativisticEffects = true
	sim.PlanetGravityEnabled = false
	accelGR := sim.CalculateAccelerationWithSnapshot(0, pos, vel, mercury.Mass, "Mercury", states)

	sim.RelativisticEffects = false
	accelNewton := sim.CalculateAccelerationWithSnapshot(0, pos, vel, mercury.Mass, "Mercury", states)

	codeGR := accelGR.Sub(accelNewton)

	assertRelativeError(t, codeGR.X, grAccel.X, 1e-8, "GR X component")
	assertRelativeError(t, codeGR.Y, grAccel.Y, 1e-8, "GR Y component")
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
	accelGR := sim.CalculateAccelerationWithSnapshot(0, pos, vel, mercury.Mass, "Mercury", states)

	sim.RelativisticEffects = false
	accelNewton := sim.CalculateAccelerationWithSnapshot(0, pos, vel, mercury.Mass, "Mercury", states)

	grCorrection := accelGR.Sub(accelNewton)

	// 1PN correction lies in the orbital plane (spanned by r and v),
	// so it should be perpendicular to the angular momentum vector L = r × v
	L := pos.Cross(vel)
	dot := grCorrection.Dot(L)
	if math.Abs(dot) > grCorrection.Magnitude()*L.Magnitude()*1e-8 {
		t.Errorf("GR correction should be perpendicular to angular momentum vector, dot = %e", dot)
	}
}
