package gr

import (
	"solar-system-sim/internal/math3d"
	"solar-system-sim/pkg/constants"
)

// CalculateGRCorrection computes the general relativistic perihelion precession correction.
// Formula: a_GR = (3G²M²)/(c²r³L) · (L × r)
func CalculateGRCorrection(bodyPos, bodyVel math3d.Vec3, sunMass, distanceToSun float64) math3d.Vec3 {
	rVec := bodyPos
	vVec := bodyVel
	LVec := rVec.Cross(vVec)
	LMag := LVec.Magnitude()

	if LMag < 1e-10 {
		return math3d.Vec3{}
	}

	c := constants.C
	return LVec.Cross(rVec).Mul(
		3 * constants.G * constants.G * sunMass * sunMass /
			(c * c * distanceToSun * distanceToSun * distanceToSun * LMag),
	)
}
