package launch

import (
	"testing"
)

func TestRocketDeltaV(t *testing.T) {
	// Single stage, Isp=300s, mass ratio 10:1 -> dv = 300 * 9.807 * ln(10) ~ 6774 m/s
	dv := RocketDeltaV(300, 100000, 10000)
	assertApprox(t, "single stage dv", dv, 6774.0, 0.01)
}

func TestTotalVehicleDeltaV(t *testing.T) {
	v := GetVehicle("generic")
	dv := TotalVehicleDeltaV(v)
	// Generic 2-stage: should have enough dv for LEO (~9+ km/s)
	if dv < 9000 || dv > 15000 {
		t.Errorf("generic vehicle dv = %.1f, expected 9000-15000", dv)
	}
}

func TestPlannerLEO(t *testing.T) {
	p := NewPlanner()
	plan := p.Plan(GetVehicle("generic"), GetDestination("leo"))

	// LEO total dv should be ~8.9-9.4 km/s
	if plan.Budget.Total < 8000 || plan.Budget.Total > 10000 {
		t.Errorf("LEO total dv = %.1f m/s, expected 8000-10000", plan.Budget.Total)
	}

	// No plane change for LEO (same inclination as KSC)
	if plan.Budget.PlaneChange > 1.0 {
		t.Errorf("LEO plane change = %.1f m/s, expected ~0", plan.Budget.PlaneChange)
	}

	// No transfer/arrival for circular LEO
	if plan.Budget.Transfer > 1.0 {
		t.Errorf("LEO transfer = %.1f m/s, expected 0", plan.Budget.Transfer)
	}
}

func TestPlannerISS(t *testing.T) {
	p := NewPlanner()
	plan := p.Plan(GetVehicle("generic"), GetDestination("iss"))

	// ISS requires plane change (~23 deg) adding ~3 km/s
	if plan.Budget.PlaneChange < 2000 || plan.Budget.PlaneChange > 4000 {
		t.Errorf("ISS plane change = %.1f m/s, expected 2000-4000", plan.Budget.PlaneChange)
	}

	// Total should be higher than LEO due to plane change
	if plan.Budget.Total < 10000 {
		t.Errorf("ISS total dv = %.1f m/s, expected > 10000", plan.Budget.Total)
	}
}

func TestPlannerGTO(t *testing.T) {
	p := NewPlanner()
	plan := p.Plan(GetVehicle("falcon"), GetDestination("gto"))

	// GTO total: ascent (~8.9) + transfer (~2.5) + circ (~1.5) ~ 12-13 km/s
	if plan.Budget.Total < 11000 || plan.Budget.Total > 14000 {
		t.Errorf("GTO total dv = %.1f m/s, expected 11000-14000", plan.Budget.Total)
	}

	// Transfer burn should be ~2.4 km/s
	if plan.Budget.Transfer < 2000 || plan.Budget.Transfer > 3000 {
		t.Errorf("GTO transfer = %.1f m/s, expected 2000-3000", plan.Budget.Transfer)
	}
}

func TestPlannerMoon(t *testing.T) {
	p := NewPlanner()
	plan := p.Plan(GetVehicle("saturnv"), GetDestination("moon"))

	// TLI burn from LEO ~ 3.1 km/s, arrival ~ 0.8 km/s
	if plan.Budget.Transfer < 2500 || plan.Budget.Transfer > 4000 {
		t.Errorf("Moon transfer = %.1f m/s, expected 2500-4000", plan.Budget.Transfer)
	}

	// Transfer time ~ 4-5 days
	days := plan.TransferTime / (24 * 3600)
	if days < 3 || days > 7 {
		t.Errorf("Moon transfer time = %.1f days, expected 3-7", days)
	}
}

func TestPlannerMars(t *testing.T) {
	p := NewPlanner()
	plan := p.Plan(GetVehicle("saturnv"), GetDestination("mars"))

	// Mars transfer time ~ 259 days
	days := plan.TransferTime / (24 * 3600)
	assertApprox(t, "Mars transfer time", days, 259.0, 0.05)

	// Total dv: ascent (~8.9) + departure (~3.6) + arrival (~2.1) ~ 14-16 km/s
	if plan.Budget.Total < 13000 || plan.Budget.Total > 18000 {
		t.Errorf("Mars total dv = %.1f m/s, expected 13000-18000", plan.Budget.Total)
	}

	// Saturn V should be feasible for Mars
	if !plan.Feasible {
		t.Errorf("Saturn V Mars: expected feasible, got not feasible (vehicle dv=%.1f, required=%.1f)",
			plan.VehicleDeltaV, plan.Budget.Total)
	}
}

func TestPlannerFeasibility(t *testing.T) {
	p := NewPlanner()

	// Generic vehicle should be feasible for LEO but not for Mars
	leoplan := p.Plan(GetVehicle("generic"), GetDestination("leo"))
	if !leoplan.Feasible {
		t.Error("Generic vehicle should be feasible for LEO")
	}

	marsplan := p.Plan(GetVehicle("generic"), GetDestination("mars"))
	if marsplan.Feasible {
		t.Error("Generic vehicle should NOT be feasible for Mars")
	}
}

func TestPropagateTrajectory(t *testing.T) {
	p := NewPlanner()
	plan := p.Plan(GetVehicle("generic"), GetDestination("leo"))
	traj := p.PropagateTrajectory(plan)

	if len(traj.Points) < 10 {
		t.Errorf("expected at least 10 trajectory points, got %d", len(traj.Points))
	}

	// First point should be near parking orbit altitude
	r0 := traj.Points[0].Position.Magnitude()
	expectedR := REarth + plan.ParkingAltitude
	rel := (r0 - expectedR) / expectedR
	if rel > 0.01 || rel < -0.01 {
		t.Errorf("initial radius = %.0f m, expected ~%.0f m", r0, expectedR)
	}
}

func TestSummary(t *testing.T) {
	p := NewPlanner()
	plan := p.Plan(GetVehicle("generic"), GetDestination("leo"))
	s := Summary(plan)
	if len(s) < 100 {
		t.Error("summary too short")
	}
}
