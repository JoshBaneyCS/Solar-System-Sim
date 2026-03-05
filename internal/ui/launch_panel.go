package ui

import (
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"solar-system-sim/internal/launch"
)

// launchState holds the current launch planner state (thread-safe).
type launchState struct {
	mu         sync.RWMutex
	planner    *launch.Planner
	plan       *launch.LaunchPlan
	trajectory *launch.Trajectory
	active     bool
}

func newLaunchState() *launchState {
	return &launchState{
		planner: launch.NewPlanner(),
	}
}

func (ls *launchState) SetResult(plan launch.LaunchPlan, traj *launch.Trajectory) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.plan = &plan
	ls.trajectory = traj
	ls.active = true
}

func (ls *launchState) Clear() {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.plan = nil
	ls.trajectory = nil
	ls.active = false
}

func (ls *launchState) GetTrajectory() *launch.Trajectory {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.trajectory
}

func (ls *launchState) IsActive() bool {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.active
}

func (a *App) createLaunchPanel() *fyne.Container {
	destKeys := launch.DestinationNames()
	destNames := launch.DestinationDisplayNames()
	vehicleKeys := launch.VehicleNames()

	selectedDest := destKeys[0]
	selectedVehicle := vehicleKeys[0]

	resultsLabel := widget.NewLabel("Select a destination and vehicle, then click Simulate.")
	resultsLabel.Wrapping = fyne.TextWrapWord

	destSelect := widget.NewSelect(destNames, func(selected string) {
		for i, name := range destNames {
			if name == selected {
				selectedDest = destKeys[i]
				break
			}
		}
	})
	destSelect.SetSelectedIndex(0)

	vehicleDisplayNames := make([]string, len(vehicleKeys))
	for i, k := range vehicleKeys {
		v := launch.GetVehicle(k)
		dv := launch.TotalVehicleDeltaV(v)
		vehicleDisplayNames[i] = fmt.Sprintf("%s (%.1f km/s)", v.Name, dv/1000)
	}

	vehicleSelect := widget.NewSelect(vehicleDisplayNames, func(selected string) {
		for i, name := range vehicleDisplayNames {
			if name == selected {
				selectedVehicle = vehicleKeys[i]
				break
			}
		}
	})
	vehicleSelect.SetSelectedIndex(0)

	simulateBtn := widget.NewButton("Simulate Launch", func() {
		v := launch.GetVehicle(selectedVehicle)
		d := launch.GetDestination(selectedDest)

		resultsLabel.SetText("Computing...")

		go func() {
			plan := a.launch.planner.Plan(v, d)
			traj := a.launch.planner.PropagateTrajectory(plan)
			a.launch.SetResult(plan, traj)

			resultsLabel.SetText(launch.Summary(plan))
		}()
	})

	clearBtn := widget.NewButton("Clear Trajectory", func() {
		a.launch.Clear()
		resultsLabel.SetText("Trajectory cleared.")
	})

	resultsScroll := container.NewVScroll(resultsLabel)
	resultsScroll.SetMinSize(fyne.NewSize(260, 300))

	return container.NewVBox(
		widget.NewLabel("Launch Planner"),
		widget.NewSeparator(),
		widget.NewLabel("Launch Site: Kennedy Space Center"),
		widget.NewSeparator(),
		widget.NewLabel("Destination:"),
		destSelect,
		widget.NewLabel("Vehicle:"),
		vehicleSelect,
		widget.NewSeparator(),
		simulateBtn,
		clearBtn,
		widget.NewSeparator(),
		resultsScroll,
	)
}
