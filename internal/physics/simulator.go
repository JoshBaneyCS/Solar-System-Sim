package physics

import (
	"image/color"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"solar-system-sim/internal/math3d"
	"solar-system-sim/internal/physics/gr"
	"solar-system-sim/pkg/constants"
)

// SimSnapshot is a read-only snapshot of the simulation state.
// It is produced by the physics goroutine and consumed by the render loop
// via an atomic pointer swap, eliminating lock contention.
type SimSnapshot struct {
	Planets     []Body
	Sun         Body
	CurrentTime float64
	TimeSpeed   float64
	IsPlaying   bool
}

// SimCommand is a mutation to apply to the simulator from the UI thread.
// Commands are sent via SendCommand and processed by the physics goroutine,
// avoiding lock contention between UI and physics.
type SimCommand struct {
	Apply func(s *Simulator)
}

// MaxSafeDt is the maximum integration timestep (seconds) before substep subdivision.
// At 8 hours, Mercury gets ~264 steps/orbit even at max TimeSpeed.
const MaxSafeDt = 28800.0

// parallelThreshold is the minimum body count before parallelizing acceleration loops.
const parallelThreshold = 12

// rk4Scratch holds pre-allocated buffers for RK4 integration to avoid per-step allocations.
type rk4Scratch struct {
	pos0, vel0 []math3d.Vec3
	k1p, k1v   []math3d.Vec3
	k2p, k2v   []math3d.Vec3
	k3p, k3v   []math3d.Vec3
	k4p, k4v   []math3d.Vec3
	pos2, vel2 []math3d.Vec3
	pos3, vel3 []math3d.Vec3
	pos4, vel4 []math3d.Vec3
	snapshot   []BodyState
	cap        int
}

func newRK4Scratch(n int) *rk4Scratch {
	s := &rk4Scratch{cap: n}
	s.allocate(n)
	return s
}

func (s *rk4Scratch) ensureSize(n int) {
	if s.cap >= n {
		return
	}
	s.allocate(n)
}

func (s *rk4Scratch) allocate(n int) {
	s.pos0 = make([]math3d.Vec3, n)
	s.vel0 = make([]math3d.Vec3, n)
	s.k1p = make([]math3d.Vec3, n)
	s.k1v = make([]math3d.Vec3, n)
	s.k2p = make([]math3d.Vec3, n)
	s.k2v = make([]math3d.Vec3, n)
	s.k3p = make([]math3d.Vec3, n)
	s.k3v = make([]math3d.Vec3, n)
	s.k4p = make([]math3d.Vec3, n)
	s.k4v = make([]math3d.Vec3, n)
	s.pos2 = make([]math3d.Vec3, n)
	s.vel2 = make([]math3d.Vec3, n)
	s.pos3 = make([]math3d.Vec3, n)
	s.vel3 = make([]math3d.Vec3, n)
	s.pos4 = make([]math3d.Vec3, n)
	s.vel4 = make([]math3d.Vec3, n)
	s.snapshot = make([]BodyState, n)
	s.cap = n
}

// Simulator holds the simulation state
type Simulator struct {
	Sun                  Body
	Planets              []Body
	TimeSpeed            float64
	IsPlaying            bool
	ShowTrails           bool
	ShowSpacetime        bool
	CurrentTime          float64
	SunMass              float64
	DefaultMass          float64
	maxTrailLen          int
	PlanetGravityEnabled bool
	RelativisticEffects  bool
	Integrator           IntegratorType
	SofteningLength      float64
	Backend              PhysicsBackend
	ShowMoons            bool
	ShowComets           bool
	ShowAsteroids        bool
	mu                   sync.RWMutex
	scratch              *rk4Scratch
	latestSnapshot       atomic.Pointer[SimSnapshot]
	stopCh               chan struct{}
	commandCh            chan SimCommand
}

// RLock acquires a read lock on the simulator mutex.
func (s *Simulator) RLock() { s.mu.RLock() }

// RUnlock releases the read lock on the simulator mutex.
func (s *Simulator) RUnlock() { s.mu.RUnlock() }

// Lock acquires a write lock on the simulator mutex.
func (s *Simulator) Lock() { s.mu.Lock() }

// Unlock releases the write lock on the simulator mutex.
func (s *Simulator) Unlock() { s.mu.Unlock() }

func NewSimulator() *Simulator {
	sunMass := 1.989e30
	sim := &Simulator{
		Sun: Body{
			Name:     "Sun",
			Mass:     sunMass,
			Position: math3d.Vec3{X: 0, Y: 0, Z: 0},
			Velocity: math3d.Vec3{X: 0, Y: 0, Z: 0},
			Color:    color.RGBA{255, 204, 0, 255},
			Radius:   30,
		},
		TimeSpeed:            1.0,
		IsPlaying:            false,
		ShowTrails:           true,
		ShowSpacetime:        false,
		SunMass:              sunMass,
		DefaultMass:          sunMass,
		maxTrailLen:          500,
		PlanetGravityEnabled: true,
		RelativisticEffects:  true,
		Integrator:           IntegratorVerlet,
	}

	for _, pData := range PlanetData {
		sim.Planets = append(sim.Planets, sim.CreatePlanetFromElements(pData))
	}

	sim.scratch = newRK4Scratch(len(sim.Planets))
	sim.commandCh = make(chan SimCommand, 32)

	initBackend(sim)

	return sim
}

func (s *Simulator) CreatePlanetFromElements(p Planet) Body {
	a := p.SemiMajorAxis * constants.AU
	e := p.Eccentricity
	i := p.Inclination * math.Pi / 180
	Omega := p.LongitudeAscendingNode * math.Pi / 180
	omega := p.ArgumentOfPerihelion * math.Pi / 180
	nu := p.InitialAnomaly

	r := a * (1 - e*e) / (1 + e*math.Cos(nu))

	xOrb := r * math.Cos(nu)
	yOrb := r * math.Sin(nu)

	x1 := xOrb*math.Cos(omega) - yOrb*math.Sin(omega)
	y1 := xOrb*math.Sin(omega) + yOrb*math.Cos(omega)
	z1 := 0.0

	x2 := x1
	y2 := y1*math.Cos(i) - z1*math.Sin(i)
	z2 := y1*math.Sin(i) + z1*math.Cos(i)

	x := x2*math.Cos(Omega) - y2*math.Sin(Omega)
	y := x2*math.Sin(Omega) + y2*math.Cos(Omega)
	z := z2

	GM := constants.G * s.SunMass
	h := math.Sqrt(GM * a * (1 - e*e))
	muOverH := GM / h

	vxOrb := -muOverH * math.Sin(nu)
	vyOrb := muOverH * (e + math.Cos(nu))

	vx1 := vxOrb*math.Cos(omega) - vyOrb*math.Sin(omega)
	vy1 := vxOrb*math.Sin(omega) + vyOrb*math.Cos(omega)
	vz1 := 0.0

	vx2 := vx1
	vy2 := vy1*math.Cos(i) - vz1*math.Sin(i)
	vz2 := vy1*math.Sin(i) + vz1*math.Cos(i)

	vx := vx2*math.Cos(Omega) - vy2*math.Sin(Omega)
	vy := vx2*math.Sin(Omega) + vy2*math.Cos(Omega)
	vz := vz2

	return Body{
		Name:           p.Name,
		Mass:           p.Mass,
		Position:       math3d.Vec3{X: x, Y: y, Z: z},
		Velocity:       math3d.Vec3{X: vx, Y: vy, Z: vz},
		Color:          p.Color,
		Radius:         p.DisplayRadius,
		Trail:          make([]math3d.Vec3, 0, s.maxTrailLen),
		ShowTrail:      true,
		Type:           p.Type,
		PhysicalRadius: p.PhysicalRadius,
	}
}

func (s *Simulator) CalculateAccelerationWithSnapshot(
	bodyIndex int,
	bodyPos math3d.Vec3,
	bodyVel math3d.Vec3,
	bodyMass float64,
	bodyName string,
	planetStates []BodyState,
) math3d.Vec3 {
	totalAccel := math3d.Vec3{X: 0, Y: 0, Z: 0}

	rSun := s.Sun.Position.Sub(bodyPos)
	distanceSun := rSun.Magnitude()

	if distanceSun > 1e6 {
		rHatSun := rSun.Normalize()
		accelMagSun := constants.G * s.SunMass / (distanceSun*distanceSun + s.SofteningLength*s.SofteningLength)
		accelSun := rHatSun.Mul(accelMagSun)

		if s.RelativisticEffects {
			relAccel := gr.CalculateGRCorrection(bodyPos, bodyVel, s.SunMass, distanceSun)
			accelSun = accelSun.Add(relAccel)
		}

		totalAccel = totalAccel.Add(accelSun)
	}

	if s.PlanetGravityEnabled {
		for i := range planetStates {
			if i == bodyIndex {
				continue
			}

			otherPos := planetStates[i].Position
			otherMass := s.Planets[i].Mass

			rPlanet := otherPos.Sub(bodyPos)
			distancePlanet := rPlanet.Magnitude()

			if distancePlanet > 1e6 {
				rHatPlanet := rPlanet.Normalize()
				accelMagPlanet := constants.G * otherMass / (distancePlanet*distancePlanet + s.SofteningLength*s.SofteningLength)
				accelPlanet := rHatPlanet.Mul(accelMagPlanet)
				totalAccel = totalAccel.Add(accelPlanet)
			}
		}
	}

	return totalAccel
}

// computeAccelerations computes kp (velocity) and kv (acceleration) for all bodies.
// Parallelizes when body count exceeds parallelThreshold.
func (s *Simulator) computeAccelerations(n int, positions, velocities []math3d.Vec3, snapshot []BodyState, kp, kv []math3d.Vec3) {
	for i := 0; i < n; i++ {
		kp[i] = velocities[i]
	}

	if n >= parallelThreshold {
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				kv[idx] = s.CalculateAccelerationWithSnapshot(idx, positions[idx], velocities[idx], s.Planets[idx].Mass, s.Planets[idx].Name, snapshot)
			}(i)
		}
		wg.Wait()
	} else {
		for i := 0; i < n; i++ {
			kv[i] = s.CalculateAccelerationWithSnapshot(i, positions[i], velocities[i], s.Planets[i].Mass, s.Planets[i].Name, snapshot)
		}
	}
}

func (s *Simulator) Step(dt float64) {
	if s.Backend != nil {
		s.syncBackendConfig()
		s.Backend.Step(dt)
		pos, vel := s.Backend.GetState()
		for i := range s.Planets {
			s.Planets[i].Position = pos[i]
			s.Planets[i].Velocity = vel[i]

			if s.ShowTrails && s.Planets[i].ShowTrail {
				s.Planets[i].Trail = append(s.Planets[i].Trail, s.Planets[i].Position)
				if len(s.Planets[i].Trail) > s.maxTrailLen {
					s.Planets[i].Trail = s.Planets[i].Trail[1:]
				}
			}
		}
		s.CurrentTime += dt
		return
	}

	if s.Integrator == IntegratorVerlet {
		s.stepVerlet(dt)
		return
	}

	n := len(s.Planets)

	// Ensure scratch buffers are large enough
	if s.scratch == nil {
		s.scratch = newRK4Scratch(n)
	}
	s.scratch.ensureSize(n)
	sc := s.scratch

	// Copy current state
	for i := range s.Planets {
		sc.pos0[i] = s.Planets[i].Position
		sc.vel0[i] = s.Planets[i].Velocity
	}

	halfDt := dt / 2
	dt6 := dt / 6

	// k1
	for i := 0; i < n; i++ {
		sc.snapshot[i] = BodyState{Position: sc.pos0[i], Velocity: sc.vel0[i]}
	}
	s.computeAccelerations(n, sc.pos0, sc.vel0, sc.snapshot, sc.k1p, sc.k1v)

	// k2
	for i := 0; i < n; i++ {
		sc.pos2[i] = sc.pos0[i].Add(sc.k1p[i].Mul(halfDt))
		sc.vel2[i] = sc.vel0[i].Add(sc.k1v[i].Mul(halfDt))
		sc.snapshot[i] = BodyState{Position: sc.pos2[i], Velocity: sc.vel2[i]}
	}
	s.computeAccelerations(n, sc.pos2, sc.vel2, sc.snapshot, sc.k2p, sc.k2v)

	// k3
	for i := 0; i < n; i++ {
		sc.pos3[i] = sc.pos0[i].Add(sc.k2p[i].Mul(halfDt))
		sc.vel3[i] = sc.vel0[i].Add(sc.k2v[i].Mul(halfDt))
		sc.snapshot[i] = BodyState{Position: sc.pos3[i], Velocity: sc.vel3[i]}
	}
	s.computeAccelerations(n, sc.pos3, sc.vel3, sc.snapshot, sc.k3p, sc.k3v)

	// k4
	for i := 0; i < n; i++ {
		sc.pos4[i] = sc.pos0[i].Add(sc.k3p[i].Mul(dt))
		sc.vel4[i] = sc.vel0[i].Add(sc.k3v[i].Mul(dt))
		sc.snapshot[i] = BodyState{Position: sc.pos4[i], Velocity: sc.vel4[i]}
	}
	s.computeAccelerations(n, sc.pos4, sc.vel4, sc.snapshot, sc.k4p, sc.k4v)

	// Final RK4 combination
	for i := range s.Planets {
		planet := &s.Planets[i]

		planet.Position = sc.pos0[i].Add(
			sc.k1p[i].Add(sc.k2p[i].Mul(2)).Add(sc.k3p[i].Mul(2)).Add(sc.k4p[i]).Mul(dt6),
		)
		planet.Velocity = sc.vel0[i].Add(
			sc.k1v[i].Add(sc.k2v[i].Mul(2)).Add(sc.k3v[i].Mul(2)).Add(sc.k4v[i]).Mul(dt6),
		)

		if s.ShowTrails && s.Planets[i].ShowTrail {
			planet.Trail = append(planet.Trail, planet.Position)
			if len(planet.Trail) > s.maxTrailLen {
				planet.Trail = planet.Trail[1:]
			}
		}
	}

	s.CurrentTime += dt
}

func (s *Simulator) Update(dt float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.IsPlaying {
		effectiveDt := dt * s.TimeSpeed
		absDt := math.Abs(effectiveDt)
		if absDt <= MaxSafeDt {
			s.Step(effectiveDt)
		} else {
			nSub := int(math.Ceil(absDt / MaxSafeDt))
			subDt := effectiveDt / float64(nSub)
			for i := 0; i < nSub; i++ {
				s.Step(subDt)
			}
		}
	}
}

// CreateMoonFromElements converts a moon's orbital elements (relative to parent) to
// heliocentric position/velocity by computing the orbit around the parent body
// and adding the parent's heliocentric state.
func (s *Simulator) CreateMoonFromElements(moon Planet, parent Body) Body {
	a := moon.SemiMajorAxis * constants.AU // SemiMajorAxis stored in AU
	e := moon.Eccentricity
	inc := moon.Inclination * math.Pi / 180
	Omega := moon.LongitudeAscendingNode * math.Pi / 180
	omega := moon.ArgumentOfPerihelion * math.Pi / 180
	nu := moon.InitialAnomaly

	r := a * (1 - e*e) / (1 + e*math.Cos(nu))

	xOrb := r * math.Cos(nu)
	yOrb := r * math.Sin(nu)

	x1 := xOrb*math.Cos(omega) - yOrb*math.Sin(omega)
	y1 := xOrb*math.Sin(omega) + yOrb*math.Cos(omega)

	x2 := x1
	y2 := y1 * math.Cos(inc)
	z2 := y1 * math.Sin(inc)

	relX := x2*math.Cos(Omega) - y2*math.Sin(Omega)
	relY := x2*math.Sin(Omega) + y2*math.Cos(Omega)
	relZ := z2

	GM := constants.G * moon.ParentMass
	h := math.Sqrt(GM * a * (1 - e*e))
	muOverH := GM / h

	vxOrb := -muOverH * math.Sin(nu)
	vyOrb := muOverH * (e + math.Cos(nu))

	vx1 := vxOrb*math.Cos(omega) - vyOrb*math.Sin(omega)
	vy1 := vxOrb*math.Sin(omega) + vyOrb*math.Cos(omega)

	vx2 := vx1
	vy2 := vy1 * math.Cos(inc)
	vz2 := vy1 * math.Sin(inc)

	relVx := vx2*math.Cos(Omega) - vy2*math.Sin(Omega)
	relVy := vx2*math.Sin(Omega) + vy2*math.Cos(Omega)
	relVz := vz2

	return Body{
		Name:           moon.Name,
		Mass:           moon.Mass,
		Position:       math3d.Vec3{X: parent.Position.X + relX, Y: parent.Position.Y + relY, Z: parent.Position.Z + relZ},
		Velocity:       math3d.Vec3{X: parent.Velocity.X + relVx, Y: parent.Velocity.Y + relVy, Z: parent.Velocity.Z + relVz},
		Color:          moon.Color,
		Radius:         moon.DisplayRadius,
		Trail:          make([]math3d.Vec3, 0, s.maxTrailLen),
		ShowTrail:      true,
		Type:           moon.Type,
		PhysicalRadius: moon.PhysicalRadius,
	}
}

// AddMoons adds all defined moons to the simulation, computing their initial
// heliocentric state from their parent body's current position/velocity.
func (s *Simulator) AddMoons() {
	for _, moonData := range MoonData {
		var parent *Body
		for i := range s.Planets {
			if s.Planets[i].Name == moonData.ParentName {
				parent = &s.Planets[i]
				break
			}
		}
		if parent == nil {
			continue
		}
		s.Planets = append(s.Planets, s.CreateMoonFromElements(moonData, *parent))
	}
	s.ShowMoons = true
}

// AddComets adds all defined comets to the simulation.
func (s *Simulator) AddComets() {
	for _, cData := range CometData {
		body := s.CreatePlanetFromElements(cData)
		body.Type = BodyTypeComet
		s.Planets = append(s.Planets, body)
	}
	s.ShowComets = true
}

// AddAsteroids adds all defined named asteroids to the simulation.
func (s *Simulator) AddAsteroids() {
	for _, aData := range AsteroidData {
		body := s.CreatePlanetFromElements(aData)
		body.Type = aData.Type
		s.Planets = append(s.Planets, body)
	}
	s.ShowAsteroids = true
}

// RemoveBodiesByType removes all bodies of the given type from the simulation.
func (s *Simulator) RemoveBodiesByType(t BodyType) {
	filtered := make([]Body, 0, len(s.Planets))
	for _, b := range s.Planets {
		if b.Type != t {
			filtered = append(filtered, b)
		}
	}
	s.Planets = filtered
}

func (s *Simulator) SetSunMass(massMultiplier float64) {
	s.SunMass = s.DefaultMass * massMultiplier
	s.syncBackendConfig()
}

// syncBackendConfig pushes current config to the Rust backend if present.
func (s *Simulator) syncBackendConfig() {
	if s.Backend != nil {
		s.Backend.SetConfig(s.SunMass, s.PlanetGravityEnabled, s.RelativisticEffects)
	}
}

func (s *Simulator) ClearTrails() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.Planets {
		s.Planets[i].Trail = make([]math3d.Vec3, 0, s.maxTrailLen)
	}
}

func (s *Simulator) GetPlanetSnapshot() []Body {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make([]Body, len(s.Planets))
	for i := range s.Planets {
		snapshot[i] = Body{
			Name:           s.Planets[i].Name,
			Mass:           s.Planets[i].Mass,
			Position:       s.Planets[i].Position,
			Velocity:       s.Planets[i].Velocity,
			Color:          s.Planets[i].Color,
			Radius:         s.Planets[i].Radius,
			Trail:          make([]math3d.Vec3, len(s.Planets[i].Trail)),
			ShowTrail:      s.Planets[i].ShowTrail,
			Type:           s.Planets[i].Type,
			PhysicalRadius: s.Planets[i].PhysicalRadius,
		}
		copy(snapshot[i].Trail, s.Planets[i].Trail)
	}

	return snapshot
}

func (s *Simulator) GetSunSnapshot() Body {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return Body{
		Name:     s.Sun.Name,
		Mass:     s.Sun.Mass,
		Position: s.Sun.Position,
		Velocity: s.Sun.Velocity,
		Color:    s.Sun.Color,
		Radius:   s.Sun.Radius,
	}
}

// publishSnapshot builds an immutable snapshot and stores it atomically.
// Must be called while holding s.mu (read or write lock).
func (s *Simulator) publishSnapshot() {
	snap := &SimSnapshot{
		Planets: make([]Body, len(s.Planets)),
		Sun: Body{
			Name:     s.Sun.Name,
			Mass:     s.Sun.Mass,
			Position: s.Sun.Position,
			Velocity: s.Sun.Velocity,
			Color:    s.Sun.Color,
			Radius:   s.Sun.Radius,
		},
		CurrentTime: s.CurrentTime,
		TimeSpeed:   s.TimeSpeed,
		IsPlaying:   s.IsPlaying,
	}
	for i := range s.Planets {
		snap.Planets[i] = Body{
			Name:           s.Planets[i].Name,
			Mass:           s.Planets[i].Mass,
			Position:       s.Planets[i].Position,
			Velocity:       s.Planets[i].Velocity,
			Color:          s.Planets[i].Color,
			Radius:         s.Planets[i].Radius,
			Trail:          make([]math3d.Vec3, len(s.Planets[i].Trail)),
			ShowTrail:      s.Planets[i].ShowTrail,
			Type:           s.Planets[i].Type,
			PhysicalRadius: s.Planets[i].PhysicalRadius,
		}
		copy(snap.Planets[i].Trail, s.Planets[i].Trail)
	}
	s.latestSnapshot.Store(snap)
}

// GetSnapshot returns the latest simulation snapshot without any locking.
// Returns nil if no snapshot has been published yet.
func (s *Simulator) GetSnapshot() *SimSnapshot {
	return s.latestSnapshot.Load()
}

// SendCommand enqueues a mutation to be applied by the physics goroutine.
// If the physics loop is not running, the command is applied directly with a lock.
// This is non-blocking from the UI thread.
func (s *Simulator) SendCommand(cmd SimCommand) {
	// If physics loop isn't running, apply directly
	if s.stopCh == nil {
		s.mu.Lock()
		cmd.Apply(s)
		s.mu.Unlock()
		return
	}
	select {
	case s.commandCh <- cmd:
	default:
		// Channel full — apply directly with lock as fallback
		s.mu.Lock()
		cmd.Apply(s)
		s.mu.Unlock()
	}
}

// drainCommands processes all pending commands. Must be called while holding s.mu write lock.
func (s *Simulator) drainCommands() {
	for {
		select {
		case cmd := <-s.commandCh:
			cmd.Apply(s)
		default:
			return
		}
	}
}

// StartPhysicsLoop launches a background goroutine that steps the simulation
// at ~60Hz. The render loop should use GetSnapshot() instead of calling Update().
func (s *Simulator) StartPhysicsLoop(dt float64) {
	s.stopCh = make(chan struct{})

	// Publish initial snapshot
	s.mu.RLock()
	s.publishSnapshot()
	s.mu.RUnlock()

	go func() {
		ticker := time.NewTicker(16 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				// Process pending UI commands + physics step under one lock
				s.mu.Lock()
				s.drainCommands()
				if s.IsPlaying {
					effectiveDt := dt * s.TimeSpeed
					absDt := math.Abs(effectiveDt)
					if absDt <= MaxSafeDt {
						s.Step(effectiveDt)
					} else {
						nSub := int(math.Ceil(absDt / MaxSafeDt))
						subDt := effectiveDt / float64(nSub)
						for i := 0; i < nSub; i++ {
							s.Step(subDt)
						}
					}
				}
				s.publishSnapshot()
				s.mu.Unlock()
			}
		}
	}()
}

// StopPhysicsLoop stops the background physics goroutine.
func (s *Simulator) StopPhysicsLoop() {
	if s.stopCh != nil {
		close(s.stopCh)
	}
}
