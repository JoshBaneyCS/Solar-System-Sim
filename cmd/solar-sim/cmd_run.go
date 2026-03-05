package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"

	"solar-system-sim/internal/physics"
	"solar-system-sim/pkg/constants"
)

type snapshot struct {
	TimeSec float64        `json:"time_s"`
	Bodies  []jsonBodyData `json:"bodies"`
}

type jsonBodyData struct {
	Name       string     `json:"name"`
	Position   [3]float64 `json:"pos"`
	Velocity   [3]float64 `json:"vel"`
	DistanceAU float64    `json:"distance_au"`
}

type ephemerisJSON struct {
	Config    jsonConfig `json:"config"`
	Snapshots []snapshot `json:"snapshots"`
}

type jsonConfig struct {
	Years      float64 `json:"years"`
	Dt         float64 `json:"dt"`
	Integrator string  `json:"integrator"`
}

func runSim(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	years := fs.Float64("years", 1.0, "Simulation duration in years")
	dt := fs.Float64("dt", 7200, "Integration timestep in seconds")
	export := fs.String("export", "", "Output file path (required)")
	format := fs.String("format", "csv", "Output format: csv or json")
	integrator := fs.String("integrator", "verlet", "Integrator: verlet or rk4")
	sampleInterval := fs.Int("sample-interval", 1, "Record every Nth step")
	fs.Parse(args)

	if *export == "" {
		fmt.Fprintln(os.Stderr, "Error: --export is required")
		fs.Usage()
		os.Exit(1)
	}

	if *sampleInterval < 1 {
		*sampleInterval = 1
	}

	*format = strings.ToLower(*format)
	if *format != "csv" && *format != "json" {
		fmt.Fprintf(os.Stderr, "Error: --format must be 'csv' or 'json', got '%s'\n", *format)
		os.Exit(1)
	}

	sim := physics.NewSimulator()
	sim.ShowTrails = false

	switch strings.ToLower(*integrator) {
	case "verlet":
		sim.Integrator = physics.IntegratorVerlet
	case "rk4":
		sim.Integrator = physics.IntegratorRK4
	default:
		fmt.Fprintf(os.Stderr, "Error: --integrator must be 'verlet' or 'rk4', got '%s'\n", *integrator)
		os.Exit(1)
	}

	totalSeconds := *years * 365.25 * 24 * 3600
	numSteps := int(math.Ceil(totalSeconds / *dt))

	fmt.Fprintf(os.Stderr, "Running simulation: %.2f years, dt=%.0fs, %d steps, integrator=%s\n",
		*years, *dt, numSteps, *integrator)

	f, err := os.Create(*export)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if *format == "csv" {
		writeCSVEphemeris(sim, f, numSteps, *dt, *sampleInterval)
	} else {
		writeJSONEphemeris(sim, f, numSteps, *dt, *sampleInterval, *years, *integrator)
	}

	fmt.Fprintf(os.Stderr, "Done. Output written to %s\n", *export)
}

func writeCSVEphemeris(sim *physics.Simulator, f *os.File, numSteps int, dt float64, sampleInterval int) {
	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"time_s", "body", "pos_x_m", "pos_y_m", "pos_z_m", "vel_x_ms", "vel_y_ms", "vel_z_ms", "distance_from_sun_m"})

	progressInterval := numSteps / 10
	if progressInterval < 1 {
		progressInterval = 1
	}

	// Write initial state
	writeCSVSnapshot(w, sim)

	for step := 1; step <= numSteps; step++ {
		sim.Step(dt)

		if step%sampleInterval == 0 {
			writeCSVSnapshot(w, sim)
		}

		if step%progressInterval == 0 {
			pct := step * 100 / numSteps
			fmt.Fprintf(os.Stderr, "Progress: %d%% (step %d/%d)\n", pct, step, numSteps)
		}
	}
}

func writeCSVSnapshot(w *csv.Writer, sim *physics.Simulator) {
	for _, p := range sim.Planets {
		dist := p.Position.Sub(sim.Sun.Position).Magnitude()
		w.Write([]string{
			fmt.Sprintf("%.1f", sim.CurrentTime),
			p.Name,
			fmt.Sprintf("%.6e", p.Position.X),
			fmt.Sprintf("%.6e", p.Position.Y),
			fmt.Sprintf("%.6e", p.Position.Z),
			fmt.Sprintf("%.6e", p.Velocity.X),
			fmt.Sprintf("%.6e", p.Velocity.Y),
			fmt.Sprintf("%.6e", p.Velocity.Z),
			fmt.Sprintf("%.6e", dist),
		})
	}
}

func writeJSONEphemeris(sim *physics.Simulator, f *os.File, numSteps int, dt float64, sampleInterval int, years float64, integrator string) {
	result := ephemerisJSON{
		Config: jsonConfig{
			Years:      years,
			Dt:         dt,
			Integrator: integrator,
		},
	}

	progressInterval := numSteps / 10
	if progressInterval < 1 {
		progressInterval = 1
	}

	// Capture initial state
	result.Snapshots = append(result.Snapshots, captureSnapshot(sim))

	for step := 1; step <= numSteps; step++ {
		sim.Step(dt)

		if step%sampleInterval == 0 {
			result.Snapshots = append(result.Snapshots, captureSnapshot(sim))
		}

		if step%progressInterval == 0 {
			pct := step * 100 / numSteps
			fmt.Fprintf(os.Stderr, "Progress: %d%% (step %d/%d)\n", pct, step, numSteps)
		}
	}

	if len(result.Snapshots) > 1_000_000 {
		fmt.Fprintf(os.Stderr, "Warning: %d snapshots in JSON output. Consider using --sample-interval to reduce size.\n", len(result.Snapshots))
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func captureSnapshot(sim *physics.Simulator) snapshot {
	s := snapshot{TimeSec: sim.CurrentTime}
	for _, p := range sim.Planets {
		dist := p.Position.Sub(sim.Sun.Position).Magnitude()
		s.Bodies = append(s.Bodies, jsonBodyData{
			Name:       p.Name,
			Position:   [3]float64{p.Position.X, p.Position.Y, p.Position.Z},
			Velocity:   [3]float64{p.Velocity.X, p.Velocity.Y, p.Velocity.Z},
			DistanceAU: dist / constants.AU,
		})
	}
	return s
}
