package launch

import "math"

// CircularVelocity returns the orbital velocity for a circular orbit at radius r
// around a body with gravitational parameter mu.
func CircularVelocity(mu, r float64) float64 {
	return math.Sqrt(mu / r)
}

// EscapeVelocity returns the escape velocity at radius r from a body with
// gravitational parameter mu.
func EscapeVelocity(mu, r float64) float64 {
	return math.Sqrt(2.0 * mu / r)
}

// HohmannDeltaV computes the two delta-v burns for a Hohmann transfer between
// circular orbits at radii r1 and r2. Returns (dv1, dv2) in m/s.
func HohmannDeltaV(mu, r1, r2 float64) (dv1, dv2 float64) {
	a := (r1 + r2) / 2.0

	v1 := math.Sqrt(mu / r1)
	vt1 := math.Sqrt(mu * (2.0/r1 - 1.0/a))
	dv1 = math.Abs(vt1 - v1)

	v2 := math.Sqrt(mu / r2)
	vt2 := math.Sqrt(mu * (2.0/r2 - 1.0/a))
	dv2 = math.Abs(v2 - vt2)

	return dv1, dv2
}

// HohmannTransferTime returns the time of flight for a Hohmann transfer
// between circular orbits at radii r1 and r2 (seconds).
func HohmannTransferTime(mu, r1, r2 float64) float64 {
	a := (r1 + r2) / 2.0
	return math.Pi * math.Sqrt(a*a*a/mu)
}

// PlaneChangeDV returns the delta-v required for a simple plane change
// at velocity v with inclination change deltaInc (radians).
func PlaneChangeDV(v, deltaInc float64) float64 {
	return 2.0 * v * math.Sin(deltaInc/2.0)
}

// HyperbolicExcessDV computes the delta-v to achieve a hyperbolic excess
// velocity vInf from a circular parking orbit at radius r around a body
// with gravitational parameter mu.
func HyperbolicExcessDV(mu, r, vInf float64) float64 {
	vCirc := CircularVelocity(mu, r)
	vDepart := math.Sqrt(vInf*vInf + 2.0*mu/r)
	return math.Abs(vDepart - vCirc)
}

// VisViva returns the orbital velocity at radius r for an orbit with
// semi-major axis a around a body with gravitational parameter mu.
func VisViva(mu, r, a float64) float64 {
	return math.Sqrt(mu * (2.0/r - 1.0/a))
}
