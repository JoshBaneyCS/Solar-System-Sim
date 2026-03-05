package physics

// IntegratorType selects the numerical integration method.
type IntegratorType int

const (
	// IntegratorRK4 is the 4th-order Runge-Kutta integrator (default).
	IntegratorRK4 IntegratorType = iota
	// IntegratorVerlet is the Velocity Verlet (symplectic) integrator.
	IntegratorVerlet
)
