package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"solar-system-sim/internal/validation"
)

func runValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	scenario := fs.String("scenario", "all", "Scenario: "+strings.Join(validation.AllScenarios(), ", ")+", all")
	years := fs.Float64("years", 10, "Simulation years")
	fs.Parse(args)

	var results []*validation.Result

	if *scenario == "all" {
		results = validation.RunAll(*years)
	} else {
		r, err := validation.RunScenario(*scenario, *years)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		results = []*validation.Result{r}
	}

	anyFailed := false
	for _, r := range results {
		fmt.Println(r.String())
		fmt.Println()
		if !r.Pass {
			anyFailed = true
		}
	}

	if anyFailed {
		os.Exit(1)
	}
}
