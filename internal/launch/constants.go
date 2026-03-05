package launch

import "math"

const (
	// MuEarth is the standard gravitational parameter of Earth (m^3/s^2)
	MuEarth = 3.986004418e14

	// MuSun is the standard gravitational parameter of the Sun (m^3/s^2)
	MuSun = 1.32712440018e20

	// REarth is the mean radius of Earth (m)
	REarth = 6.371e6

	// G0 is standard gravity at sea level (m/s^2)
	G0 = 9.80665

	// KSCLatitude is Kennedy Space Center latitude (degrees)
	KSCLatitude = 28.5724

	// KSCLatitudeRad is KSC latitude in radians
	KSCLatitudeRad = KSCLatitude * math.Pi / 180.0

	// EarthRotationalVelocity is Earth's surface velocity at KSC latitude (m/s)
	EarthRotationalVelocity = 407.0

	// GravityDragLoss is the typical combined gravity and drag loss for Earth ascent (m/s)
	GravityDragLoss = 1500.0

	// GEOAltitude is geostationary orbit altitude (m)
	GEOAltitude = 35786e3

	// MoonDistance is the average Earth-Moon distance (m)
	MoonDistance = 384400e3

	// MarsOrbitSMA is Mars semi-major axis (m)
	MarsOrbitSMA = 1.524 * 1.496e11

	// EarthOrbitSMA is Earth semi-major axis (m)
	EarthOrbitSMA = 1.496e11
)
