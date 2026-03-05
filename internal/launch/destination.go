package launch

import "math"

// ReferenceFrame indicates the coordinate frame for a destination.
type ReferenceFrame int

const (
	EarthCentered ReferenceFrame = iota
	Heliocentric
)

// Destination represents a target orbit for a launch.
type Destination struct {
	Name          string
	Altitude      float64 // meters above Earth surface (Earth-centered)
	ApoapsisAlt   float64 // meters above Earth surface (for elliptical)
	SemiMajorAxis float64 // meters (heliocentric)
	Inclination   float64 // radians
	Eccentricity  float64
	Frame         ReferenceFrame
}

// Destinations is the catalog of available destination presets.
var Destinations = map[string]Destination{
	"leo": {
		Name:        "LEO (200 km)",
		Altitude:    200e3,
		ApoapsisAlt: 200e3,
		Inclination: KSCLatitudeRad,
		Frame:       EarthCentered,
	},
	"iss": {
		Name:        "ISS Orbit",
		Altitude:    408e3,
		ApoapsisAlt: 408e3,
		Inclination: 51.6 * math.Pi / 180.0,
		Frame:       EarthCentered,
	},
	"gto": {
		Name:        "GTO",
		Altitude:    200e3,
		ApoapsisAlt: GEOAltitude,
		Inclination: KSCLatitudeRad,
		Frame:       EarthCentered,
	},
	"moon": {
		Name:        "Moon Transfer (TLI)",
		Altitude:    200e3,
		ApoapsisAlt: MoonDistance,
		Inclination: KSCLatitudeRad,
		Frame:       EarthCentered,
	},
	"mars": {
		Name:          "Mars Transfer (Hohmann)",
		Altitude:      200e3,
		SemiMajorAxis: MarsOrbitSMA,
		Inclination:   1.85 * math.Pi / 180.0,
		Frame:         Heliocentric,
	},
}

// GetDestination returns a destination by key name. Returns LEO if not found.
func GetDestination(name string) Destination {
	if d, ok := Destinations[name]; ok {
		return d
	}
	return Destinations["leo"]
}

// DestinationNames returns the available destination keys in display order.
func DestinationNames() []string {
	return []string{"leo", "iss", "gto", "moon", "mars"}
}

// DestinationDisplayNames returns display names in the same order as DestinationNames.
func DestinationDisplayNames() []string {
	keys := DestinationNames()
	names := make([]string, len(keys))
	for i, k := range keys {
		names[i] = Destinations[k].Name
	}
	return names
}
