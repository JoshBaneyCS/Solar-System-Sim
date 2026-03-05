package launch

import (
	"encoding/csv"
	"fmt"
	"io"
)

// WriteCSV writes a launch plan summary and trajectory data to CSV format.
func WriteCSV(plan LaunchPlan, traj *Trajectory, w io.Writer) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Header comment rows (prefixed with # for parsers that support it)
	cw.Write([]string{"# Kennedy Space Center Launch Plan"})
	cw.Write([]string{fmt.Sprintf("# Vehicle: %s", plan.Vehicle.Name)})
	cw.Write([]string{fmt.Sprintf("# Destination: %s", plan.Destination.Name)})
	cw.Write([]string{fmt.Sprintf("# Total DeltaV: %.1f m/s", plan.Budget.Total)})
	cw.Write([]string{fmt.Sprintf("# Feasible: %v", plan.Feasible)})
	cw.Write([]string{""})

	// Column headers
	cw.Write([]string{
		"time_s", "pos_x_m", "pos_y_m", "pos_z_m",
		"vel_x_ms", "vel_y_ms", "vel_z_ms", "distance_m",
	})

	if traj == nil {
		return cw.Error()
	}

	for _, pt := range traj.Points {
		dist := pt.Position.Magnitude()
		cw.Write([]string{
			fmt.Sprintf("%.1f", pt.Time),
			fmt.Sprintf("%.3f", pt.Position.X),
			fmt.Sprintf("%.3f", pt.Position.Y),
			fmt.Sprintf("%.3f", pt.Position.Z),
			fmt.Sprintf("%.3f", pt.Velocity.X),
			fmt.Sprintf("%.3f", pt.Velocity.Y),
			fmt.Sprintf("%.3f", pt.Velocity.Z),
			fmt.Sprintf("%.3f", dist),
		})
	}

	return cw.Error()
}
