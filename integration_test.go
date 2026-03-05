package main

import (
	"math"
	"testing"
)

// computeTotalEnergy calculates the total mechanical energy of the system.
// E = Σ(½mv²) + Σ(-GMm/r) for each planet relative to the Sun.
func computeTotalEnergy(sim *Simulator) float64 {
	totalE := 0.0
	for _, p := range sim.Planets {
		v := p.Velocity.Magnitude()
		r := p.Position.Sub(sim.Sun.Position).Magnitude()

		kinetic := 0.5 * p.Mass * v * v
		potential := -G * sim.SunMass * p.Mass / r

		totalE += kinetic + potential
	}
	return totalE
}

// computeTotalAngularMomentum calculates L = Σ m(r × v) for all planets.
func computeTotalAngularMomentum(sim *Simulator) Vec3 {
	totalL := Vec3{}
	for _, p := range sim.Planets {
		r := p.Position.Sub(sim.Sun.Position)
		L := r.Cross(p.Velocity).Mul(p.Mass)
		totalL = totalL.Add(L)
	}
	return totalL
}

func TestEnergyConservation(t *testing.T) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = false // Sun-only for cleaner energy conservation
	sim.RelativisticEffects = false

	E0 := computeTotalEnergy(sim)

	// Run 1000 steps at default timestep
	for i := 0; i < 1000; i++ {
		sim.step(baseTimeStep)
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
		sim.step(baseTimeStep)
	}

	L1 := computeTotalAngularMomentum(sim)

	// Check each component
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

	// Earth is at index 2
	earthIdx := 2

	// Record initial angle of Earth relative to Sun
	initialPos := sim.Planets[earthIdx].Position
	initialAngle := math.Atan2(initialPos.Y, initialPos.X)

	prevAngle := initialAngle
	totalAngle := 0.0

	// Run until Earth completes one full orbit (2π radians of angular travel)
	// Expected: ~365.25 days = 365.25 * 86400 / 7200 ≈ 4383 steps
	maxSteps := 6000
	for step := 0; step < maxSteps; step++ {
		sim.step(baseTimeStep)

		pos := sim.Planets[earthIdx].Position
		angle := math.Atan2(pos.Y, pos.X)

		// Accumulate angular change, handling wrapping
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
		sim.step(baseTimeStep)
	}

	E1 := computeTotalEnergy(sim)

	relDrift := math.Abs((E1 - E0) / E0)
	t.Logf("N-body energy: initial=%e, final=%e, relative drift=%e", E0, E1, relDrift)

	// N-body energy conservation is slightly worse due to planet-planet interactions
	// but should still be good with RK4
	if relDrift > 1e-5 {
		t.Errorf("N-body energy conservation violated: relative drift %e > 1e-5", relDrift)
	}
}
