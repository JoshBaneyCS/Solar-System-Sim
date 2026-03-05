package main

import "testing"

func BenchmarkStep(b *testing.B) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = true
	sim.RelativisticEffects = true
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.step(baseTimeStep)
	}
}

func BenchmarkStepNewtonianOnly(b *testing.B) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = false
	sim.RelativisticEffects = false
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.step(baseTimeStep)
	}
}

func BenchmarkStepNBody(b *testing.B) {
	sim := NewSimulator()
	sim.PlanetGravityEnabled = true
	sim.RelativisticEffects = false
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.step(baseTimeStep)
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
		sim.calculateAccelerationWithSnapshot(0, sim.Planets[0].Position, sim.Planets[0].Velocity,
			sim.Planets[0].Mass, sim.Planets[0].Name, states)
	}
}
