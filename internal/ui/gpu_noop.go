//go:build !rust_render

package ui

import (
	"solar-system-sim/internal/render"
)

func (a *App) initGPU() *render.GPURenderer {
	return nil
}
