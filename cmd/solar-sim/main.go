package main

import (
	"fmt"
	"os"
)

const usage = `Solar System Simulator

Usage:
  solar-sim <command> [flags]

Commands:
  gui        Launch the graphical user interface
  run        Run a headless simulation and export ephemeris
  validate   Run physics validation scenarios
  launch     Compute launch plan and trajectory
  assets     Asset pipeline commands (verify)

Run 'solar-sim <command> --help' for details on each command.`

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "gui":
		runGUI()
	case "run":
		runSim(os.Args[2:])
	case "validate":
		runValidate(os.Args[2:])
	case "launch":
		runLaunch(os.Args[2:])
	case "assets":
		runAssets(os.Args[2:])
	case "--help", "-h", "help":
		fmt.Println(usage)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		fmt.Println(usage)
		os.Exit(1)
	}
}
