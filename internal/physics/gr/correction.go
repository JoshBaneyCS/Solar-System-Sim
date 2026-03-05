package gr

import (
	"solar-system-sim/internal/math3d"
	"solar-system-sim/pkg/constants"
)

// CalculateGRCorrection computes the first post-Newtonian (1PN) correction
// to the acceleration of a body orbiting the Sun.
//
// Standard 1PN expression for a test particle in a Schwarzschild field:
//   a_GR = (GM/(c²r³)) * [(4GM/r - v²)r + 4(r·v)v]
//
// This produces Mercury's perihelion precession of ~43 arcsec/century.
func CalculateGRCorrection(bodyPos, bodyVel math3d.Vec3, sunMass, distanceToSun float64) math3d.Vec3 {
	GM := constants.G * sunMass
	c := constants.C
	r := distanceToSun
	v2 := bodyVel.Dot(bodyVel)
	rdotv := bodyPos.Dot(bodyVel)

	coeff := GM / (c * c * r * r * r)

	// (4GM/r - v²) * r_vec + 4(r·v) * v_vec
	term1 := bodyPos.Mul(4*GM/r - v2)
	term2 := bodyVel.Mul(4 * rdotv)

	return term1.Add(term2).Mul(coeff)
}
