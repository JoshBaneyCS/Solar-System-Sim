//go:build !rust_render

package ui

import (
	"solar-system-sim/internal/render"
)

func (a *App) initGPU() *render.GPURenderer {
	return nil
}

// detectGPUInfo is a no-op when rust_render is not enabled.
func (a *App) detectGPUInfo() {}
