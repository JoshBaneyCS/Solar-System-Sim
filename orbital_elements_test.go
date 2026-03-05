package main

import (
	"math"
	"testing"
)

func TestCircularOrbit(t *testing.T) {
	sim := NewSimulator()
	body := sim.createPlanetFromElements(Planet{
		Name:           "TestBody",
		Mass:           1e24,
		SemiMajorAxis:  1.0, // 1 AU
		Eccentricity:   0.0, // circular
		Inclination:    0.0,
		InitialAnomaly: 0.0,
	})

	// For circular orbit with e=0, nu=0: position should be at (a, 0, 0)
	expectedR := 1.0 * AU
	assertFloat64Near(t, body.Position.X, expectedR, expectedR*1e-10, "X position")
	assertFloat64Near(t, body.Position.Y, 0, 1e-3, "Y position")
	assertFloat64Near(t, body.Position.Z, 0, 1e-3, "Z position")

	// For circular orbit (e=0), the code's velocity formula reduces to the standard
	// v_circular = sqrt(GM/a) since h/r = sqrt(GM*a)/a = sqrt(GM/a) when e=0
	GM := G * sim.SunMass
	vCircular := math.Sqrt(GM / expectedR)
	assertFloat64Near(t, body.Velocity.X, 0, vCircular*1e-10, "Vx")
	assertRelativeError(t, body.Velocity.Y, vCircular, 1e-10, "Vy")
	assertFloat64Near(t, body.Velocity.Z, 0, 1e-3, "Vz")
}

func TestOrbitalRadius(t *testing.T) {
	sim := NewSimulator()

	for _, pData := range planetData {
		t.Run(pData.Name, func(t *testing.T) {
			body := sim.createPlanetFromElements(pData)

			a := pData.SemiMajorAxis * AU
			e := pData.Eccentricity
			nu := pData.InitialAnomaly

			// Verify r = a(1-e²)/(1+e·cos(ν))
			expectedR := a * (1 - e*e) / (1 + e*math.Cos(nu))
			gotR := body.Position.Magnitude()

			assertRelativeError(t, gotR, expectedR, 1e-8, "orbital radius")
		})
	}
}

func TestInclinedOrbit(t *testing.T) {
	sim := NewSimulator()

	body := sim.createPlanetFromElements(Planet{
		Name:           "Inclined",
		Mass:           1e24,
		SemiMajorAxis:  1.0,
		Eccentricity:   0.01,
		Inclination:    30.0, // 30 degrees
		InitialAnomaly: math.Pi / 4,
	})

	// With 30° inclination and non-zero anomaly, Z should be non-zero
	if math.Abs(body.Position.Z) < 1e6 {
		t.Errorf("expected non-zero Z for inclined orbit, got Z = %e", body.Position.Z)
	}
	if math.Abs(body.Velocity.Z) < 1e-3 {
		t.Errorf("expected non-zero Vz for inclined orbit, got Vz = %e", body.Velocity.Z)
	}
}

func TestVelocityFormula(t *testing.T) {
	// Verify the code's velocity formula: vx = -h*sin(nu)/r, vy = h*(e+cos(nu))/r
	// This is the code's implementation, which we baseline here.
	sim := NewSimulator()
	GM := G * sim.SunMass

	for _, pData := range planetData {
		t.Run(pData.Name, func(t *testing.T) {
			a := pData.SemiMajorAxis * AU
			e := pData.Eccentricity
			nu := pData.InitialAnomaly

			r := a * (1 - e*e) / (1 + e*math.Cos(nu))
			h := math.Sqrt(GM * a * (1 - e*e))

			// Expected velocities in orbital plane from code's formula
			expectedVxOrb := -h * math.Sin(nu) / r
			expectedVyOrb := h * (e + math.Cos(nu)) / r

			body := sim.createPlanetFromElements(pData)

			// For zero inclination and zero node/perihelion, orbital plane = XY plane
			// For non-zero angles, we verify magnitude instead
			expectedSpeed := math.Sqrt(expectedVxOrb*expectedVxOrb + expectedVyOrb*expectedVyOrb)
			gotSpeed := body.Velocity.Magnitude()

			assertRelativeError(t, gotSpeed, expectedSpeed, 1e-8, "speed magnitude")
		})
	}
}

func TestRotationsPreserveMagnitude(t *testing.T) {
	sim := NewSimulator()

	for _, pData := range planetData {
		t.Run(pData.Name, func(t *testing.T) {
			a := pData.SemiMajorAxis * AU
			e := pData.Eccentricity
			nu := pData.InitialAnomaly

			// Orbital plane position magnitude
			r := a * (1 - e*e) / (1 + e*math.Cos(nu))

			body := sim.createPlanetFromElements(pData)

			// 3D rotations should preserve distance from origin
			assertRelativeError(t, body.Position.Magnitude(), r, 1e-8, "position magnitude preserved")
		})
	}
}

func TestZeroInclinationInEcliptic(t *testing.T) {
	sim := NewSimulator()

	// Earth has 0° inclination - should remain in ecliptic plane (Z ≈ 0)
	// Note: argument of perihelion rotation still applies in XY plane
	body := sim.createPlanetFromElements(Planet{
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
