package ui

import (
	"fmt"
	"sync"
	"time"

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

	// Playback controls (initially hidden)
	playbackSection := container.NewVBox()
	telemetryLabel := widget.NewLabel("")
	telemetryLabel.Wrapping = fyne.TextWrapWord

	var playPauseBtn *widget.Button
	speedSlider := widget.NewSlider(0, 6) // 2^0=1x to 2^6=64x
	speedSlider.Value = 3                 // 8x default
	speedSlider.Step = 0.5
	speedLabel := widget.NewLabel("Playback: 8.0x")

	timelineSlider := widget.NewSlider(0, 100)
	timelineSlider.Step = 0.5

	speedSlider.OnChanged = func(v float64) {
		speed := 1.0
		for i := 0.0; i < v; i++ {
			speed *= 2
		}
		speedLabel.SetText(fmt.Sprintf("Playback: %.1fx", speed))
		if a.playback != nil {
			a.playback.SetSpeed(speed)
		}
	}

	timelineSlider.OnChanged = func(v float64) {
		if a.playback != nil {
			t := v / 100.0 * a.playback.TotalTime()
			a.playback.SetTime(t)
		}
	}

	playPauseBtn = widget.NewButton("Play", func() {
		if a.playback == nil {
			return
		}
		if a.playback.IsPlaying() {
			a.playback.Pause()
			playPauseBtn.SetText("Play")
		} else {
			a.playback.Play()
			playPauseBtn.SetText("Pause")
		}
	})

	showPlaybackControls := func() {
		playbackSection.Objects = []fyne.CanvasObject{
			widget.NewSeparator(),
			widget.NewLabel("Mission Playback:"),
			container.NewHBox(playPauseBtn),
			speedLabel,
			speedSlider,
			widget.NewLabel("Timeline:"),
			timelineSlider,
			telemetryLabel,
		}
		playbackSection.Refresh()
	}

	hidePlaybackControls := func() {
		a.playback = nil
		a.renderer.LaunchVehiclePos = nil
		playbackSection.Objects = nil
		playbackSection.Refresh()
	}

	simulateBtn := widget.NewButton("Simulate Launch", func() {
		v := launch.GetVehicle(selectedVehicle)
		d := launch.GetDestination(selectedDest)

		resultsLabel.SetText("Computing...")

		go func() {
			plan := a.launch.planner.Plan(v, d)
			traj := a.launch.planner.PropagateTrajectory(plan)
			a.launch.SetResult(plan, traj)

			// Get Earth position for mission playback
			planets := a.simulator.GetPlanetSnapshot()
			var earthPos = planets[2].Position
			if len(planets) > 2 {
				earthPos = planets[2].Position
			}

			a.playback = NewMissionPlayback(traj, earthPos)

			resultsLabel.SetText(launch.Summary(plan))
			showPlaybackControls()
		}()
	})

	clearBtn := widget.NewButton("Clear Trajectory", func() {
		a.launch.Clear()
		hidePlaybackControls()
		resultsLabel.SetText("Trajectory cleared.")
	})

	// Start telemetry update goroutine
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		for range ticker.C {
			if a.playback != nil {
				telem := a.playback.CurrentTelemetry()
				days := telem.ElapsedTime / 86400
				speedKmS := telem.Speed / 1000
				distAU := telem.DistFromEarth / 1.496e11
				text := fmt.Sprintf("Elapsed: %.1f days\nSpeed: %.1f km/s\nDistance: %.4f AU\nProgress: %.1f%%",
					days, speedKmS, distAU, telem.ProgressPct)
				telemetryLabel.SetText(text)

				// Update timeline slider position
				if a.playback.TotalTime() > 0 {
					pct := a.playback.CurrentTimeSeconds() / a.playback.TotalTime() * 100
					timelineSlider.Value = pct
					timelineSlider.Refresh()
				}
			}
		}
	}()

	resultsScroll := container.NewVScroll(resultsLabel)
	resultsScroll.SetMinSize(fyne.NewSize(260, 200))

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
		playbackSection,
	)
}
