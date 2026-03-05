package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"solar-system-sim/internal/launch"
)

func runLaunch(args []string) {
	fs := flag.NewFlagSet("launch", flag.ExitOnError)
	dest := fs.String("dest", "leo", "Destination: leo, iss, gto, moon, mars")
	vehicle := fs.String("vehicle", "generic", "Vehicle preset: generic, falcon, saturnv")
	export := fs.String("export", "", "Output CSV file path (optional)")
	listDests := fs.Bool("list-destinations", false, "List available destinations")
	listVehicles := fs.Bool("list-vehicles", false, "List available vehicles")
	fs.Parse(args)

	if *listDests {
		fmt.Println("Available destinations:")
		for _, k := range launch.DestinationNames() {
			d := launch.GetDestination(k)
			fmt.Printf("  %-8s  %s\n", k, d.Name)
		}
		return
	}

	if *listVehicles {
		fmt.Println("Available vehicles:")
		for _, k := range launch.VehicleNames() {
			v := launch.GetVehicle(k)
			dv := launch.TotalVehicleDeltaV(v)
			fmt.Printf("  %-10s  %s (%d stages, %.1f km/s dv)\n", k, v.Name, len(v.Stages), dv/1000)
		}
		return
	}

	v := launch.GetVehicle(strings.ToLower(*vehicle))
	d := launch.GetDestination(strings.ToLower(*dest))

	planner := launch.NewPlanner()
	plan := planner.Plan(v, d)

	fmt.Println(launch.Summary(plan))

	if *export != "" {
		traj := planner.PropagateTrajectory(plan)

		f, err := os.Create(*export)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()

		if err := launch.WriteCSV(plan, traj, f); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing CSV: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nTrajectory written to %s (%d points)\n", *export, len(traj.Points))
	}
}
