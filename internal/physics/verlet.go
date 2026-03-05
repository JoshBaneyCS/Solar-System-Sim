package physics

import "solar-system-sim/internal/math3d"

// stepVerlet performs one Velocity Verlet integration step.
//
// The algorithm is:
//  1. v(t + dt/2) = v(t) + a(t) * dt/2
//  2. x(t + dt)  = x(t) + v(t + dt/2) * dt
//  3. a(t + dt)  = acceleration(x(t + dt))
//  4. v(t + dt)  = v(t + dt/2) + a(t + dt) * dt/2
//
// Velocity Verlet is symplectic, meaning it conserves energy much better
// than RK4 over long integration times.
func (s *Simulator) stepVerlet(dt float64) {
	n := len(s.Planets)

	// Build snapshot of current state for acceleration computation
	states := make([]BodyState, n)
	for i := range s.Planets {
		states[i] = BodyState{
			Position: s.Planets[i].Position,
			Velocity: s.Planets[i].Velocity,
		}
	}

	// Step 1: Compute current accelerations
	accel := make([]math3d.Vec3, n)
	for i := range s.Planets {
		accel[i] = s.CalculateAccelerationWithSnapshot(i,
			s.Planets[i].Position, s.Planets[i].Velocity,
			s.Planets[i].Mass, s.Planets[i].Name, states)
	}

	// Step 2: Half-step velocity and full-step position
	halfVel := make([]math3d.Vec3, n)
	for i := range s.Planets {
		halfVel[i] = s.Planets[i].Velocity.Add(accel[i].Mul(dt / 2))
		s.Planets[i].Position = s.Planets[i].Position.Add(halfVel[i].Mul(dt))
	}

	// Step 3: Compute new accelerations at new positions
	newStates := make([]BodyState, n)
	for i := range s.Planets {
		newStates[i] = BodyState{
			Position: s.Planets[i].Position,
			Velocity: halfVel[i],
		}
	}

	for i := range s.Planets {
		newAccel := s.CalculateAccelerationWithSnapshot(i,
			s.Planets[i].Position, halfVel[i],
			s.Planets[i].Mass, s.Planets[i].Name, newStates)

		// Step 4: Complete the velocity step
		s.Planets[i].Velocity = halfVel[i].Add(newAccel.Mul(dt / 2))

		if s.ShowTrails && s.Planets[i].ShowTrail {
			s.Planets[i].Trail = append(s.Planets[i].Trail, s.Planets[i].Position)
			if len(s.Planets[i].Trail) > s.maxTrailLen {
				s.Planets[i].Trail = s.Planets[i].Trail[1:]
			}
		}
	}

	s.CurrentTime += dt
}
