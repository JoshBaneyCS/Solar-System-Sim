package ui

import (
	"fmt"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"solar-system-sim/internal/physics"
	"solar-system-sim/pkg/constants"
)

func (a *App) createBodiesPanel() fyne.CanvasObject {
	cards := container.NewVBox()

	type bodyLabels struct {
		dist, vel, period *widget.Label
		idx               int
	}
	var labels []bodyLabels

	// Sun card
	sunName := widget.NewLabel("Sun")
	sunName.TextStyle = fyne.TextStyle{Bold: true}
	sunMassLabel := widget.NewLabel(fmt.Sprintf("Mass: %.3e kg", a.simulator.Sun.Mass))
	sunFollowBtn := widget.NewButton("Follow", func() {
		a.viewport.Lock()
		a.viewport.FollowBody = &a.simulator.Sun
		a.viewport.Unlock()
	})
	cards.Add(container.NewVBox(sunName, sunMassLabel, sunFollowBtn, widget.NewSeparator()))

	// Group bodies by type
	groups := []struct {
		title    string
		bodyType physics.BodyType
	}{
		{"Planets", physics.BodyTypePlanet},
		{"Dwarf Planets", physics.BodyTypeDwarfPlanet},
		{"Moons", physics.BodyTypeMoon},
		{"Comets", physics.BodyTypeComet},
		{"Asteroids", physics.BodyTypeAsteroid},
	}

	for _, group := range groups {
		groupCards := container.NewVBox()
		hasEntries := false

		for i := range a.simulator.Planets {
			idx := i
			p := &a.simulator.Planets[idx]

			if p.Type != group.bodyType {
				continue
			}
			hasEntries = true

			name := widget.NewLabel(p.Name)
			name.TextStyle = fyne.TextStyle{Bold: true}

			massLabel := widget.NewLabel(fmt.Sprintf("Mass: %.3e kg", p.Mass))
			distLabel := widget.NewLabel("Distance: --")
			velLabel := widget.NewLabel("Velocity: --")
			periodLabel := widget.NewLabel("Period: --")

			trailCheck := widget.NewCheck("Show Trail", func(checked bool) {
				a.simulator.Lock()
				a.simulator.Planets[idx].ShowTrail = checked
				if !checked {
					a.simulator.Planets[idx].Trail = a.simulator.Planets[idx].Trail[:0]
				}
				a.simulator.Unlock()
			})
			trailCheck.Checked = p.ShowTrail

			followBtn := widget.NewButton("Follow", func() {
				a.simulator.RLock()
				body := &a.simulator.Planets[idx]
				a.simulator.RUnlock()
				a.viewport.Lock()
				a.viewport.FollowBody = body
				a.viewport.Unlock()
			})

			card := container.NewVBox(
				name, massLabel, distLabel, velLabel, periodLabel,
				trailCheck, followBtn, widget.NewSeparator(),
			)
			groupCards.Add(card)
			labels = append(labels, bodyLabels{dist: distLabel, vel: velLabel, period: periodLabel, idx: idx})
		}

		if hasEntries {
			header := widget.NewLabel(group.title)
			header.TextStyle = fyne.TextStyle{Bold: true}
			cards.Add(widget.NewSeparator())
			cards.Add(header)
			cards.Add(groupCards)
		}
	}

	// Live update goroutine
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		for range ticker.C {
			planets := a.simulator.GetPlanetSnapshot()
			sun := a.simulator.GetSunSnapshot()
			for _, bl := range labels {
				if bl.idx >= len(planets) {
					continue
				}
				p := planets[bl.idx]
				dist := p.Position.Sub(sun.Position).Magnitude()
				vel := p.Velocity.Magnitude()
				period := 2 * math.Pi * dist / vel / 86400

				bl.dist.SetText(fmt.Sprintf("Distance: %.4f AU", dist/constants.AU))
				bl.vel.SetText(fmt.Sprintf("Velocity: %.2f km/s", vel/1000))
				bl.period.SetText(fmt.Sprintf("Period: %.1f days", period))
			}
		}
	}()

	scroll := container.NewVScroll(cards)
	scroll.SetMinSize(fyne.NewSize(260, 600))
	return scroll
}
