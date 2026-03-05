package validation

import (
	"testing"
)

func TestEnergyScenario(t *testing.T) {
	r := ValidateEnergyConservation(1.0)
	t.Log(r.String())
	if !r.Pass {
		t.Errorf("energy conservation failed: %s", r.Details)
	}
}

func TestAngularMomentumScenario(t *testing.T) {
	r := ValidateAngularMomentumConservation(1.0)
	t.Log(r.String())
	if !r.Pass {
		t.Errorf("angular momentum conservation failed: %s", r.Details)
	}
}

func TestKeplerEarth(t *testing.T) {
	r := ValidateKeplerPeriod("Earth", 2.0)
	t.Log(r.String())
	if !r.Pass {
		t.Errorf("Kepler period (Earth) failed: %s", r.Details)
	}
}

func TestKeplerMercury(t *testing.T) {
	r := ValidateKeplerPeriod("Mercury", 1.0)
	t.Log(r.String())
	if !r.Pass {
		t.Errorf("Kepler period (Mercury) failed: %s", r.Details)
	}
}

func TestMercuryPrecession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Mercury precession test in short mode")
	}

	// Run for 10 years — enough for ~41 Mercury orbits
	r := ValidateMercuryPrecession(10.0)
	t.Log(r.String())
	if !r.Pass {
		t.Errorf("Mercury precession failed: measured=%.2f arcsec/century, expected=%.2f +/- %.0f%%",
			r.Measured, r.Expected, r.Tolerance*100)
	}
}
