package physics

import (
	"testing"

	"solar-system-sim/pkg/constants"
)

func BenchmarkStep(b *testing.B) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = true
	sim.RelativisticEffects = true
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.Step(constants.BaseTimeStep)
	}
}

func BenchmarkStepNewtonianOnly(b *testing.B) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = false
	sim.RelativisticEffects = false
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.Step(constants.BaseTimeStep)
	}
}

func BenchmarkStepNBody(b *testing.B) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = true
	sim.RelativisticEffects = false
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.Step(constants.BaseTimeStep)
	}
}

func BenchmarkCalculateAcceleration(b *testing.B) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = true
	sim.RelativisticEffects = true
	states := make([]BodyState, len(sim.Planets))
	for i, p := range sim.Planets {
		states[i] = BodyState{Position: p.Position, Velocity: p.Velocity}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.CalculateAccelerationWithSnapshot(0, sim.Planets[0].Position, sim.Planets[0].Velocity,
			sim.Planets[0].Mass, sim.Planets[0].Name, states)
	}
}
