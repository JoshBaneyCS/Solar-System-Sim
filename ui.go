package main

import (
	"fmt"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

// App is the main application
type App struct {
	fyneApp   fyne.App
	window    fyne.Window
	simulator *Simulator
	viewport  *ViewPort
	renderer  *Renderer
	canvas    *fyne.Container
}

func NewApp() *App {
	fyneApp := app.New()
	window := fyneApp.NewWindow("Solar System Simulator")

	sim := NewSimulator()
	vp := NewViewPort()
	renderer := NewRenderer(sim, vp)

	return &App{
		fyneApp:   fyneApp,
		window:    window,
		simulator: sim,
		viewport:  vp,
		renderer:  renderer,
	}
}

func (a *App) createPhysicsPanel() *fyne.Container {
	equations := widget.NewLabel("")
	equations.Wrapping = fyne.TextWrapWord

	updateEquations := func() {
		planets := a.simulator.GetPlanetSnapshot()
		sun := a.simulator.GetSunSnapshot()

		a.simulator.mu.RLock()
		sunMass := a.simulator.SunMass
		planetGravityEnabled := a.simulator.PlanetGravityEnabled
		relativisticEffects := a.simulator.RelativisticEffects
		currentTime := a.simulator.CurrentTime
		timeSpeed := a.simulator.TimeSpeed
		a.simulator.mu.RUnlock()

		if len(planets) < 3 {
			return
		}

		earth := planets[2]
		r := earth.Position.Magnitude()
		v := earth.Velocity.Magnitude()

		totalForce := Vec3{0, 0, 0}

		rSun := sun.Position.Sub(earth.Position)
		distSun := rSun.Magnitude()
		if distSun > 1e6 {
			forceSun := rSun.Normalize().Mul(G * sunMass * earth.Mass / (distSun * distSun))
			totalForce = totalForce.Add(forceSun)
		}

		var planetForceText string
		if planetGravityEnabled {
			planetForceCount := 0
			for i := range planets {
				if i == 2 {
					continue
				}
				other := planets[i]
				rPlanet := other.Position.Sub(earth.Position)
				distPlanet := rPlanet.Magnitude()
				if distPlanet > 1e6 {
					forcePlanet := rPlanet.Normalize().Mul(G * other.Mass * earth.Mass / (distPlanet * distPlanet))
					totalForce = totalForce.Add(forcePlanet)
					planetForceCount++
				}
			}
			planetForceText = fmt.Sprintf("\nPlanet-Planet Interactions: ENABLED (%d forces)", planetForceCount)
		} else {
			planetForceText = "\nPlanet-Planet Interactions: DISABLED"
		}

		relText := ""
		if relativisticEffects {
			relText = "\nGeneral Relativity: ENABLED (Mercury precession: ~43\"/century)"
		} else {
			relText = "\nGeneral Relativity: DISABLED"
		}

		eqText := fmt.Sprintf(`Physics Equations:

Newton's Law of Universal Gravitation (3D):
F⃗ = -GMm/r² · r̂
where G = %.3e m³kg⁻¹s⁻²
      M = %.3e kg (Sun mass)
      r = %.3e m (distance)

N-Body Problem (Planet-Planet Gravity):
F⃗ᵢ = Σⱼ≠ᵢ (-GMⱼmᵢ/rᵢⱼ²) · r̂ᵢⱼ%s

General Relativity Correction (Mercury):
a⃗_GR = (3G²M²)/(c²r³L) · (L⃗ × r⃗)
Adds perihelion precession%s

Newton's Second Law:
F⃗ = ma⃗ = m(d²r⃗/dt²)

Acceleration:
a⃗ = -GM/r² · r̂ (+ planet + GR corrections)
|a⃗| = %.3e m/s²

Current Earth Values:
Position (3D): (%.3e, %.3e, %.3e) m
Distance from Sun: %.3e m (%.3f AU)
Orbital Velocity: %.3e m/s (%.1f km/s)
Orbital Period: %.1f days
Total Force Magnitude: %.3e N

Simulation Time: %.1f days (%.2f years)
Time Speed: %.1fx
Zoom: %.2fx`,
			G,
			sunMass,
			r,
			planetForceText,
			relText,
			G*sunMass/(r*r),
			earth.Position.X, earth.Position.Y, earth.Position.Z,
			r, r/AU,
			v, v/1000,
			2*math.Pi*r/v/(24*3600),
			totalForce.Magnitude(),
			currentTime/(24*3600),
			currentTime/(365.25*24*3600),
			timeSpeed,
			a.viewport.Zoom)

		equations.SetText(eqText)
	}

	updateEquations()

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		for range ticker.C {
			updateEquations()
			equations.Refresh()
		}
	}()

	scroll := container.NewVScroll(equations)
	scroll.SetMinSize(fyne.NewSize(350, 600))

	return container.NewMax(scroll)
}

func (a *App) createControls() *fyne.Container {
	playButton := widget.NewButton("Play", func() {
		a.simulator.IsPlaying = !a.simulator.IsPlaying
	})

	speedLabel := widget.NewLabel(fmt.Sprintf("Speed: %.1fx", a.simulator.TimeSpeed))
	speedSlider := widget.NewSlider(-10, 10)
	speedSlider.Value = 0
	speedSlider.Step = 0.1
	speedSlider.OnChanged = func(value float64) {
		a.simulator.TimeSpeed = math.Pow(2, value)
		speedLabel.SetText(fmt.Sprintf("Speed: %.1fx", a.simulator.TimeSpeed))
	}

	rewindButton := widget.NewButton("⏪ Rewind", func() {
		if a.simulator.TimeSpeed > 0 {
			a.simulator.TimeSpeed = -a.simulator.TimeSpeed
		}
		a.simulator.IsPlaying = true
	})

	forwardButton := widget.NewButton("Fast Forward ⏩", func() {
		if a.simulator.TimeSpeed < 0 {
			a.simulator.TimeSpeed = -a.simulator.TimeSpeed
		}
		a.simulator.IsPlaying = true
	})

	trailsCheck := widget.NewCheck("Show Orbital Trails", func(checked bool) {
		a.simulator.ShowTrails = checked
		if !checked {
			a.simulator.ClearTrails()
		}
	})
	trailsCheck.Checked = true

	spacetimeCheck := widget.NewCheck("Show Spacetime Fabric", func(checked bool) {
		a.simulator.ShowSpacetime = checked
	})
	spacetimeCheck.Checked = false

	planetGravityCheck := widget.NewCheck("Planet-Planet Gravity (N-Body)", func(checked bool) {
		a.simulator.PlanetGravityEnabled = checked
	})
	planetGravityCheck.Checked = true

	relativityCheck := widget.NewCheck("General Relativity (Mercury)", func(checked bool) {
		a.simulator.RelativisticEffects = checked
	})
	relativityCheck.Checked = true

	sunMassLabel := widget.NewLabel("Sun Mass: 1.00x")
	sunMassSlider := widget.NewSlider(0.1, 5.0)
	sunMassSlider.Value = 1.0
	sunMassSlider.Step = 0.1
	sunMassSlider.OnChanged = func(value float64) {
		a.simulator.SetSunMass(value)
		sunMassLabel.SetText(fmt.Sprintf("Sun Mass: %.2fx", value))
	}

	zoomLabel := widget.NewLabel(fmt.Sprintf("Zoom: %.2fx", a.viewport.Zoom))
	zoomSlider := widget.NewSlider(-2, 3)
	zoomSlider.Value = 0
	zoomSlider.Step = 0.1
	zoomSlider.OnChanged = func(value float64) {
		zoom := math.Pow(2, value)
		a.viewport.SetZoom(zoom)
		zoomLabel.SetText(fmt.Sprintf("Zoom: %.2fx", zoom))
	}

	autoFitButton := widget.NewButton("🔍 Auto-Fit All Planets", func() {
		planets := a.simulator.GetPlanetSnapshot()
		sun := a.simulator.GetSunSnapshot()
		a.viewport.AutoFit(planets, sun)
		zoomSlider.Value = math.Log2(a.viewport.Zoom)
		zoomLabel.SetText(fmt.Sprintf("Zoom: %.2fx", a.viewport.Zoom))
	})

	followOptions := []string{"None (Free Camera)", "Sun", "Mercury", "Venus", "Earth", "Mars", "Jupiter", "Saturn", "Uranus", "Neptune"}
	followSelect := widget.NewSelect(followOptions, func(selected string) {
		if selected == "None (Free Camera)" {
			a.viewport.mu.Lock()
			a.viewport.FollowBody = nil
			a.viewport.mu.Unlock()
		} else if selected == "Sun" {
			a.viewport.mu.Lock()
			a.viewport.FollowBody = &a.simulator.Sun
			a.viewport.mu.Unlock()
		} else {
			a.simulator.mu.RLock()
			for i := range a.simulator.Planets {
				if a.simulator.Planets[i].Name == selected {
					a.viewport.mu.Lock()
					a.viewport.FollowBody = &a.simulator.Planets[i]
					a.viewport.mu.Unlock()
					break
				}
			}
			a.simulator.mu.RUnlock()
		}
	})
	followSelect.Selected = "None (Free Camera)"

	enable3DCheck := widget.NewCheck("Enable 3D View", func(checked bool) {
		a.viewport.mu.Lock()
		a.viewport.Use3D = checked
		a.viewport.mu.Unlock()
	})

	rotXLabel := widget.NewLabel("Pitch: 0°")
	rotXSlider := widget.NewSlider(-math.Pi, math.Pi)
	rotXSlider.Value = 0
	rotXSlider.Step = 0.1
	rotXSlider.OnChanged = func(value float64) {
		a.viewport.mu.Lock()
		a.viewport.RotationX = value
		a.viewport.mu.Unlock()
		rotXLabel.SetText(fmt.Sprintf("Pitch: %.0f°", value*180/math.Pi))
	}

	rotYLabel := widget.NewLabel("Yaw: 0°")
	rotYSlider := widget.NewSlider(-math.Pi, math.Pi)
	rotYSlider.Value = 0
	rotYSlider.Step = 0.1
	rotYSlider.OnChanged = func(value float64) {
		a.viewport.mu.Lock()
		a.viewport.RotationY = value
		a.viewport.mu.Unlock()
		rotYLabel.SetText(fmt.Sprintf("Yaw: %.0f°", value*180/math.Pi))
	}

	rotZLabel := widget.NewLabel("Roll: 0°")
	rotZSlider := widget.NewSlider(-math.Pi, math.Pi)
	rotZSlider.Value = 0
	rotZSlider.Step = 0.1
	rotZSlider.OnChanged = func(value float64) {
		a.viewport.mu.Lock()
		a.viewport.RotationZ = value
		a.viewport.mu.Unlock()
		rotZLabel.SetText(fmt.Sprintf("Roll: %.0f°", value*180/math.Pi))
	}

	reset3DButton := widget.NewButton("Reset 3D View", func() {
		a.viewport.mu.Lock()
		a.viewport.RotationX = 0
		a.viewport.RotationY = 0
		a.viewport.RotationZ = 0
		a.viewport.mu.Unlock()
		rotXSlider.Value = 0
		rotYSlider.Value = 0
		rotZSlider.Value = 0
	})

	resetButton := widget.NewButton("Reset Simulation", func() {
		a.simulator = NewSimulator()
		a.viewport = NewViewPort()
		a.renderer.simulator = a.simulator
		a.renderer.viewport = a.viewport
		speedSlider.Value = 0
		zoomSlider.Value = 0
		sunMassSlider.Value = 1.0
		rotXSlider.Value = 0
		rotYSlider.Value = 0
		rotZSlider.Value = 0
		trailsCheck.Checked = true
		spacetimeCheck.Checked = false
		planetGravityCheck.Checked = true
		relativityCheck.Checked = true
		enable3DCheck.Checked = false
		followSelect.Selected = "None (Free Camera)"
	})

	controls := container.NewVBox(
		widget.NewLabel("Controls:"),
		playButton,
		widget.NewSeparator(),
		widget.NewLabel("Time Control:"),
		speedLabel,
		speedSlider,
		rewindButton,
		forwardButton,
		widget.NewSeparator(),
		widget.NewLabel("Camera Controls:"),
		zoomLabel,
		zoomSlider,
		autoFitButton,
		widget.NewLabel("Follow:"),
		followSelect,
		widget.NewSeparator(),
		widget.NewLabel("3D View:"),
		enable3DCheck,
		rotXLabel,
		rotXSlider,
		rotYLabel,
		rotYSlider,
		rotZLabel,
		rotZSlider,
		reset3DButton,
		widget.NewSeparator(),
		widget.NewLabel("Display Options:"),
		trailsCheck,
		spacetimeCheck,
		widget.NewSeparator(),
		widget.NewLabel("Physics Options:"),
		planetGravityCheck,
		relativityCheck,
		widget.NewSeparator(),
		widget.NewLabel("Sun Properties:"),
		sunMassLabel,
		sunMassSlider,
		widget.NewSeparator(),
		resetButton,
	)

	return controls
}

func (a *App) Run() {
	physicsPanel := a.createPhysicsPanel()
	controls := a.createControls()

	// Create canvas with initial size
	a.canvas = a.renderer.CreateCanvas()

	// Create a responsive canvas container that scales with the window
	// Background rectangle that will resize dynamically
	canvasRect := canvas.NewRectangle(color.Transparent)

	canvasContainer := container.NewMax(canvasRect, a.canvas)

	// Layout: Controls on left, canvas in center, physics on right
	leftScroll := container.NewScroll(controls)
	leftScroll.SetMinSize(fyne.NewSize(280, 600)) // Ensure left panel is always visible
	leftPanel := leftScroll

	mainContent := container.NewHSplit(
		canvasContainer,
		physicsPanel,
	)
	mainContent.SetOffset(0.7) // 70% for canvas, 30% for physics

	content := container.NewBorder(nil, nil, leftPanel, nil, mainContent)

	a.window.SetContent(content)

	// Make window scale to screen size
	go func() {
		time.Sleep(50 * time.Millisecond) // Wait for window to initialize

		// Start with a good default size
		a.window.Resize(fyne.NewSize(1600, 900))

		// Center the window on screen
		a.window.CenterOnScreen()
	}()

	// Add menu bar with view options
	viewMenu := fyne.NewMenu("View",
		fyne.NewMenuItem("Maximize Window", func() {
			// Get desktop size and maximize to fit
			// This will trigger the resize handler automatically
			a.window.Resize(fyne.NewSize(2000, 1200))
			a.window.CenterOnScreen()
		}),
		fyne.NewMenuItem("Fullscreen", func() {
			a.window.SetFullScreen(!a.window.FullScreen())
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Reset Window Size", func() {
			a.window.SetFullScreen(false)
			a.window.Resize(fyne.NewSize(1600, 900))
			a.window.CenterOnScreen()
		}),
	)

	mainMenu := fyne.NewMainMenu(viewMenu)
	a.window.SetMainMenu(mainMenu)

	// Track the canvas container for dynamic size updates
	var lastWidth, lastHeight float32

	// Continuously update viewport size when window is resized
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		for range ticker.C {
			size := canvasContainer.Size()

			// Only update if size has changed to avoid unnecessary updates
			if size.Width != lastWidth || size.Height != lastHeight {
				lastWidth = size.Width
				lastHeight = size.Height

				// Update viewport to match canvas container size
				if size.Width > 0 && size.Height > 0 {
					a.viewport.UpdateCanvasSize(float64(size.Width), float64(size.Height))

					// Update background rectangle to match
					canvasRect.Resize(size)
				}
			}
		}
	}()

	// Animation loop
	go func() {
		ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS
		for range ticker.C {
			a.simulator.Update(baseTimeStep)

			// Update canvas objects
			a.canvas.Objects = a.renderer.CreateCanvas().Objects
			a.canvas.Refresh()
		}
	}()

	a.window.ShowAndRun()
}
