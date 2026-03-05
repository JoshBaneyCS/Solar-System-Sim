package launch

import "solar-system-sim/internal/math3d"

// TrajectoryPoint represents a single point along a trajectory.
type TrajectoryPoint struct {
	Time     float64    // seconds since launch
	Position math3d.Vec3 // meters
	Velocity math3d.Vec3 // m/s
}

// Trajectory is an ordered sequence of trajectory points.
type Trajectory struct {
	Points []TrajectoryPoint
	Frame  ReferenceFrame
}

// ToHeliocentric converts an Earth-centered trajectory to heliocentric
// by adding Earth's position to each point.
func (t *Trajectory) ToHeliocentric(earthPos math3d.Vec3) *Trajectory {
	if t.Frame == Heliocentric {
		return t
	}
	result := &Trajectory{
		Points: make([]TrajectoryPoint, len(t.Points)),
		Frame:  Heliocentric,
	}
	for i, p := range t.Points {
		result.Points[i] = TrajectoryPoint{
			Time:     p.Time,
			Position: p.Position.Add(earthPos),
			Velocity: p.Velocity,
		}
	}
	return result
}

// ToEarthCentered converts a heliocentric trajectory to Earth-centered
// by subtracting Earth's position from each point.
func (t *Trajectory) ToEarthCentered(earthPos math3d.Vec3) *Trajectory {
	if t.Frame == EarthCentered {
		return t
	}
	result := &Trajectory{
		Points: make([]TrajectoryPoint, len(t.Points)),
		Frame:  EarthCentered,
	}
	for i, p := range t.Points {
		result.Points[i] = TrajectoryPoint{
			Time:     p.Time,
			Position: p.Position.Sub(earthPos),
			Velocity: p.Velocity,
		}
	}
	return result
}
