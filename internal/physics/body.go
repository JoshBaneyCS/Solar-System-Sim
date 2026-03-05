package physics

import (
	"image/color"

	"solar-system-sim/internal/math3d"
)

// Body represents a celestial body
type Body struct {
	Name      string
	Mass      float64 // kg
	Position  math3d.Vec3
	Velocity  math3d.Vec3
	Color     color.Color
	Radius    float64 // display radius in pixels
	Trail     []math3d.Vec3
	ShowTrail bool
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

// BodyState holds a snapshot of body position and velocity for RK4 integration
type BodyState struct {
	Position math3d.Vec3
	Velocity math3d.Vec3
}
