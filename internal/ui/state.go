package ui

import (
	"sync"
	"time"

	"solar-system-sim/internal/physics"
)

// AppState is the single source of truth for all UI-observable application state.
// All mutations go through setter methods which update local state immediately
// and send non-blocking commands to the physics goroutine via SendCommand.
type AppState struct {
	mu sync.RWMutex

	// Canonical state
	showTrails    bool
	showSpacetime bool
	showLabels    bool
	planetGravity bool
	relativity    bool
	integrator    physics.IntegratorType
	timeSpeed     float64
	isPlaying     bool
	showMoons     bool
	showComets    bool
	showAsteroids bool
	showBelt      bool

	// Backing references
	simulator *physics.Simulator
	app       *App

	// Observer pattern
	listeners   []func()
	notifyTimer *time.Timer
}

// NewAppState creates a new AppState bound to the given simulator and app.
func NewAppState(sim *physics.Simulator, a *App) *AppState {
	return &AppState{
		simulator:     sim,
		app:           a,
		showTrails:    true,
		showSpacetime: false,
		showLabels:    true,
		planetGravity: true,
		relativity:    true,
		integrator:    physics.IntegratorVerlet,
		timeSpeed:     1.0,
		isPlaying:     true,
		showMoons:     true,
		showComets:    false,
		showAsteroids: false,
		showBelt:      true,
	}
}

// AddListener registers a callback that fires on any state change.
func (s *AppState) AddListener(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, fn)
}

// notifyListeners debounces listener notifications to avoid flooding
// Fyne's event loop during rapid slider drags.
func (s *AppState) notifyListeners() {
	if s.notifyTimer != nil {
		s.notifyTimer.Stop()
	}
	s.notifyTimer = time.AfterFunc(50*time.Millisecond, func() {
		s.mu.RLock()
		listeners := s.listeners
		s.mu.RUnlock()
		for _, fn := range listeners {
			fn()
		}
	})
}

// --- Getters ---

func (s *AppState) ShowTrails() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.showTrails
}

func (s *AppState) ShowSpacetime() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.showSpacetime
}

func (s *AppState) ShowLabels() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.showLabels
}

func (s *AppState) PlanetGravity() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.planetGravity
}

func (s *AppState) Relativity() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.relativity
}

func (s *AppState) Integrator() physics.IntegratorType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.integrator
}

func (s *AppState) TimeSpeed() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.timeSpeed
}

func (s *AppState) IsPlaying() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isPlaying
}

// --- Setters (update local state + send non-blocking command to physics) ---

func (s *AppState) SetShowTrails(v bool) {
	s.mu.Lock()
	s.showTrails = v
	s.mu.Unlock()
	s.simulator.SendCommand(physics.SimCommand{
		Apply: func(sim *physics.Simulator) {
			sim.ShowTrails = v
			if !v {
				for i := range sim.Planets {
					sim.Planets[i].Trail = sim.Planets[i].Trail[:0]
				}
			}
		},
	})
	s.notifyListeners()
}

func (s *AppState) SetShowSpacetime(v bool) {
	s.mu.Lock()
	s.showSpacetime = v
	s.mu.Unlock()
	s.simulator.SendCommand(physics.SimCommand{
		Apply: func(sim *physics.Simulator) {
			sim.ShowSpacetime = v
		},
	})
	s.notifyListeners()
}

func (s *AppState) SetShowLabels(v bool) {
	s.mu.Lock()
	s.showLabels = v
	if s.app != nil {
		s.app.showLabels = v
		s.app.renderer.ShowLabels = v
	}
	s.mu.Unlock()
	s.notifyListeners()
}

func (s *AppState) SetPlanetGravity(v bool) {
	s.mu.Lock()
	s.planetGravity = v
	s.mu.Unlock()
	s.simulator.SendCommand(physics.SimCommand{
		Apply: func(sim *physics.Simulator) {
			sim.PlanetGravityEnabled = v
		},
	})
	s.notifyListeners()
}

func (s *AppState) SetRelativity(v bool) {
	s.mu.Lock()
	s.relativity = v
	s.mu.Unlock()
	s.simulator.SendCommand(physics.SimCommand{
		Apply: func(sim *physics.Simulator) {
			sim.RelativisticEffects = v
		},
	})
	s.notifyListeners()
}

func (s *AppState) SetIntegrator(v physics.IntegratorType) {
	s.mu.Lock()
	s.integrator = v
	s.mu.Unlock()
	s.simulator.SendCommand(physics.SimCommand{
		Apply: func(sim *physics.Simulator) {
			sim.Integrator = v
		},
	})
	s.notifyListeners()
}

func (s *AppState) SetTimeSpeed(v float64) {
	s.mu.Lock()
	s.timeSpeed = v
	s.mu.Unlock()
	s.simulator.SendCommand(physics.SimCommand{
		Apply: func(sim *physics.Simulator) {
			sim.TimeSpeed = v
		},
	})
	s.notifyListeners()
}

func (s *AppState) SetIsPlaying(v bool) {
	s.mu.Lock()
	s.isPlaying = v
	s.mu.Unlock()
	s.simulator.SendCommand(physics.SimCommand{
		Apply: func(sim *physics.Simulator) {
			sim.IsPlaying = v
		},
	})
	s.notifyListeners()
}

func (s *AppState) ShowMoons() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.showMoons
}

func (s *AppState) ShowComets() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.showComets
}

func (s *AppState) ShowAsteroids() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.showAsteroids
}

func (s *AppState) ShowBelt() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.showBelt
}

func (s *AppState) SetShowMoons(v bool) {
	s.mu.Lock()
	s.showMoons = v
	s.mu.Unlock()
	s.simulator.SendCommand(physics.SimCommand{
		Apply: func(sim *physics.Simulator) {
			if v && !sim.ShowMoons {
				sim.AddMoons()
			} else if !v && sim.ShowMoons {
				sim.RemoveBodiesByType(physics.BodyTypeMoon)
				sim.ShowMoons = false
			}
		},
	})
	s.notifyListeners()
}

func (s *AppState) SetShowComets(v bool) {
	s.mu.Lock()
	s.showComets = v
	s.mu.Unlock()
	s.simulator.SendCommand(physics.SimCommand{
		Apply: func(sim *physics.Simulator) {
			if v && !sim.ShowComets {
				sim.AddComets()
			} else if !v && sim.ShowComets {
				sim.RemoveBodiesByType(physics.BodyTypeComet)
				sim.ShowComets = false
			}
		},
	})
	s.notifyListeners()
}

func (s *AppState) SetShowAsteroids(v bool) {
	s.mu.Lock()
	s.showAsteroids = v
	s.mu.Unlock()
	s.simulator.SendCommand(physics.SimCommand{
		Apply: func(sim *physics.Simulator) {
			if v && !sim.ShowAsteroids {
				sim.AddAsteroids()
			} else if !v && sim.ShowAsteroids {
				sim.RemoveBodiesByType(physics.BodyTypeAsteroid)
				sim.RemoveBodiesByType(physics.BodyTypeDwarfPlanet)
				sim.ShowAsteroids = false
			}
		},
	})
	s.notifyListeners()
}

func (s *AppState) SetShowBelt(v bool) {
	s.mu.Lock()
	s.showBelt = v
	if s.app != nil && s.app.renderer != nil {
		s.app.renderer.ShowBelt = v
	}
	s.mu.Unlock()
	s.notifyListeners()
}

// ApplyFromSettings loads settings values into the state and simulator.
func (s *AppState) ApplyFromSettings(settings Settings) {
	s.mu.Lock()
	s.showTrails = settings.ShowTrails
	s.showSpacetime = settings.ShowSpacetime
	s.showLabels = settings.ShowLabels
	s.planetGravity = settings.PlanetGravity
	s.relativity = settings.Relativity
	if settings.Integrator == "rk4" {
		s.integrator = physics.IntegratorRK4
	} else {
		s.integrator = physics.IntegratorVerlet
	}

	if s.app != nil {
		s.app.showLabels = settings.ShowLabels
		s.app.renderer.ShowLabels = settings.ShowLabels
		s.app.renderer.ShowBelt = settings.ShowBelt
	}
	s.mu.Unlock()

	// Send all simulator changes as a single command
	integ := s.integrator
	s.simulator.SendCommand(physics.SimCommand{
		Apply: func(sim *physics.Simulator) {
			sim.ShowTrails = settings.ShowTrails
			sim.ShowSpacetime = settings.ShowSpacetime
			sim.PlanetGravityEnabled = settings.PlanetGravity
			sim.RelativisticEffects = settings.Relativity
			sim.Integrator = integ
		},
	})

	// Apply body toggles
	s.SetShowMoons(settings.ShowMoons)
	s.SetShowComets(settings.ShowComets)
	s.SetShowAsteroids(settings.ShowAsteroids)

	s.notifyListeners()
}

// ToSettings converts the current state to a Settings struct for persistence.
func (s *AppState) ToSettings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	intStr := "verlet"
	if s.integrator == physics.IntegratorRK4 {
		intStr = "rk4"
	}
	return Settings{
		ShowTrails:    s.showTrails,
		ShowSpacetime: s.showSpacetime,
		ShowLabels:    s.showLabels,
		PlanetGravity: s.planetGravity,
		Relativity:    s.relativity,
		Integrator:    intStr,
		ShowMoons:     s.showMoons,
		ShowComets:    s.showComets,
		ShowAsteroids: s.showAsteroids,
		ShowBelt:      s.showBelt,
	}
}

// ResetToDefaults resets state to default values.
func (s *AppState) ResetToDefaults() {
	s.SetShowTrails(true)
	s.SetShowSpacetime(false)
	s.SetShowLabels(true)
	s.SetPlanetGravity(true)
	s.SetRelativity(true)
	s.SetIntegrator(physics.IntegratorVerlet)
	s.SetTimeSpeed(1.0)
	s.SetIsPlaying(true)
	s.SetShowMoons(true)
	s.SetShowComets(false)
	s.SetShowAsteroids(false)
	s.SetShowBelt(true)
}

// RebindSimulator rebinds the state to a new simulator (used after reset).
func (s *AppState) RebindSimulator(sim *physics.Simulator) {
	s.mu.Lock()
	s.simulator = sim
	s.mu.Unlock()
}
