package physics

import (
	"image/color"
	"math"
	"sync"

	"solar-system-sim/internal/math3d"
	"solar-system-sim/internal/physics/gr"
	"solar-system-sim/pkg/constants"
)

// MaxSafeDt is the maximum integration timestep (seconds) before substep subdivision.
// At 8 hours, Mercury gets ~264 steps/orbit even at max TimeSpeed.
const MaxSafeDt = 28800.0

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
	mu                   sync.RWMutex
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
		Name:     p.Name,
		Mass:     p.Mass,
		Position: math3d.Vec3{X: x, Y: y, Z: z},
		Velocity: math3d.Vec3{X: vx, Y: vy, Z: vz},
		Color:    p.Color,
		Radius:   p.DisplayRadius,
		Trail:    make([]math3d.Vec3, 0, s.maxTrailLen),
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

		if s.RelativisticEffects && bodyName == "Mercury" {
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

func (s *Simulator) Step(dt float64) {
	if s.Backend != nil {
		s.syncBackendConfig()
		s.Backend.Step(dt)
		pos, vel := s.Backend.GetState()
		for i := range s.Planets {
			s.Planets[i].Position = pos[i]
			s.Planets[i].Velocity = vel[i]

			if s.ShowTrails {
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

	pos0 := make([]math3d.Vec3, n)
	vel0 := make([]math3d.Vec3, n)
	for i := range s.Planets {
		pos0[i] = s.Planets[i].Position
		vel0[i] = s.Planets[i].Velocity
	}

	k1p := make([]math3d.Vec3, n)
	k1v := make([]math3d.Vec3, n)
	snapshot0 := make([]BodyState, n)
	for i := range s.Planets {
		snapshot0[i] = BodyState{Position: pos0[i], Velocity: vel0[i]}
	}
	for i := range s.Planets {
		k1p[i] = vel0[i]
		k1v[i] = s.CalculateAccelerationWithSnapshot(i, pos0[i], vel0[i], s.Planets[i].Mass, s.Planets[i].Name, snapshot0)
	}

	pos2 := make([]math3d.Vec3, n)
	vel2 := make([]math3d.Vec3, n)
	k2p := make([]math3d.Vec3, n)
	k2v := make([]math3d.Vec3, n)
	snapshot2 := make([]BodyState, n)
	for i := range s.Planets {
		pos2[i] = pos0[i].Add(k1p[i].Mul(dt / 2))
		vel2[i] = vel0[i].Add(k1v[i].Mul(dt / 2))
		snapshot2[i] = BodyState{Position: pos2[i], Velocity: vel2[i]}
	}
	for i := range s.Planets {
		k2p[i] = vel2[i]
		k2v[i] = s.CalculateAccelerationWithSnapshot(i, pos2[i], vel2[i], s.Planets[i].Mass, s.Planets[i].Name, snapshot2)
	}

	pos3 := make([]math3d.Vec3, n)
	vel3 := make([]math3d.Vec3, n)
	k3p := make([]math3d.Vec3, n)
	k3v := make([]math3d.Vec3, n)
	snapshot3 := make([]BodyState, n)
	for i := range s.Planets {
		pos3[i] = pos0[i].Add(k2p[i].Mul(dt / 2))
		vel3[i] = vel0[i].Add(k2v[i].Mul(dt / 2))
		snapshot3[i] = BodyState{Position: pos3[i], Velocity: vel3[i]}
	}
	for i := range s.Planets {
		k3p[i] = vel3[i]
		k3v[i] = s.CalculateAccelerationWithSnapshot(i, pos3[i], vel3[i], s.Planets[i].Mass, s.Planets[i].Name, snapshot3)
	}

	pos4 := make([]math3d.Vec3, n)
	vel4 := make([]math3d.Vec3, n)
	k4p := make([]math3d.Vec3, n)
	k4v := make([]math3d.Vec3, n)
	snapshot4 := make([]BodyState, n)
	for i := range s.Planets {
		pos4[i] = pos0[i].Add(k3p[i].Mul(dt))
		vel4[i] = vel0[i].Add(k3v[i].Mul(dt))
		snapshot4[i] = BodyState{Position: pos4[i], Velocity: vel4[i]}
	}
	for i := range s.Planets {
		k4p[i] = vel4[i]
		k4v[i] = s.CalculateAccelerationWithSnapshot(i, pos4[i], vel4[i], s.Planets[i].Mass, s.Planets[i].Name, snapshot4)
	}

	for i := range s.Planets {
		planet := &s.Planets[i]

		planet.Position = pos0[i].Add(
			k1p[i].Add(k2p[i].Mul(2)).Add(k3p[i].Mul(2)).Add(k4p[i]).Mul(dt / 6),
		)
		planet.Velocity = vel0[i].Add(
			k1v[i].Add(k2v[i].Mul(2)).Add(k3v[i].Mul(2)).Add(k4v[i]).Mul(dt / 6),
		)

		if s.ShowTrails {
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
			Name:     s.Planets[i].Name,
			Mass:     s.Planets[i].Mass,
			Position: s.Planets[i].Position,
			Velocity: s.Planets[i].Velocity,
			Color:    s.Planets[i].Color,
			Radius:   s.Planets[i].Radius,
			Trail:    make([]math3d.Vec3, len(s.Planets[i].Trail)),
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
