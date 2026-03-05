package validation

import (
	"fmt"
	"math"

	"solar-system-sim/internal/math3d"
	"solar-system-sim/internal/physics"
	"solar-system-sim/pkg/constants"
)

// ValidateMercuryPrecession measures Mercury's perihelion precession rate
// with GR enabled and compares to the expected ~43 arcsec/century.
//
// Method: track the Laplace-Runge-Lenz (eccentricity) vector, which points
// toward perihelion. Its angular drift over time gives the precession rate.
// Run with and without GR, subtract to isolate the GR component.
func ValidateMercuryPrecession(years float64) *Result {
	if years < 1 {
		years = 1
	}

	grRate := measurePrecessionLRL(true, years)
	newtonRate := measurePrecessionLRL(false, years)

	// GR-only precession rate (arcsec/century)
	grOnlyRate := (grRate - newtonRate) * 100.0

	// Expected: ~43 arcsec/century
	expected := 43.0
	tolerance := 0.7 // allow range roughly 13-73 arcsec/century

	pass := math.Abs(grOnlyRate-expected)/expected < tolerance

	return &Result{
		Scenario:  "Mercury Perihelion Precession (GR)",
		Pass:      pass,
		Measured:  grOnlyRate,
		Expected:  expected,
		Tolerance: tolerance,
		Units:     "arcsec/century",
		Details: fmt.Sprintf(
			"GR+Newton=%.4f arcsec/yr, Newton-only=%.4f arcsec/yr, GR-only=%.2f arcsec/century (over %.0f years)",
			grRate, newtonRate, grOnlyRate, years),
	}
}

// lrlVector computes the Laplace-Runge-Lenz (eccentricity) vector for a
// body orbiting the Sun. This vector points toward perihelion and its
// magnitude equals the eccentricity.
//
// A = (v × L) / (GM) - r_hat
// where L = r × v (specific angular momentum)
func lrlVector(pos, vel math3d.Vec3, GM float64) math3d.Vec3 {
	L := pos.Cross(vel) // specific angular momentum
	vCrossL := vel.Cross(L)
	rHat := pos.Normalize()
	return vCrossL.Mul(1.0 / GM).Sub(rHat)
}

// measurePrecessionLRL runs the simulation and measures the precession rate
// of Mercury's perihelion using the Laplace-Runge-Lenz vector.
func measurePrecessionLRL(grEnabled bool, years float64) float64 {
	sim := physics.NewSimulator()
	sim.PlanetGravityEnabled = true
	sim.RelativisticEffects = grEnabled

	mercuryIdx := 0
	GM := constants.G * sim.SunMass

	totalSeconds := years * 365.25 * 24 * 3600
	steps := int(totalSeconds / constants.BaseTimeStep)

	// Sample the LRL vector angle at regular intervals and use linear
	// regression to extract the precession rate
	sampleInterval := 100 // every 100 steps
	var times []float64
	var angles []float64

	// Get initial LRL vector angle
	pos0 := sim.Planets[mercuryIdx].Position
	vel0 := sim.Planets[mercuryIdx].Velocity
	A0 := lrlVector(pos0, vel0, GM)
	prevAngle := math.Atan2(A0.Y, A0.X)
	cumAngle := 0.0

	for step := 0; step < steps; step++ {
		sim.Step(constants.BaseTimeStep)

		if (step+1)%sampleInterval == 0 {
			pos := sim.Planets[mercuryIdx].Position
			vel := sim.Planets[mercuryIdx].Velocity
			A := lrlVector(pos, vel, GM)
			angle := math.Atan2(A.Y, A.X)

			// Unwrap angle
			dAngle := angle - prevAngle
			if dAngle > math.Pi {
				dAngle -= 2 * math.Pi
			} else if dAngle < -math.Pi {
				dAngle += 2 * math.Pi
			}
			cumAngle += dAngle
			prevAngle = angle

			times = append(times, sim.CurrentTime)
			angles = append(angles, cumAngle)
		}
	}

	if len(times) < 2 {
		return 0
	}

	// Linear regression: angle = slope * time + intercept
	n := float64(len(times))
	var sumT, sumA, sumTT, sumTA float64
	for i := range times {
		sumT += times[i]
		sumA += angles[i]
		sumTT += times[i] * times[i]
		sumTA += times[i] * angles[i]
	}
	slope := (n*sumTA - sumT*sumA) / (n*sumTT - sumT*sumT)

	// Convert rad/s to arcsec/yr
	arcsecPerRad := 180.0 / math.Pi * 3600.0
	secondsPerYear := 365.25 * 24 * 3600.0

	return slope * arcsecPerRad * secondsPerYear
}
