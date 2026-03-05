package physics

import (
	"math"
	"testing"

	"solar-system-sim/internal/math3d"
	"solar-system-sim/pkg/constants"
)

func computeTotalEnergy(sim *Simulator) float64 {
	totalE := 0.0
	for _, p := range sim.Planets {
		v := p.Velocity.Magnitude()
		r := p.Position.Sub(sim.Sun.Position).Magnitude()

		kinetic := 0.5 * p.Mass * v * v
		potential := -constants.G * sim.SunMass * p.Mass / r

		totalE += kinetic + potential
	}
	return totalE
}

func computeTotalAngularMomentum(sim *Simulator) math3d.Vec3 {
	totalL := math3d.Vec3{}
	for _, p := range sim.Planets {
		r := p.Position.Sub(sim.Sun.Position)
		L := r.Cross(p.Velocity).Mul(p.Mass)
		totalL = totalL.Add(L)
	}
	return totalL
}

func TestEnergyConservation(t *testing.T) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = false
	sim.RelativisticEffects = false

	E0 := computeTotalEnergy(sim)

	for i := 0; i < 1000; i++ {
		sim.Step(constants.BaseTimeStep)
	}

	E1 := computeTotalEnergy(sim)

	relDrift := math.Abs((E1 - E0) / E0)
	t.Logf("Energy: initial=%e, final=%e, relative drift=%e", E0, E1, relDrift)

	if relDrift > 1e-6 {
		t.Errorf("energy conservation violated: relative drift %e > 1e-6", relDrift)
	}
}

func TestAngularMomentumConservation(t *testing.T) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = false
	sim.RelativisticEffects = false

	L0 := computeTotalAngularMomentum(sim)

	for i := 0; i < 1000; i++ {
		sim.Step(constants.BaseTimeStep)
	}

	L1 := computeTotalAngularMomentum(sim)

	for _, comp := range []struct {
		name string
		v0   float64
		v1   float64
	}{
		{"Lx", L0.X, L1.X},
		{"Ly", L0.Y, L1.Y},
		{"Lz", L0.Z, L1.Z},
	} {
		if comp.v0 == 0 {
			continue
		}
		relDrift := math.Abs((comp.v1 - comp.v0) / comp.v0)
		if relDrift > 1e-6 {
			t.Errorf("%s conservation violated: relative drift %e > 1e-6", comp.name, relDrift)
		}
	}

	t.Logf("L0 magnitude: %e", L0.Magnitude())
	t.Logf("L1 magnitude: %e", L1.Magnitude())
	t.Logf("Relative change: %e", math.Abs(L1.Magnitude()-L0.Magnitude())/L0.Magnitude())
}

func TestEarthOrbitalPeriod(t *testing.T) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = false
	sim.RelativisticEffects = false

	earthIdx := 2

	initialPos := sim.Planets[earthIdx].Position
	initialAngle := math.Atan2(initialPos.Y, initialPos.X)

	prevAngle := initialAngle
	totalAngle := 0.0

	maxSteps := 6000
	for step := 0; step < maxSteps; step++ {
		sim.Step(constants.BaseTimeStep)

		pos := sim.Planets[earthIdx].Position
		angle := math.Atan2(pos.Y, pos.X)

		dAngle := angle - prevAngle
		if dAngle > math.Pi {
			dAngle -= 2 * math.Pi
		} else if dAngle < -math.Pi {
			dAngle += 2 * math.Pi
		}
		totalAngle += dAngle
		prevAngle = angle

		if math.Abs(totalAngle) >= 2*math.Pi {
			elapsedDays := sim.CurrentTime / 86400.0
			t.Logf("Earth completed orbit in %.2f days (%d steps)", elapsedDays, step+1)

			expectedDays := 365.25
			relErr := math.Abs(elapsedDays-expectedDays) / expectedDays
			if relErr > 0.01 {
				t.Errorf("orbital period %.2f days differs from expected %.2f by %.2f%%",
					elapsedDays, expectedDays, relErr*100)
			}
			return
		}
	}

	t.Fatalf("Earth did not complete orbit in %d steps (total angle = %.4f rad)", maxSteps, totalAngle)
}

func TestEnergyConservationWithNBody(t *testing.T) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = true
	sim.RelativisticEffects = false

	E0 := computeTotalEnergy(sim)

	for i := 0; i < 1000; i++ {
		sim.Step(constants.BaseTimeStep)
	}

	E1 := computeTotalEnergy(sim)

	relDrift := math.Abs((E1 - E0) / E0)
	t.Logf("N-body energy: initial=%e, final=%e, relative drift=%e", E0, E1, relDrift)

	if relDrift > 1e-5 {
		t.Errorf("N-body energy conservation violated: relative drift %e > 1e-5", relDrift)
	}
}
