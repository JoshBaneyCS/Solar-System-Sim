package physics

import (
	"math"
	"testing"

	"solar-system-sim/pkg/constants"
)

func TestVerletEnergyConservation(t *testing.T) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = false
	sim.RelativisticEffects = false
	sim.Integrator = IntegratorVerlet

	E0 := computeTotalEnergy(sim)

	for i := 0; i < 1000; i++ {
		sim.Step(constants.BaseTimeStep)
	}

	E1 := computeTotalEnergy(sim)
	relDrift := math.Abs((E1 - E0) / E0)
	t.Logf("Verlet energy: initial=%e, final=%e, relative drift=%e", E0, E1, relDrift)

	// Verlet is symplectic, so energy should be very well conserved
	if relDrift > 1e-6 {
		t.Errorf("Verlet energy conservation violated: relative drift %e > 1e-6", relDrift)
	}
}

func TestVerletAngularMomentumConservation(t *testing.T) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = false
	sim.RelativisticEffects = false
	sim.Integrator = IntegratorVerlet

	L0 := computeTotalAngularMomentum(sim)

	for i := 0; i < 1000; i++ {
		sim.Step(constants.BaseTimeStep)
	}

	L1 := computeTotalAngularMomentum(sim)
	relDrift := math.Abs(L1.Magnitude()-L0.Magnitude()) / L0.Magnitude()
	t.Logf("Verlet angular momentum: L0=%e, L1=%e, relative drift=%e",
		L0.Magnitude(), L1.Magnitude(), relDrift)

	if relDrift > 1e-6 {
		t.Errorf("Verlet angular momentum conservation violated: relative drift %e > 1e-6", relDrift)
	}
}

func TestVerletEarthOrbitalPeriod(t *testing.T) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = false
	sim.RelativisticEffects = false
	sim.Integrator = IntegratorVerlet

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
			t.Logf("Verlet: Earth completed orbit in %.2f days (%d steps)", elapsedDays, step+1)

			expectedDays := 365.25
			relErr := math.Abs(elapsedDays-expectedDays) / expectedDays
			if relErr > 0.01 {
				t.Errorf("orbital period %.2f days differs from expected %.2f by %.2f%%",
					elapsedDays, expectedDays, relErr*100)
			}
			return
		}
	}

	t.Fatalf("Verlet: Earth did not complete orbit in %d steps (total angle = %.4f rad)", maxSteps, totalAngle)
}
