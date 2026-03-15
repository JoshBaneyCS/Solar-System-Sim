package physics

import (
	"image/color"
	"math"
	"math/rand"
)

// AsteroidData contains orbital elements for major named asteroids.
// All orbits are heliocentric.
var AsteroidData = []Planet{
	// Main belt — major bodies
	{
		Name:                   "Ceres",
		Mass:                   9.3835e20,
		SemiMajorAxis:          2.7691,
		Eccentricity:           0.0758,
		Inclination:            10.594,
		LongitudeAscendingNode: 80.394,
		ArgumentOfPerihelion:   72.522,
		OrbitalPeriod:          1683,
		Color:                  color.RGBA{170, 160, 150, 255},
		DisplayRadius:          3,
		InitialAnomaly:         0,
		Type:                   BodyTypeDwarfPlanet,
		PhysicalRadius:         473e3,
	},
	{
		Name:                   "Vesta",
		Mass:                   2.5908e20,
		SemiMajorAxis:          2.3615,
		Eccentricity:           0.0887,
		Inclination:            7.134,
		LongitudeAscendingNode: 103.851,
		ArgumentOfPerihelion:   149.841,
		OrbitalPeriod:          1325,
		Color:                  color.RGBA{180, 170, 160, 255},
		DisplayRadius:          3,
		InitialAnomaly:         math.Pi / 3,
		Type:                   BodyTypeAsteroid,
		PhysicalRadius:         262.7e3,
	},
	{
		Name:                   "Pallas",
		Mass:                   2.1125e20,
		SemiMajorAxis:          2.7724,
		Eccentricity:           0.2313,
		Inclination:            34.832,
		LongitudeAscendingNode: 173.097,
		ArgumentOfPerihelion:   310.202,
		OrbitalPeriod:          1686,
		Color:                  color.RGBA{160, 155, 145, 255},
		DisplayRadius:          3,
		InitialAnomaly:         2 * math.Pi / 3,
		Type:                   BodyTypeAsteroid,
		PhysicalRadius:         256e3,
	},
	{
		Name:                   "Hygiea",
		Mass:                   8.32e19,
		SemiMajorAxis:          3.1421,
		Eccentricity:           0.1146,
		Inclination:            3.832,
		LongitudeAscendingNode: 283.519,
		ArgumentOfPerihelion:   312.364,
		OrbitalPeriod:          2034,
		Color:                  color.RGBA{140, 135, 130, 255},
		DisplayRadius:          2,
		InitialAnomaly:         math.Pi,
		Type:                   BodyTypeAsteroid,
		PhysicalRadius:         215e3,
	},

	// Trans-Neptunian dwarf planets
	{
		Name:                   "Pluto",
		Mass:                   1.303e22,
		SemiMajorAxis:          39.482,
		Eccentricity:           0.2488,
		Inclination:            17.16,
		LongitudeAscendingNode: 110.299,
		ArgumentOfPerihelion:   113.834,
		OrbitalPeriod:          90560,
		Color:                  color.RGBA{210, 190, 170, 255},
		DisplayRadius:          3,
		InitialAnomaly:         0,
		Type:                   BodyTypeDwarfPlanet,
		PhysicalRadius:         1.1883e6,
	},

	// Near-Earth asteroids
	{
		Name:                   "Apophis",
		Mass:                   6.1e10,
		SemiMajorAxis:          0.9224,
		Eccentricity:           0.1911,
		Inclination:            3.339,
		LongitudeAscendingNode: 204.446,
		ArgumentOfPerihelion:   126.393,
		OrbitalPeriod:          323.6,
		Color:                  color.RGBA{200, 100, 80, 255},
		DisplayRadius:          2,
		InitialAnomaly:         math.Pi / 4,
		Type:                   BodyTypeAsteroid,
		PhysicalRadius:         185,
	},
	{
		Name:                   "Bennu",
		Mass:                   7.329e10,
		SemiMajorAxis:          1.1264,
		Eccentricity:           0.2037,
		Inclination:            6.035,
		LongitudeAscendingNode: 2.061,
		ArgumentOfPerihelion:   66.223,
		OrbitalPeriod:          436.6,
		Color:                  color.RGBA{100, 90, 80, 255},
		DisplayRadius:          2,
		InitialAnomaly:         3 * math.Pi / 4,
		Type:                   BodyTypeAsteroid,
		PhysicalRadius:         245,
	},
}

// BeltParticle represents a visual-only asteroid belt particle (not N-body simulated).
type BeltParticle struct {
	SemiMajorAxis  float64 // AU
	Eccentricity   float64
	Inclination    float64 // radians
	InitialAnomaly float64 // radians
}

// GenerateBeltParticles creates a deterministic set of visual belt particles.
func GenerateBeltParticles(count int) []BeltParticle {
	rng := rand.New(rand.NewSource(42)) // deterministic seed
	particles := make([]BeltParticle, count)
	for i := range particles {
		// Main belt: 2.1 to 3.3 AU, with Kirkwood gaps at 2.5, 2.82, 2.95 AU
		a := 2.1 + rng.Float64()*1.2
		// Skip Kirkwood gap regions (approximate)
		for (a > 2.48 && a < 2.52) || (a > 2.80 && a < 2.84) || (a > 2.93 && a < 2.97) {
			a = 2.1 + rng.Float64()*1.2
		}
		particles[i] = BeltParticle{
			SemiMajorAxis:  a,
			Eccentricity:   rng.Float64() * 0.15,                       // 0 to 0.15
			Inclination:    (rng.Float64()*2 - 1) * 20 * math.Pi / 180, // -20 to +20 deg
			InitialAnomaly: rng.Float64() * 2 * math.Pi,
		}
	}
	return particles
}
