package physics

import (
	"image/color"

	"solar-system-sim/internal/math3d"
)

// BodyType categorizes celestial bodies for rendering and UI grouping.
type BodyType int

const (
	BodyTypeStar BodyType = iota
	BodyTypePlanet
	BodyTypeDwarfPlanet
	BodyTypeMoon
	BodyTypeComet
	BodyTypeAsteroid
)

// Body represents a celestial body
type Body struct {
	Name           string
	Mass           float64 // kg
	Position       math3d.Vec3
	Velocity       math3d.Vec3
	Color          color.Color
	Radius         float64 // display radius in pixels
	Trail          []math3d.Vec3
	ShowTrail      bool
	Type           BodyType
	PhysicalRadius float64 // real radius in meters (for zoom-to-fill rendering)
}

// Planet holds real solar system data
type Planet struct {
	Name                   string
	Mass                   float64 // kg
	SemiMajorAxis          float64 // AU (heliocentric) or meters (for moons, relative to parent)
	Eccentricity           float64
	Inclination            float64 // degrees
	LongitudeAscendingNode float64 // degrees
	ArgumentOfPerihelion   float64 // degrees
	OrbitalPeriod          float64 // Earth days
	Color                  color.Color
	DisplayRadius          float64
	InitialAnomaly         float64 // radians
	Type                   BodyType
	ParentName             string  // empty for heliocentric orbits; "Earth", "Jupiter", etc. for moons
	ParentMass             float64 // mass of parent body in kg (for moon orbit computation)
	PhysicalRadius         float64 // real radius in meters
}

// BodyState holds a snapshot of body position and velocity for RK4 integration
type BodyState struct {
	Position math3d.Vec3
	Velocity math3d.Vec3
}
