package launch

import (
	"math"
	"testing"
)

func assertApprox(t *testing.T, name string, got, want, tolerance float64) {
	t.Helper()
	rel := math.Abs(got-want) / want
	if rel > tolerance {
		t.Errorf("%s: got %.4f, want %.4f (rel error %.4f > %.4f)", name, got, want, rel, tolerance)
	}
}

func TestCircularVelocityLEO(t *testing.T) {
	r := REarth + 200e3
	v := CircularVelocity(MuEarth, r)
	// LEO at 200km should be ~7788 m/s
	assertApprox(t, "LEO circular velocity", v, 7788.0, 0.01)
}

func TestCircularVelocityGEO(t *testing.T) {
	r := REarth + GEOAltitude
	v := CircularVelocity(MuEarth, r)
	// GEO should be ~3075 m/s
	assertApprox(t, "GEO circular velocity", v, 3075.0, 0.01)
}

func TestEscapeVelocityEarth(t *testing.T) {
	v := EscapeVelocity(MuEarth, REarth)
	// Earth surface escape velocity ~11186 m/s
	assertApprox(t, "Earth escape velocity", v, 11186.0, 0.01)
}

func TestHohmannLEOtoGEO(t *testing.T) {
	r1 := REarth + 200e3
	r2 := REarth + GEOAltitude
	dv1, dv2 := HohmannDeltaV(MuEarth, r1, r2)

	// LEO to GEO: dv1 ~ 2.46 km/s, dv2 ~ 1.48 km/s
	assertApprox(t, "LEO-GEO dv1", dv1, 2460.0, 0.02)
	assertApprox(t, "LEO-GEO dv2", dv2, 1480.0, 0.02)
}

func TestHohmannTransferTimeEarthMars(t *testing.T) {
	tof := HohmannTransferTime(MuSun, EarthOrbitSMA, MarsOrbitSMA)
	days := tof / (24 * 3600)
	// Earth-Mars Hohmann transfer ~259 days
	assertApprox(t, "Earth-Mars transfer time", days, 259.0, 0.03)
}

func TestPlaneChange(t *testing.T) {
	v := 7788.0 // LEO velocity
	dInc := (51.6 - 28.57) * math.Pi / 180.0
	dv := PlaneChangeDV(v, dInc)
	// ~23 deg plane change at LEO ~ 3100 m/s
	assertApprox(t, "plane change dv", dv, 3100.0, 0.05)
}

func TestHyperbolicExcessDV(t *testing.T) {
	r := REarth + 200e3
	// TLI: hyperbolic excess ~0.9 km/s for Moon
	vInf := 900.0
	dv := HyperbolicExcessDV(MuEarth, r, vInf)
	// TLI burn from LEO ~ 3.13 km/s
	assertApprox(t, "TLI dv", dv, 3130.0, 0.05)
}
