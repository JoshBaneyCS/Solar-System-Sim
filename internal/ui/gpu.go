//go:build rust_render

package ui

import (
	"solar-system-sim/internal/render"
)

// initGPU attempts to create a GPU renderer. Returns nil on failure (graceful fallback).
func (a *App) initGPU() *render.GPURenderer {
	return render.NewGPURenderer(a.simulator, a.viewport, a.renderer, 800, 600)
}
