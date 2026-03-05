package validation

import (
	"fmt"
	"math"

	"solar-system-sim/internal/math3d"
	"solar-system-sim/internal/physics"
	"solar-system-sim/pkg/constants"
)

func computeTotalAngularMomentum(sim *physics.Simulator) math3d.Vec3 {
	totalL := math3d.Vec3{}
	for _, p := range sim.Planets {
		r := p.Position.Sub(sim.Sun.Position)
		L := r.Cross(p.Velocity).Mul(p.Mass)
		totalL = totalL.Add(L)
	}
	return totalL
}

// ValidateAngularMomentumConservation runs the simulation and checks
// that total angular momentum magnitude is conserved.
func ValidateAngularMomentumConservation(years float64) *Result {
	sim := physics.NewSimulator()
	sim.PlanetGravityEnabled = false
	sim.RelativisticEffects = false

	L0 := computeTotalAngularMomentum(sim)

	totalSeconds := years * 365.25 * 24 * 3600
	steps := int(totalSeconds / constants.BaseTimeStep)

	for i := 0; i < steps; i++ {
		sim.Step(constants.BaseTimeStep)
	}

	L1 := computeTotalAngularMomentum(sim)
	relDrift := math.Abs(L1.Magnitude()-L0.Magnitude()) / L0.Magnitude()

	tolerance := 1e-6
	return &Result{
		Scenario:  "Angular Momentum Conservation",
		Pass:      relDrift < tolerance,
		Measured:  relDrift,
		Expected:  0,
		Tolerance: tolerance,
		Units:     "(relative drift)",
		Details:   fmt.Sprintf("|L0|=%.6e, |L1|=%.6e, %d steps over %.1f years", L0.Magnitude(), L1.Magnitude(), steps, years),
	}
}
