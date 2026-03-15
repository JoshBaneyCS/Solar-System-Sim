package ui

import (
	"sync"
	"testing"

	"solar-system-sim/internal/physics"
)

func TestAppState_SetShowTrails(t *testing.T) {
	sim := physics.NewSimulator()
	state := NewAppState(sim, nil)

	state.SetShowTrails(false)
	if state.ShowTrails() {
		t.Error("expected ShowTrails false")
	}
	sim.RLock()
	if sim.ShowTrails {
		t.Error("expected simulator ShowTrails false")
	}
	sim.RUnlock()

	state.SetShowTrails(true)
	if !state.ShowTrails() {
		t.Error("expected ShowTrails true")
	}
}

func TestAppState_ListenerFired(t *testing.T) {
	sim := physics.NewSimulator()
	state := NewAppState(sim, nil)

	fired := false
	state.AddListener(func() {
		fired = true
	})

	state.SetShowSpacetime(true)
	if !fired {
		t.Error("listener should have been called")
	}
}

func TestAppState_SetIntegrator(t *testing.T) {
	sim := physics.NewSimulator()
	state := NewAppState(sim, nil)

	state.SetIntegrator(physics.IntegratorRK4)
	if state.Integrator() != physics.IntegratorRK4 {
		t.Error("expected RK4 integrator")
	}

	sim.RLock()
	if sim.Integrator != physics.IntegratorRK4 {
		t.Error("expected simulator RK4")
	}
	sim.RUnlock()
}

func TestAppState_ToSettings(t *testing.T) {
	sim := physics.NewSimulator()
	state := NewAppState(sim, nil)

	state.SetShowTrails(false)
	state.SetRelativity(false)
	state.SetIntegrator(physics.IntegratorRK4)

	s := state.ToSettings()
	if s.ShowTrails {
		t.Error("expected ShowTrails false in settings")
	}
	if s.Relativity {
		t.Error("expected Relativity false in settings")
	}
	if s.Integrator != "rk4" {
		t.Errorf("expected integrator 'rk4', got '%s'", s.Integrator)
	}
}

func TestAppState_ApplyFromSettings(t *testing.T) {
	sim := physics.NewSimulator()
	state := NewAppState(sim, nil)

	settings := Settings{
		ShowTrails:    false,
		ShowSpacetime: true,
		ShowLabels:    false,
		PlanetGravity: false,
		Relativity:    false,
		Integrator:    "rk4",
	}

	state.ApplyFromSettings(settings)

	if state.ShowTrails() {
		t.Error("expected trails false")
	}
	if !state.ShowSpacetime() {
		t.Error("expected spacetime true")
	}
	if state.PlanetGravity() {
		t.Error("expected planet gravity false")
	}
	if state.Integrator() != physics.IntegratorRK4 {
		t.Error("expected RK4")
	}
}

func TestAppState_RebindSimulator(t *testing.T) {
	sim1 := physics.NewSimulator()
	sim2 := physics.NewSimulator()
	state := NewAppState(sim1, nil)

	state.RebindSimulator(sim2)
	state.SetShowTrails(false)

	sim2.RLock()
	if sim2.ShowTrails {
		t.Error("expected new simulator to have trails false")
	}
	sim2.RUnlock()
}

func TestAppState_ConcurrentAccess(t *testing.T) {
	sim := physics.NewSimulator()
	state := NewAppState(sim, nil)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			state.SetShowTrails(true)
			state.SetTimeSpeed(2.0)
		}()
		go func() {
			defer wg.Done()
			_ = state.ShowTrails()
			_ = state.TimeSpeed()
		}()
	}
	wg.Wait()
}

func TestAppState_ResetToDefaults(t *testing.T) {
	sim := physics.NewSimulator()
	state := NewAppState(sim, nil)

	state.SetShowTrails(false)
	state.SetRelativity(false)
	state.SetTimeSpeed(10.0)

	state.ResetToDefaults()

	if !state.ShowTrails() {
		t.Error("expected trails true after reset")
	}
	if !state.Relativity() {
		t.Error("expected relativity true after reset")
	}
	if state.TimeSpeed() != 1.0 {
		t.Errorf("expected time speed 1.0, got %f", state.TimeSpeed())
	}
}
