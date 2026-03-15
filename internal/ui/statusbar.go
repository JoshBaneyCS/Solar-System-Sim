package ui

import (
	"fmt"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

// StatusBar displays simulation status information at the bottom of the window.
type StatusBar struct {
	fpsLabel     *widget.Label
	simTimeLabel *widget.Label
	speedLabel   *widget.Label
	zoomLabel    *widget.Label
	infoLabel    *widget.Label
	container    *fyne.Container
	frameCount   atomic.Int64
}

// NewStatusBar creates a new status bar.
func NewStatusBar() *StatusBar {
	sb := &StatusBar{
		fpsLabel:     widget.NewLabel("FPS: --"),
		simTimeLabel: widget.NewLabel("Time: --"),
		speedLabel:   widget.NewLabel("Speed: --"),
		zoomLabel:    widget.NewLabel("Zoom: --"),
		infoLabel:    widget.NewLabel(""),
	}

	sb.fpsLabel.TextStyle = fyne.TextStyle{Monospace: true}
	sb.simTimeLabel.TextStyle = fyne.TextStyle{Monospace: true}
	sb.speedLabel.TextStyle = fyne.TextStyle{Monospace: true}
	sb.zoomLabel.TextStyle = fyne.TextStyle{Monospace: true}
	sb.infoLabel.TextStyle = fyne.TextStyle{Monospace: true}

	bg := canvas.NewRectangle(color.RGBA{15, 15, 30, 255})
	bg.SetMinSize(fyne.NewSize(0, 28))

	content := container.NewHBox(
		sb.fpsLabel,
		widget.NewSeparator(),
		sb.simTimeLabel,
		widget.NewSeparator(),
		sb.speedLabel,
		widget.NewSeparator(),
		sb.zoomLabel,
		widget.NewSeparator(),
		sb.infoLabel,
	)

	sb.container = container.NewMax(bg, container.NewCenter(content))
	return sb
}

// Container returns the status bar's fyne container.
func (sb *StatusBar) Container() *fyne.Container {
	return sb.container
}

// IncrementFrame should be called each render frame for FPS counting.
func (sb *StatusBar) IncrementFrame() {
	sb.frameCount.Add(1)
}

// StartFPSCounter launches a goroutine that updates the FPS label every second.
func (sb *StatusBar) StartFPSCounter() {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for range ticker.C {
			fps := sb.frameCount.Swap(0)
			sb.fpsLabel.SetText(fmt.Sprintf("FPS: %d", fps))
		}
	}()
}

// Update refreshes the status bar with current simulation state.
func (sb *StatusBar) Update(simTimeDays, timeSpeed, zoom float64) {
	years := simTimeDays / 365.25
	if years >= 1 {
		sb.simTimeLabel.SetText(fmt.Sprintf("Time: %.2f yr", years))
	} else {
		sb.simTimeLabel.SetText(fmt.Sprintf("Time: %.1f d", simTimeDays))
	}
	sb.speedLabel.SetText(fmt.Sprintf("Speed: %.1fx", timeSpeed))
	sb.zoomLabel.SetText(fmt.Sprintf("Zoom: %.2fx", zoom))
}

// SetInfo sets the info label text (e.g., runtime info).
func (sb *StatusBar) SetInfo(text string) {
	sb.infoLabel.SetText(text)
}
