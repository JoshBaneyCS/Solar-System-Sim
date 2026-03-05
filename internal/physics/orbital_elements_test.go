package physics

import (
	"math"
	"testing"

	"solar-system-sim/pkg/constants"
)

func TestCircularOrbit(t *testing.T) {
	sim := NewSimulator()
	body := sim.CreatePlanetFromElements(Planet{
		Name:           "TestBody",
		Mass:           1e24,
		SemiMajorAxis:  1.0,
		Eccentricity:   0.0,
		Inclination:    0.0,
		InitialAnomaly: 0.0,
	})

	expectedR := 1.0 * constants.AU
	assertFloat64Near(t, body.Position.X, expectedR, expectedR*1e-10, "X position")
	assertFloat64Near(t, body.Position.Y, 0, 1e-3, "Y position")
	assertFloat64Near(t, body.Position.Z, 0, 1e-3, "Z position")

	GM := constants.G * sim.SunMass
	vCircular := math.Sqrt(GM / expectedR)
	assertFloat64Near(t, body.Velocity.X, 0, vCircular*1e-10, "Vx")
	assertRelativeError(t, body.Velocity.Y, vCircular, 1e-10, "Vy")
	assertFloat64Near(t, body.Velocity.Z, 0, 1e-3, "Vz")
}

func TestOrbitalRadius(t *testing.T) {
	sim := NewSimulator()

	for _, pData := range PlanetData {
		t.Run(pData.Name, func(t *testing.T) {
			body := sim.CreatePlanetFromElements(pData)

			a := pData.SemiMajorAxis * constants.AU
			e := pData.Eccentricity
			nu := pData.InitialAnomaly

			expectedR := a * (1 - e*e) / (1 + e*math.Cos(nu))
			gotR := body.Position.Magnitude()

			assertRelativeError(t, gotR, expectedR, 1e-8, "orbital radius")
		})
	}
}

func TestInclinedOrbit(t *testing.T) {
	sim := NewSimulator()

	body := sim.CreatePlanetFromElements(Planet{
		Name:           "Inclined",
		Mass:           1e24,
		SemiMajorAxis:  1.0,
		Eccentricity:   0.01,
		Inclination:    30.0,
		InitialAnomaly: math.Pi / 4,
	})

	if math.Abs(body.Position.Z) < 1e6 {
		t.Errorf("expected non-zero Z for inclined orbit, got Z = %e", body.Position.Z)
	}
	if math.Abs(body.Velocity.Z) < 1e-3 {
		t.Errorf("expected non-zero Vz for inclined orbit, got Vz = %e", body.Velocity.Z)
	}
}

func TestVelocityFormula(t *testing.T) {
	sim := NewSimulator()
	GM := constants.G * sim.SunMass

	for _, pData := range PlanetData {
		t.Run(pData.Name, func(t *testing.T) {
			a := pData.SemiMajorAxis * constants.AU
			e := pData.Eccentricity
			nu := pData.InitialAnomaly

			h := math.Sqrt(GM * a * (1 - e*e))

			muOverH := GM / h
			expectedVxOrb := -muOverH * math.Sin(nu)
			expectedVyOrb := muOverH * (e + math.Cos(nu))

			body := sim.CreatePlanetFromElements(pData)

			expectedSpeed := math.Sqrt(expectedVxOrb*expectedVxOrb + expectedVyOrb*expectedVyOrb)
			gotSpeed := body.Velocity.Magnitude()

			assertRelativeError(t, gotSpeed, expectedSpeed, 1e-8, "speed magnitude")
		})
	}
}

func TestRotationsPreserveMagnitude(t *testing.T) {
	sim := NewSimulator()

	for _, pData := range PlanetData {
		t.Run(pData.Name, func(t *testing.T) {
			a := pData.SemiMajorAxis * constants.AU
			e := pData.Eccentricity
			nu := pData.InitialAnomaly

			r := a * (1 - e*e) / (1 + e*math.Cos(nu))

			body := sim.CreatePlanetFromElements(pData)

			assertRelativeError(t, body.Position.Magnitude(), r, 1e-8, "position magnitude preserved")
		})
	}
}

func TestZeroInclinationInEcliptic(t *testing.T) {
	sim := NewSimulator()

	body := sim.CreatePlanetFromElements(Planet{
		Name:           "Flat",
		Mass:           1e24,
		SemiMajorAxis:  1.0,
		Eccentricity:   0.01,
		Inclination:    0.0,
		InitialAnomaly: math.Pi / 3,
	})

	assertFloat64Near(t, body.Position.Z, 0, 1e-3, "Z should be 0 for zero inclination")
	assertFloat64Near(t, body.Velocity.Z, 0, 1e-6, "Vz should be 0 for zero inclination")
}
