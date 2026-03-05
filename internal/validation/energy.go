package validation

import (
	"fmt"
	"math"

	"solar-system-sim/internal/physics"
	"solar-system-sim/pkg/constants"
)

func computeTotalEnergy(sim *physics.Simulator) float64 {
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

// ValidateEnergyConservation runs the simulation for the given number of years
// and checks that total energy is conserved to within tolerance.
func ValidateEnergyConservation(years float64) *Result {
	sim := physics.NewSimulator()
	sim.PlanetGravityEnabled = true
	sim.RelativisticEffects = false

	E0 := computeTotalEnergy(sim)

	totalSeconds := years * 365.25 * 24 * 3600
	steps := int(totalSeconds / constants.BaseTimeStep)

	for i := 0; i < steps; i++ {
		sim.Step(constants.BaseTimeStep)
	}

	E1 := computeTotalEnergy(sim)
	relDrift := math.Abs((E1 - E0) / E0)

	tolerance := 1e-4 * years // N-body with 2-hour timestep; RK4 drift scales with time
	return &Result{
		Scenario:  "Energy Conservation (N-body)",
		Pass:      relDrift < tolerance,
		Measured:  relDrift,
		Expected:  0,
		Tolerance: tolerance,
		Units:     "(relative drift)",
		Details:   fmt.Sprintf("E0=%.6e J, E1=%.6e J, %d steps over %.1f years", E0, E1, steps, years),
	}
}
