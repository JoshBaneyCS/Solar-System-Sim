package ui

import (
	"sync"

	"solar-system-sim/internal/physics"
)

// AppState is the single source of truth for all UI-observable application state.
// All mutations go through setter methods which atomically update the simulator,
// persist to settings, and notify registered listeners.
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

	// Backing references
	simulator *physics.Simulator
	app       *App

	// Observer pattern
	listeners []func()
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
	}
}

// AddListener registers a callback that fires on any state change.
func (s *AppState) AddListener(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, fn)
}

func (s *AppState) notifyListeners() {
	for _, fn := range s.listeners {
		fn()
	}
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

// --- Setters (atomically update simulator + notify listeners) ---

func (s *AppState) SetShowTrails(v bool) {
	s.mu.Lock()
	s.showTrails = v
	s.simulator.Lock()
	s.simulator.ShowTrails = v
	s.simulator.Unlock()
	if !v {
		s.simulator.ClearTrails()
	}
	s.mu.Unlock()
	s.notifyListeners()
}

func (s *AppState) SetShowSpacetime(v bool) {
	s.mu.Lock()
	s.showSpacetime = v
	s.simulator.Lock()
	s.simulator.ShowSpacetime = v
	s.simulator.Unlock()
	s.mu.Unlock()
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
	s.simulator.Lock()
	s.simulator.PlanetGravityEnabled = v
	s.simulator.Unlock()
	s.mu.Unlock()
	s.notifyListeners()
}

func (s *AppState) SetRelativity(v bool) {
	s.mu.Lock()
	s.relativity = v
	s.simulator.Lock()
	s.simulator.RelativisticEffects = v
	s.simulator.Unlock()
	s.mu.Unlock()
	s.notifyListeners()
}

func (s *AppState) SetIntegrator(v physics.IntegratorType) {
	s.mu.Lock()
	s.integrator = v
	s.simulator.Lock()
	s.simulator.Integrator = v
	s.simulator.Unlock()
	s.mu.Unlock()
	s.notifyListeners()
}

func (s *AppState) SetTimeSpeed(v float64) {
	s.mu.Lock()
	s.timeSpeed = v
	s.simulator.Lock()
	s.simulator.TimeSpeed = v
	s.simulator.Unlock()
	s.mu.Unlock()
	s.notifyListeners()
}

func (s *AppState) SetIsPlaying(v bool) {
	s.mu.Lock()
	s.isPlaying = v
	s.simulator.Lock()
	s.simulator.IsPlaying = v
	s.simulator.Unlock()
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

	s.simulator.Lock()
	s.simulator.ShowTrails = settings.ShowTrails
	s.simulator.ShowSpacetime = settings.ShowSpacetime
	s.simulator.PlanetGravityEnabled = settings.PlanetGravity
	s.simulator.RelativisticEffects = settings.Relativity
	s.simulator.Integrator = s.integrator
	s.simulator.Unlock()

	if s.app != nil {
		s.app.showLabels = settings.ShowLabels
		s.app.renderer.ShowLabels = settings.ShowLabels
	}
	s.mu.Unlock()
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
}

// RebindSimulator rebinds the state to a new simulator (used after reset).
func (s *AppState) RebindSimulator(sim *physics.Simulator) {
	s.mu.Lock()
	s.simulator = sim
	s.mu.Unlock()
}
