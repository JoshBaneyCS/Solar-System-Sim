package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// Settings holds persistent application configuration.
type Settings struct {
	GPUMode       string // "auto", "on", "off"
	RayTracing    string // "auto", "on", "off"
	QualityPreset string // "low", "medium", "high"
	Integrator    string // "verlet", "rk4"
	ShowTrails    bool
	ShowSpacetime bool
	ShowLabels    bool
	PlanetGravity bool
	Relativity    bool
	ShowMoons     bool
	ShowComets    bool
	ShowAsteroids bool
	ShowBelt      bool
}

// DefaultSettings returns the default configuration.
func DefaultSettings() Settings {
	return Settings{
		GPUMode:       "auto",
		RayTracing:    "auto",
		QualityPreset: "medium",
		Integrator:    "verlet",
		ShowTrails:    true,
		ShowSpacetime: false,
		ShowLabels:    true,
		PlanetGravity: true,
		Relativity:    true,
		ShowMoons:     true,
		ShowComets:    false,
		ShowAsteroids: false,
		ShowBelt:      true,
	}
}

// LoadSettings reads settings from Fyne preferences.
func LoadSettings(prefs fyne.Preferences) Settings {
	d := DefaultSettings()
	return Settings{
		GPUMode:       prefs.StringWithFallback("gpu_mode", d.GPUMode),
		RayTracing:    prefs.StringWithFallback("ray_tracing", d.RayTracing),
		QualityPreset: prefs.StringWithFallback("quality_preset", d.QualityPreset),
		Integrator:    prefs.StringWithFallback("integrator", d.Integrator),
		ShowTrails:    prefs.BoolWithFallback("show_trails", d.ShowTrails),
		ShowSpacetime: prefs.BoolWithFallback("show_spacetime", d.ShowSpacetime),
		ShowLabels:    prefs.BoolWithFallback("show_labels", d.ShowLabels),
		PlanetGravity: prefs.BoolWithFallback("planet_gravity", d.PlanetGravity),
		Relativity:    prefs.BoolWithFallback("relativity", d.Relativity),
		ShowMoons:     prefs.BoolWithFallback("show_moons", d.ShowMoons),
		ShowComets:    prefs.BoolWithFallback("show_comets", d.ShowComets),
		ShowAsteroids: prefs.BoolWithFallback("show_asteroids", d.ShowAsteroids),
		ShowBelt:      prefs.BoolWithFallback("show_belt", d.ShowBelt),
	}
}

// Save writes settings to Fyne preferences.
func (s Settings) Save(prefs fyne.Preferences) {
	prefs.SetString("gpu_mode", s.GPUMode)
	prefs.SetString("ray_tracing", s.RayTracing)
	prefs.SetString("quality_preset", s.QualityPreset)
	prefs.SetString("integrator", s.Integrator)
	prefs.SetBool("show_trails", s.ShowTrails)
	prefs.SetBool("show_spacetime", s.ShowSpacetime)
	prefs.SetBool("show_labels", s.ShowLabels)
	prefs.SetBool("planet_gravity", s.PlanetGravity)
	prefs.SetBool("relativity", s.Relativity)
	prefs.SetBool("show_moons", s.ShowMoons)
	prefs.SetBool("show_comets", s.ShowComets)
	prefs.SetBool("show_asteroids", s.ShowAsteroids)
	prefs.SetBool("show_belt", s.ShowBelt)
}

// showSettingsDialog opens a modal settings dialog.
// It reads from the centralized AppState and writes back through it.
func (a *App) showSettingsDialog() {
	// Read current state into a local Settings copy for the dialog
	s := a.state.ToSettings()
	s.GPUMode = a.settings.GPUMode
	s.RayTracing = a.settings.RayTracing
	s.QualityPreset = a.settings.QualityPreset

	gpuSelect := widget.NewSelect([]string{"Auto", "On", "Off"}, func(v string) {
		s.GPUMode = v
	})
	gpuSelect.Selected = s.GPUMode

	rtSelect := widget.NewSelect([]string{"Auto", "On", "Off"}, func(v string) {
		s.RayTracing = v
	})
	rtSelect.Selected = s.RayTracing

	qualitySelect := widget.NewSelect([]string{"Low", "Medium", "High"}, func(v string) {
		s.QualityPreset = v
	})
	qualitySelect.Selected = s.QualityPreset

	integratorSelect := widget.NewSelect([]string{"Verlet (symplectic)", "RK4 (classic)"}, func(v string) {
		if v == "RK4 (classic)" {
			s.Integrator = "rk4"
		} else {
			s.Integrator = "verlet"
		}
	})
	if s.Integrator == "rk4" {
		integratorSelect.Selected = "RK4 (classic)"
	} else {
		integratorSelect.Selected = "Verlet (symplectic)"
	}

	trailsCheck := widget.NewCheck("Show Trails", func(v bool) { s.ShowTrails = v })
	trailsCheck.Checked = s.ShowTrails

	spacetimeCheck := widget.NewCheck("Show Spacetime Fabric", func(v bool) { s.ShowSpacetime = v })
	spacetimeCheck.Checked = s.ShowSpacetime

	labelsCheck := widget.NewCheck("Show Labels", func(v bool) { s.ShowLabels = v })
	labelsCheck.Checked = s.ShowLabels

	gravityCheck := widget.NewCheck("Planet-Planet Gravity", func(v bool) { s.PlanetGravity = v })
	gravityCheck.Checked = s.PlanetGravity

	relativityCheck := widget.NewCheck("General Relativity", func(v bool) { s.Relativity = v })
	relativityCheck.Checked = s.Relativity

	form := widget.NewForm(
		widget.NewFormItem("GPU Mode", gpuSelect),
		widget.NewFormItem("Ray Tracing", rtSelect),
		widget.NewFormItem("Quality", qualitySelect),
		widget.NewFormItem("Integrator", integratorSelect),
		widget.NewFormItem("", trailsCheck),
		widget.NewFormItem("", spacetimeCheck),
		widget.NewFormItem("", labelsCheck),
		widget.NewFormItem("", gravityCheck),
		widget.NewFormItem("", relativityCheck),
	)

	dialog.ShowCustomConfirm("Settings", "Apply", "Cancel", form, func(apply bool) {
		if apply {
			a.settings = s
			a.state.ApplyFromSettings(s)
			s.Save(a.fyneApp.Preferences())
		}
	}, a.window)
}
