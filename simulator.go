package main

import (
	"image/color"
	"math"
	"sync"
)

// Body represents a celestial body
type Body struct {
	Name     string
	Mass     float64 // kg
	Position Vec3    // meters
	Velocity Vec3    // m/s
	Color    color.Color
	Radius   float64 // display radius in pixels
	Trail    []Vec3  // orbital trail
}

// Planet holds real solar system data
type Planet struct {
	Name                   string
	Mass                   float64 // kg
	SemiMajorAxis          float64 // AU
	Eccentricity           float64
	Inclination            float64 // degrees
	LongitudeAscendingNode float64 // degrees
	ArgumentOfPerihelion   float64 // degrees
	OrbitalPeriod          float64 // Earth days
	Color                  color.Color
	DisplayRadius          float64
	InitialAnomaly         float64 // radians
}

// Real planetary data with 3D orbital elements
var planetData = []Planet{
	{
		Name:                   "Mercury",
		Mass:                   3.3011e23,
		SemiMajorAxis:          0.387,
		Eccentricity:           0.2056,
		Inclination:            7.005,
		LongitudeAscendingNode: 48.331,
		ArgumentOfPerihelion:   29.124,
		OrbitalPeriod:          87.97,
		Color:                  color.RGBA{169, 169, 169, 255},
		DisplayRadius:          4,
		InitialAnomaly:         0,
	},
	{
		Name:                   "Venus",
		Mass:                   4.8675e24,
		SemiMajorAxis:          0.723,
		Eccentricity:           0.0068,
		Inclination:            3.395,
		LongitudeAscendingNode: 76.680,
		ArgumentOfPerihelion:   54.884,
		OrbitalPeriod:          224.7,
		Color:                  color.RGBA{255, 198, 73, 255},
		DisplayRadius:          8,
		InitialAnomaly:         math.Pi / 4,
	},
	{
		Name:                   "Earth",
		Mass:                   5.972e24,
		SemiMajorAxis:          1.0,
		Eccentricity:           0.0167,
		Inclination:            0.0,
		LongitudeAscendingNode: 0.0,
		ArgumentOfPerihelion:   102.937,
		OrbitalPeriod:          365.25,
		Color:                  color.RGBA{100, 149, 237, 255},
		DisplayRadius:          8,
		InitialAnomaly:         math.Pi / 2,
	},
	{
		Name:                   "Mars",
		Mass:                   6.4171e23,
		SemiMajorAxis:          1.524,
		Eccentricity:           0.0934,
		Inclination:            1.850,
		LongitudeAscendingNode: 49.558,
		ArgumentOfPerihelion:   286.502,
		OrbitalPeriod:          687.0,
		Color:                  color.RGBA{193, 68, 14, 255},
		DisplayRadius:          6,
		InitialAnomaly:         3 * math.Pi / 4,
	},
	{
		Name:                   "Jupiter",
		Mass:                   1.8982e27,
		SemiMajorAxis:          5.203,
		Eccentricity:           0.0489,
		Inclination:            1.303,
		LongitudeAscendingNode: 100.464,
		ArgumentOfPerihelion:   273.867,
		OrbitalPeriod:          4331,
		Color:                  color.RGBA{216, 202, 157, 255},
		DisplayRadius:          20,
		InitialAnomaly:         math.Pi,
	},
	{
		Name:                   "Saturn",
		Mass:                   5.6834e26,
		SemiMajorAxis:          9.537,
		Eccentricity:           0.0565,
		Inclination:            2.485,
		LongitudeAscendingNode: 113.665,
		ArgumentOfPerihelion:   339.392,
		OrbitalPeriod:          10747,
		Color:                  color.RGBA{250, 222, 164, 255},
		DisplayRadius:          18,
		InitialAnomaly:         5 * math.Pi / 4,
	},
	{
		Name:                   "Uranus",
		Mass:                   8.6810e25,
		SemiMajorAxis:          19.191,
		Eccentricity:           0.0457,
		Inclination:            0.773,
		LongitudeAscendingNode: 74.006,
		ArgumentOfPerihelion:   96.998,
		OrbitalPeriod:          30589,
		Color:                  color.RGBA{79, 208, 231, 255},
		DisplayRadius:          14,
		InitialAnomaly:         3 * math.Pi / 2,
	},
	{
		Name:                   "Neptune",
		Mass:                   1.02413e26,
		SemiMajorAxis:          30.07,
		Eccentricity:           0.0113,
		Inclination:            1.770,
		LongitudeAscendingNode: 131.784,
		ArgumentOfPerihelion:   276.336,
		OrbitalPeriod:          59800,
		Color:                  color.RGBA{63, 84, 186, 255},
		DisplayRadius:          14,
		InitialAnomaly:         7 * math.Pi / 4,
	},
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
	mu                   sync.RWMutex
}

// BodyState holds a snapshot of body position and velocity for RK4 integration
type BodyState struct {
	Position Vec3
	Velocity Vec3
}

func NewSimulator() *Simulator {
	sunMass := 1.989e30
	sim := &Simulator{
		Sun: Body{
			Name:     "Sun",
			Mass:     sunMass,
			Position: Vec3{0, 0, 0},
			Velocity: Vec3{0, 0, 0},
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
	}

	for _, pData := range planetData {
		sim.Planets = append(sim.Planets, sim.createPlanetFromElements(pData))
	}

	return sim
}

func (s *Simulator) createPlanetFromElements(p Planet) Body {
	a := p.SemiMajorAxis * AU
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

	GM := G * s.SunMass
	h := math.Sqrt(GM * a * (1 - e*e))

	vxOrb := -h * math.Sin(nu) / r
	vyOrb := h * (e + math.Cos(nu)) / r

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
		Position: Vec3{x, y, z},
		Velocity: Vec3{vx, vy, vz},
		Color:    p.Color,
		Radius:   p.DisplayRadius,
		Trail:    make([]Vec3, 0, s.maxTrailLen),
	}
}

func (s *Simulator) calculateAccelerationWithSnapshot(
	bodyIndex int,
	bodyPos Vec3,
	bodyVel Vec3,
	bodyMass float64,
	bodyName string,
	planetStates []BodyState,
) Vec3 {
	totalAccel := Vec3{0, 0, 0}

	rSun := s.Sun.Position.Sub(bodyPos)
	distanceSun := rSun.Magnitude()

	if distanceSun > 1e6 {
		rHatSun := rSun.Normalize()
		accelMagSun := G * s.SunMass / (distanceSun * distanceSun)
		accelSun := rHatSun.Mul(accelMagSun)

		if s.RelativisticEffects && bodyName == "Mercury" {
			c := 299792458.0
			rVec := bodyPos
			vVec := bodyVel
			LVec := rVec.Cross(vVec)
			LMag := LVec.Magnitude()

			if LMag > 1e-10 {
				relAccel := LVec.Cross(rVec).Mul(3 * G * G * s.SunMass * s.SunMass /
					(c * c * distanceSun * distanceSun * distanceSun * LMag))
				accelSun = accelSun.Add(relAccel)
			}
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
				accelMagPlanet := G * otherMass / (distancePlanet * distancePlanet)
				accelPlanet := rHatPlanet.Mul(accelMagPlanet)
				totalAccel = totalAccel.Add(accelPlanet)
			}
		}
	}

	return totalAccel
}

func (s *Simulator) step(dt float64) {
	n := len(s.Planets)

	// Store initial states for all planets
	pos0 := make([]Vec3, n)
	vel0 := make([]Vec3, n)
	for i := range s.Planets {
		pos0[i] = s.Planets[i].Position
		vel0[i] = s.Planets[i].Velocity
	}

	// Calculate k1 for all planets at t=0
	k1p := make([]Vec3, n)
	k1v := make([]Vec3, n)
	snapshot0 := make([]BodyState, n)
	for i := range s.Planets {
		snapshot0[i] = BodyState{Position: pos0[i], Velocity: vel0[i]}
	}
	for i := range s.Planets {
		k1p[i] = vel0[i]
		k1v[i] = s.calculateAccelerationWithSnapshot(i, pos0[i], vel0[i], s.Planets[i].Mass, s.Planets[i].Name, snapshot0)
	}

	// Advance ALL planets to t+dt/2, then calculate k2
	pos2 := make([]Vec3, n)
	vel2 := make([]Vec3, n)
	k2p := make([]Vec3, n)
	k2v := make([]Vec3, n)
	snapshot2 := make([]BodyState, n)
	for i := range s.Planets {
		pos2[i] = pos0[i].Add(k1p[i].Mul(dt / 2))
		vel2[i] = vel0[i].Add(k1v[i].Mul(dt / 2))
		snapshot2[i] = BodyState{Position: pos2[i], Velocity: vel2[i]}
	}
	for i := range s.Planets {
		k2p[i] = vel2[i]
		k2v[i] = s.calculateAccelerationWithSnapshot(i, pos2[i], vel2[i], s.Planets[i].Mass, s.Planets[i].Name, snapshot2)
	}

	// Advance ALL planets to t+dt/2 using k2, then calculate k3
	pos3 := make([]Vec3, n)
	vel3 := make([]Vec3, n)
	k3p := make([]Vec3, n)
	k3v := make([]Vec3, n)
	snapshot3 := make([]BodyState, n)
	for i := range s.Planets {
		pos3[i] = pos0[i].Add(k2p[i].Mul(dt / 2))
		vel3[i] = vel0[i].Add(k2v[i].Mul(dt / 2))
		snapshot3[i] = BodyState{Position: pos3[i], Velocity: vel3[i]}
	}
	for i := range s.Planets {
		k3p[i] = vel3[i]
		k3v[i] = s.calculateAccelerationWithSnapshot(i, pos3[i], vel3[i], s.Planets[i].Mass, s.Planets[i].Name, snapshot3)
	}

	// Advance ALL planets to t+dt using k3, then calculate k4
	pos4 := make([]Vec3, n)
	vel4 := make([]Vec3, n)
	k4p := make([]Vec3, n)
	k4v := make([]Vec3, n)
	snapshot4 := make([]BodyState, n)
	for i := range s.Planets {
		pos4[i] = pos0[i].Add(k3p[i].Mul(dt))
		vel4[i] = vel0[i].Add(k3v[i].Mul(dt))
		snapshot4[i] = BodyState{Position: pos4[i], Velocity: vel4[i]}
	}
	for i := range s.Planets {
		k4p[i] = vel4[i]
		k4v[i] = s.calculateAccelerationWithSnapshot(i, pos4[i], vel4[i], s.Planets[i].Mass, s.Planets[i].Name, snapshot4)
	}

	// Update all planets using RK4 weighted average
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
		s.step(dt * s.TimeSpeed)
	}
}

func (s *Simulator) SetSunMass(massMultiplier float64) {
	s.SunMass = s.DefaultMass * massMultiplier
}

func (s *Simulator) ClearTrails() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.Planets {
		s.Planets[i].Trail = make([]Vec3, 0, s.maxTrailLen)
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
			Trail:    make([]Vec3, len(s.Planets[i].Trail)),
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
