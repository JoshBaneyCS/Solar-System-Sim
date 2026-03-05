package validation

import (
	"fmt"
	"strings"
)

// Result holds the outcome of a validation scenario.
type Result struct {
	Scenario  string
	Pass      bool
	Measured  float64
	Expected  float64
	Tolerance float64
	Units     string
	Details   string
}

// String returns a human-readable summary of the result.
func (r *Result) String() string {
	status := "PASS"
	if !r.Pass {
		status = "FAIL"
	}
	s := fmt.Sprintf("[%s] %s\n  Measured:  %.6e %s\n  Expected:  %.6e %s\n  Tolerance: %.2e",
		status, r.Scenario, r.Measured, r.Units, r.Expected, r.Units, r.Tolerance)
	if r.Details != "" {
		s += "\n  Details: " + r.Details
	}
	return s
}

// AllScenarios returns the list of available scenario names.
func AllScenarios() []string {
	return []string{
		"energy",
		"angular-momentum",
		"kepler-earth",
		"kepler-mercury",
		"mercury-precession",
	}
}

// RunScenario executes a named validation scenario.
func RunScenario(name string, years float64) (*Result, error) {
	switch strings.ToLower(name) {
	case "energy":
		return ValidateEnergyConservation(years), nil
	case "angular-momentum":
		return ValidateAngularMomentumConservation(years), nil
	case "kepler-earth":
		return ValidateKeplerPeriod("Earth", years), nil
	case "kepler-mercury":
		return ValidateKeplerPeriod("Mercury", years), nil
	case "mercury-precession":
		return ValidateMercuryPrecession(years), nil
	default:
		return nil, fmt.Errorf("unknown scenario: %s", name)
	}
}

// RunAll executes all scenarios and returns results.
func RunAll(years float64) []*Result {
	var results []*Result
	for _, name := range AllScenarios() {
		r, _ := RunScenario(name, years)
		results = append(results, r)
	}
	return results
}
