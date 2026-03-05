package launch

import "math"

// RocketDeltaV computes the delta-v for a single stage using the Tsiolkovsky
// rocket equation: dv = Isp * g0 * ln(m0 / mf)
func RocketDeltaV(isp, m0, mf float64) float64 {
	if mf <= 0 || m0 <= mf {
		return 0
	}
	return isp * G0 * math.Log(m0/mf)
}

// StageDeltaV computes the delta-v for a single stage, given a payload mass
// riding on top. m0 = stage.WetMass + payload, mf = stage.DryMass + payload.
func StageDeltaV(s Stage, payloadMass float64) float64 {
	m0 := s.WetMass + payloadMass
	mf := s.DryMass + payloadMass
	return RocketDeltaV(s.Isp, m0, mf)
}

// TotalVehicleDeltaV computes the total delta-v for a multi-stage vehicle
// by chaining stages (first stage carries all upper stages as payload).
func TotalVehicleDeltaV(v Vehicle) float64 {
	total := 0.0
	for i := 0; i < len(v.Stages); i++ {
		// Payload for stage i = sum of all upper stages' wet mass
		payload := 0.0
		for j := i + 1; j < len(v.Stages); j++ {
			payload += v.Stages[j].WetMass
		}
		total += StageDeltaV(v.Stages[i], payload)
	}
	return total
}
