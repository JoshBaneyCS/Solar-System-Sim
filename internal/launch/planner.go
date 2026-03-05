package launch

import (
	"fmt"
	"math"

	"solar-system-sim/internal/math3d"
)

// DeltaVBudget breaks down the total delta-v requirement.
type DeltaVBudget struct {
	Ascent      float64 // ascent to parking orbit (includes gravity/drag losses)
	PlaneChange float64 // inclination change (if needed)
	Transfer    float64 // transfer burn
	Arrival     float64 // circularization or capture burn
	Total       float64 // sum of all components
}

// LaunchPlan is the result of a launch simulation.
type LaunchPlan struct {
	Vehicle        Vehicle
	Destination    Destination
	Budget         DeltaVBudget
	TransferTime   float64 // seconds
	ParkingOrbitV  float64 // parking orbit velocity (m/s)
	ParkingAltitude float64 // parking orbit altitude (m)
	VehicleDeltaV  float64 // total vehicle dv capability (m/s)
	Feasible       bool    // vehicle has enough dv
}

// Planner computes launch plans for given vehicle/destination pairs.
type Planner struct{}

// NewPlanner creates a new launch planner.
func NewPlanner() *Planner {
	return &Planner{}
}

// Plan computes the dv budget and feasibility for a launch.
func (p *Planner) Plan(v Vehicle, d Destination) LaunchPlan {
	plan := LaunchPlan{
		Vehicle:     v,
		Destination: d,
	}

	// Parking orbit: 200km circular (all launches start from LEO parking orbit)
	parkingAlt := d.Altitude
	if parkingAlt <= 0 {
		parkingAlt = 200e3
	}
	rPark := REarth + parkingAlt
	plan.ParkingAltitude = parkingAlt
	plan.ParkingOrbitV = CircularVelocity(MuEarth, rPark)

	// Ascent dv: orbital velocity + gravity/drag losses - Earth rotation assist
	plan.Budget.Ascent = plan.ParkingOrbitV + GravityDragLoss - EarthRotationalVelocity

	// Plane change: only for Earth-centered destinations where target inclination
	// differs from launch latitude. For interplanetary, plane changes are handled
	// as part of the departure burn and are negligible for near-ecliptic targets.
	if d.Frame == EarthCentered {
		incDiff := math.Abs(d.Inclination - KSCLatitudeRad)
		if incDiff > 0.001 { // ~0.06 degrees threshold
			plan.Budget.PlaneChange = PlaneChangeDV(plan.ParkingOrbitV, incDiff)
		}
	}

	switch {
	case d.Frame == Heliocentric:
		// Interplanetary transfer (Mars)
		p.planInterplanetary(&plan, rPark)
	case d.ApoapsisAlt > 100000e3:
		// Lunar transfer (TLI)
		p.planLunar(&plan, rPark)
	case d.ApoapsisAlt > d.Altitude+1e3:
		// Elliptical transfer (GTO)
		p.planElliptical(&plan, rPark)
	default:
		// Direct insertion to circular orbit (LEO, ISS)
		plan.Budget.Transfer = 0
		plan.Budget.Arrival = 0
		plan.TransferTime = 0
	}

	plan.Budget.Total = plan.Budget.Ascent + plan.Budget.PlaneChange +
		plan.Budget.Transfer + plan.Budget.Arrival

	plan.VehicleDeltaV = TotalVehicleDeltaV(v)
	plan.Feasible = plan.VehicleDeltaV >= plan.Budget.Total

	return plan
}

// planElliptical computes dv for a transfer to an elliptical orbit (e.g., GTO).
func (p *Planner) planElliptical(plan *LaunchPlan, rPark float64) {
	rApo := REarth + plan.Destination.ApoapsisAlt
	dv1, dv2 := HohmannDeltaV(MuEarth, rPark, rApo)
	plan.Budget.Transfer = dv1
	plan.Budget.Arrival = dv2
	plan.TransferTime = HohmannTransferTime(MuEarth, rPark, rApo)
}

// planLunar computes dv for a trans-lunar injection.
func (p *Planner) planLunar(plan *LaunchPlan, rPark float64) {
	// TLI as Hohmann transfer to Moon distance
	rMoon := REarth + plan.Destination.ApoapsisAlt
	dv1, dv2 := HohmannDeltaV(MuEarth, rPark, rMoon)
	plan.Budget.Transfer = dv1

	// Lunar orbit insertion: compute capture burn from hyperbolic excess
	// The arrival excess velocity relative to Moon is approximately dv2
	// Capture into 100km lunar orbit
	rLunarOrbit := RMoon + 100e3
	plan.Budget.Arrival = HyperbolicExcessDV(MuMoon, rLunarOrbit, dv2)

	plan.TransferTime = HohmannTransferTime(MuEarth, rPark, rMoon)
}

// planInterplanetary computes dv for an interplanetary Hohmann transfer.
func (p *Planner) planInterplanetary(plan *LaunchPlan, rPark float64) {
	// Hohmann transfer in heliocentric frame
	dv1Helio, dv2Helio := HohmannDeltaV(MuSun, EarthOrbitSMA, plan.Destination.SemiMajorAxis)

	// Convert heliocentric dv1 to hyperbolic excess at Earth departure
	plan.Budget.Transfer = HyperbolicExcessDV(MuEarth, rPark, dv1Helio)

	// Arrival burn at Mars (simplified: hyperbolic excess to capture)
	// Mars mu ≈ 4.283e13, parking orbit ~300km above Mars surface (r ≈ 3.69e6)
	muMars := 4.283e13
	rMarsOrbit := 3.3895e6 + 300e3
	plan.Budget.Arrival = HyperbolicExcessDV(muMars, rMarsOrbit, dv2Helio)

	plan.TransferTime = HohmannTransferTime(MuSun, EarthOrbitSMA, plan.Destination.SemiMajorAxis)
}

// PropagateTrajectory generates a numerical trajectory for the launch plan.
func (p *Planner) PropagateTrajectory(plan LaunchPlan) *Trajectory {
	rPark := REarth + plan.ParkingAltitude

	switch {
	case plan.Destination.Frame == Heliocentric:
		// Interplanetary: propagate in heliocentric frame
		// Start at Earth's orbit, apply departure velocity
		pos := math3d.Vec3{X: EarthOrbitSMA, Y: 0, Z: 0}
		vCirc := CircularVelocity(MuSun, EarthOrbitSMA)
		_, dv2 := HohmannDeltaV(MuSun, EarthOrbitSMA, plan.Destination.SemiMajorAxis)
		_ = dv2
		dv1, _ := HohmannDeltaV(MuSun, EarthOrbitSMA, plan.Destination.SemiMajorAxis)
		vel := math3d.Vec3{X: 0, Y: vCirc + dv1, Z: 0}

		traj := Propagate(pos, vel, PropagatorConfig{
			Mu:       MuSun,
			TimeStep: 3600,
			Duration: plan.TransferTime,
		})
		traj.Frame = Heliocentric
		return traj

	default:
		// Earth-centered: propagate from parking orbit
		pos := math3d.Vec3{X: rPark, Y: 0, Z: 0}

		// Apply transfer burn
		vPark := CircularVelocity(MuEarth, rPark)
		totalV := vPark + plan.Budget.Transfer
		vel := math3d.Vec3{X: 0, Y: totalV, Z: 0}

		duration := plan.TransferTime
		if duration <= 0 {
			// For circular orbits, propagate one full orbit
			duration = 2 * math.Pi * rPark / vPark
		}

		traj := Propagate(pos, vel, PropagatorConfig{
			Mu:       MuEarth,
			TimeStep: 60,
			Duration: duration,
		})
		traj.Frame = EarthCentered
		return traj
	}
}

// Summary returns a human-readable summary of the launch plan.
func Summary(plan LaunchPlan) string {
	status := "FEASIBLE"
	if !plan.Feasible {
		status = "NOT FEASIBLE (insufficient dv)"
	}

	transferDays := plan.TransferTime / (24 * 3600)

	return fmt.Sprintf(`Kennedy Space Center Launch Plan
================================
Vehicle: %s (total dv: %.1f m/s = %.2f km/s)
Destination: %s
Status: %s

Delta-V Budget:
  Ascent to parking orbit:  %8.1f m/s  (%.2f km/s)
  Plane change:             %8.1f m/s  (%.2f km/s)
  Transfer burn:            %8.1f m/s  (%.2f km/s)
  Arrival burn:             %8.1f m/s  (%.2f km/s)
  ─────────────────────────────────
  Total required:           %8.1f m/s  (%.2f km/s)

Parking Orbit:
  Altitude: %.0f km
  Velocity: %.1f m/s (%.2f km/s)

Transfer Time: %.1f days (%.2f hours)

Formulas Used:
  Tsiolkovsky: dv = Isp * g0 * ln(m0/mf)
  Orbital velocity: v = sqrt(mu/r)
  Hohmann transfer: dv1 = sqrt(mu/r1) * (sqrt(2*r2/(r1+r2)) - 1)
  Transfer time: t = pi * sqrt(a^3/mu)`,
		plan.Vehicle.Name, plan.VehicleDeltaV, plan.VehicleDeltaV/1000,
		plan.Destination.Name,
		status,
		plan.Budget.Ascent, plan.Budget.Ascent/1000,
		plan.Budget.PlaneChange, plan.Budget.PlaneChange/1000,
		plan.Budget.Transfer, plan.Budget.Transfer/1000,
		plan.Budget.Arrival, plan.Budget.Arrival/1000,
		plan.Budget.Total, plan.Budget.Total/1000,
		plan.ParkingAltitude/1000,
		plan.ParkingOrbitV, plan.ParkingOrbitV/1000,
		transferDays, plan.TransferTime/3600,
	)
}
