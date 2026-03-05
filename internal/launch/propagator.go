package launch

import (
	"solar-system-sim/internal/math3d"
)

// PropagatorConfig controls numerical trajectory propagation.
type PropagatorConfig struct {
	Mu       float64 // gravitational parameter of central body
	TimeStep float64 // integration timestep (seconds)
	Duration float64 // total propagation time (seconds)
	MaxSteps int     // safety limit on number of steps
}

// Propagate generates a trajectory using RK4 integration for a body
// in a central force field (2-body problem).
func Propagate(pos, vel math3d.Vec3, cfg PropagatorConfig) *Trajectory {
	traj := &Trajectory{
		Frame: EarthCentered,
	}

	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 100000
	}

	dt := cfg.TimeStep
	t := 0.0
	p := pos
	v := vel

	// Record interval: store ~1000 points max for rendering
	recordInterval := 1
	totalSteps := int(cfg.Duration / dt)
	if totalSteps > 1000 {
		recordInterval = totalSteps / 1000
	}

	step := 0
	for t < cfg.Duration && step < cfg.MaxSteps {
		if step%recordInterval == 0 {
			traj.Points = append(traj.Points, TrajectoryPoint{
				Time:     t,
				Position: p,
				Velocity: v,
			})
		}

		p, v = rk4Step(p, v, dt, cfg.Mu)
		t += dt
		step++
	}

	// Always record the final point
	traj.Points = append(traj.Points, TrajectoryPoint{
		Time:     t,
		Position: p,
		Velocity: v,
	})

	return traj
}

// rk4Step performs a single RK4 integration step for a 2-body problem.
func rk4Step(pos, vel math3d.Vec3, dt, mu float64) (math3d.Vec3, math3d.Vec3) {
	accel := func(p math3d.Vec3) math3d.Vec3 {
		r := p.Magnitude()
		if r < 1e3 {
			return math3d.Vec3{}
		}
		return p.Normalize().Mul(-mu / (r * r))
	}

	// k1
	a1 := accel(pos)
	k1v := vel
	k1a := a1

	// k2
	p2 := pos.Add(k1v.Mul(dt / 2))
	v2 := vel.Add(k1a.Mul(dt / 2))
	a2 := accel(p2)
	k2v := v2
	k2a := a2

	// k3
	p3 := pos.Add(k2v.Mul(dt / 2))
	v3 := vel.Add(k2a.Mul(dt / 2))
	a3 := accel(p3)
	k3v := v3
	k3a := a3

	// k4
	p4 := pos.Add(k3v.Mul(dt))
	v4 := vel.Add(k3a.Mul(dt))
	a4 := accel(p4)
	k4v := v4
	k4a := a4

	newPos := pos.Add(k1v.Add(k2v.Mul(2)).Add(k3v.Mul(2)).Add(k4v).Mul(dt / 6))
	newVel := vel.Add(k1a.Add(k2a.Mul(2)).Add(k3a.Mul(2)).Add(k4a).Mul(dt / 6))

	return newPos, newVel
}
