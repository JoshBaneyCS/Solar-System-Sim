package ui

import (
	"image/png"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"solar-system-sim/internal/physics"
)

func (a *App) buildMainMenu() *fyne.MainMenu {
	// --- File menu ---
	exportScreenshot := fyne.NewMenuItem("Export Screenshot...", func() {
		dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil || writer == nil {
				return
			}
			defer writer.Close()
			img := a.window.Canvas().Capture()
			png.Encode(writer, img)
		}, a.window)
	})

	quit := fyne.NewMenuItem("Quit", func() {
		a.fyneApp.Quit()
	})

	fileMenu := fyne.NewMenu("File", exportScreenshot, fyne.NewMenuItemSeparator(), quit)

	// --- View menu ---
	trailsItem := fyne.NewMenuItem("Toggle Trails", nil)
	trailsItem.Checked = a.state.ShowTrails()

	spacetimeItem := fyne.NewMenuItem("Toggle Spacetime Fabric", nil)
	spacetimeItem.Checked = a.state.ShowSpacetime()

	labelsItem := fyne.NewMenuItem("Toggle Labels", nil)
	labelsItem.Checked = a.state.ShowLabels()

	trailsItem.Action = func() {
		a.state.SetShowTrails(!a.state.ShowTrails())
	}

	spacetimeItem.Action = func() {
		a.state.SetShowSpacetime(!a.state.ShowSpacetime())
	}

	labelsItem.Action = func() {
		a.state.SetShowLabels(!a.state.ShowLabels())
	}

	// Register listener to keep menu check marks in sync
	a.state.AddListener(func() {
		trailsItem.Checked = a.state.ShowTrails()
		spacetimeItem.Checked = a.state.ShowSpacetime()
		labelsItem.Checked = a.state.ShowLabels()
	})

	maximizeItem := fyne.NewMenuItem("Maximize Window", func() {
		a.window.Resize(fyne.NewSize(2000, 1200))
		a.window.CenterOnScreen()
	})
	fullscreenItem := fyne.NewMenuItem("Fullscreen", func() {
		a.window.SetFullScreen(!a.window.FullScreen())
	})
	resetSizeItem := fyne.NewMenuItem("Reset Window Size", func() {
		a.window.SetFullScreen(false)
		a.window.Resize(fyne.NewSize(1600, 900))
		a.window.CenterOnScreen()
	})

	viewMenu := fyne.NewMenu("View",
		trailsItem, spacetimeItem, labelsItem,
		fyne.NewMenuItemSeparator(),
		maximizeItem, fullscreenItem, resetSizeItem,
	)

	// --- Simulation menu ---
	playPause := fyne.NewMenuItem("Play/Pause", func() {
		a.state.SetIsPlaying(!a.state.IsPlaying())
	})

	resetItem := fyne.NewMenuItem("Reset", func() {
		a.simulator = physics.NewSimulator()
		a.renderer.Simulator = a.simulator
		a.state.RebindSimulator(a.simulator)
		a.state.ResetToDefaults()
	})

	verletItem := fyne.NewMenuItem("Integrator: Verlet", nil)
	rk4Item := fyne.NewMenuItem("Integrator: RK4", nil)

	updateIntegratorChecks := func() {
		isVerlet := a.state.Integrator() == physics.IntegratorVerlet
		verletItem.Checked = isVerlet
		rk4Item.Checked = !isVerlet
	}
	updateIntegratorChecks()

	verletItem.Action = func() {
		a.state.SetIntegrator(physics.IntegratorVerlet)
		updateIntegratorChecks()
	}
	rk4Item.Action = func() {
		a.state.SetIntegrator(physics.IntegratorRK4)
		updateIntegratorChecks()
	}

	// Sync integrator menu checks on state changes
	a.state.AddListener(func() {
		updateIntegratorChecks()
	})

	simMenu := fyne.NewMenu("Simulation",
		playPause, resetItem,
		fyne.NewMenuItemSeparator(),
		verletItem, rk4Item,
	)

	// --- Settings menu ---
	settingsItem := fyne.NewMenuItem("Settings...", func() {
		a.showSettingsDialog()
	})
	settingsMenu := fyne.NewMenu("Settings", settingsItem)

	// --- About menu ---
	aboutItem := fyne.NewMenuItem("About...", func() {
		a.showAboutWindow()
	})
	aboutMenu := fyne.NewMenu("About", aboutItem)

	return fyne.NewMainMenu(fileMenu, viewMenu, simMenu, settingsMenu, aboutMenu)
}
