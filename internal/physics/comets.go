package physics

import (
	"image/color"
	"math"
)

// CometData contains orbital elements for notable comets.
// All orbits are heliocentric. Comets have very high eccentricity.
var CometData = []Planet{
	{
		Name:                   "Halley",
		Mass:                   2.2e14,
		SemiMajorAxis:          17.834,
		Eccentricity:           0.96714,
		Inclination:            162.26,
		LongitudeAscendingNode: 58.42,
		ArgumentOfPerihelion:   111.33,
		OrbitalPeriod:          27510, // ~75.3 years
		Color:                  color.RGBA{200, 220, 255, 255},
		DisplayRadius:          3,
		InitialAnomaly:         math.Pi, // near aphelion
		Type:                   BodyTypeComet,
		PhysicalRadius:         5.5e3, // ~5.5 km nucleus
	},
	{
		Name:                   "Hale-Bopp",
		Mass:                   1.3e16,
		SemiMajorAxis:          186.0,
		Eccentricity:           0.99503,
		Inclination:            89.43,
		LongitudeAscendingNode: 282.47,
		ArgumentOfPerihelion:   130.59,
		OrbitalPeriod:          927775, // ~2520 years
		Color:                  color.RGBA{180, 200, 255, 255},
		DisplayRadius:          3,
		InitialAnomaly:         math.Pi * 0.8,
		Type:                   BodyTypeComet,
		PhysicalRadius:         30e3, // ~30 km nucleus
	},
	{
		Name:                   "Encke",
		Mass:                   7.0e13,
		SemiMajorAxis:          2.215,
		Eccentricity:           0.8471,
		Inclination:            11.78,
		LongitudeAscendingNode: 334.57,
		ArgumentOfPerihelion:   186.55,
		OrbitalPeriod:          1204, // ~3.3 years
		Color:                  color.RGBA{170, 190, 230, 255},
		DisplayRadius:          2,
		InitialAnomaly:         math.Pi / 6,
		Type:                   BodyTypeComet,
		PhysicalRadius:         2.4e3,
	},
	{
		Name:                   "Swift-Tuttle",
		Mass:                   5.0e14,
		SemiMajorAxis:          26.092,
		Eccentricity:           0.9632,
		Inclination:            113.45,
		LongitudeAscendingNode: 139.38,
		ArgumentOfPerihelion:   152.98,
		OrbitalPeriod:          48676, // ~133 years
		Color:                  color.RGBA{190, 210, 250, 255},
		DisplayRadius:          3,
		InitialAnomaly:         math.Pi * 0.6,
		Type:                   BodyTypeComet,
		PhysicalRadius:         13e3, // ~26 km diameter
	},
}
