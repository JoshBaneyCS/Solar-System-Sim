package validation

import (
	"fmt"
	"math"

	"solar-system-sim/internal/physics"
	"solar-system-sim/pkg/constants"
)

// expectedPeriod returns the theoretical Kepler period in days for a planet.
func expectedPeriod(planetName string) float64 {
	switch planetName {
	case "Mercury":
		return 87.969
	case "Venus":
		return 224.701
	case "Earth":
		return 365.256
	case "Mars":
		return 686.980
	default:
		return 0
	}
}

// ValidateKeplerPeriod measures the orbital period of a planet by tracking
// when it completes full orbits (angle crosses 2π) and compares to theory.
func ValidateKeplerPeriod(planetName string, years float64) *Result {
	sim := physics.NewSimulator()
	sim.PlanetGravityEnabled = false
	sim.RelativisticEffects = false

	planetIdx := -1
	for i, p := range sim.Planets {
		if p.Name == planetName {
			planetIdx = i
			break
		}
	}
	if planetIdx < 0 {
		return &Result{
			Scenario: fmt.Sprintf("Kepler Period (%s)", planetName),
			Pass:     false,
			Details:  fmt.Sprintf("planet %q not found", planetName),
		}
	}

	totalSeconds := years * 365.25 * 24 * 3600
	maxSteps := int(totalSeconds / constants.BaseTimeStep)

	// Track cumulative angle in the orbital plane using atan2 on projected x,y
	pos0 := sim.Planets[planetIdx].Position
	prevAngle := math.Atan2(pos0.Y, pos0.X)
	cumulativeAngle := 0.0

	var orbitCompletionTimes []float64

	for step := 0; step < maxSteps; step++ {
		sim.Step(constants.BaseTimeStep)

		pos := sim.Planets[planetIdx].Position
		angle := math.Atan2(pos.Y, pos.X)

		// Compute angle change, handling wrap-around
		dAngle := angle - prevAngle
		if dAngle > math.Pi {
			dAngle -= 2 * math.Pi
		} else if dAngle < -math.Pi {
			dAngle += 2 * math.Pi
		}

		cumulativeAngle += dAngle
		prevAngle = angle

		// Detect orbit completion: each time cumulative angle crosses a multiple of 2π
		orbitNum := int(cumulativeAngle / (2 * math.Pi))
		if orbitNum > len(orbitCompletionTimes) {
			orbitCompletionTimes = append(orbitCompletionTimes, sim.CurrentTime)
		}
	}

	if len(orbitCompletionTimes) < 1 {
		return &Result{
			Scenario: fmt.Sprintf("Kepler Period (%s)", planetName),
			Pass:     false,
			Details:  fmt.Sprintf("no complete orbits in %.1f years", years),
		}
	}

	// Measure period: time for all detected orbits
	nOrbits := float64(len(orbitCompletionTimes))
	totalTime := orbitCompletionTimes[len(orbitCompletionTimes)-1]
	measuredDays := (totalTime / nOrbits) / 86400.0

	expected := expectedPeriod(planetName)
	relErr := math.Abs(measuredDays-expected) / expected

	tolerance := 0.01 // 1%
	return &Result{
		Scenario:  fmt.Sprintf("Kepler Period (%s)", planetName),
		Pass:      relErr < tolerance,
		Measured:  measuredDays,
		Expected:  expected,
		Tolerance: tolerance,
		Units:     "days",
		Details:   fmt.Sprintf("relative error: %.4f%%, %d orbits detected", relErr*100, int(nOrbits)),
	}
}
