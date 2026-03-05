//go:build !rust_render

package render

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"solar-system-sim/internal/physics"
	"solar-system-sim/internal/viewport"
)

// GPURenderer stub when rust_render is not enabled.
type GPURenderer struct{}

// NewGPURenderer always returns nil without rust_render build tag.
func NewGPURenderer(_ *physics.Simulator, _ *viewport.ViewPort, _ *Renderer, _, _ uint32) *GPURenderer {
	return nil
}

func (g *GPURenderer) Raster() *canvas.Raster              { return nil }
func (g *GPURenderer) Resize(_, _ uint32)                  {}
func (g *GPURenderer) Refresh()                            {}
func (g *GPURenderer) Free()                               {}
func (g *GPURenderer) SetRTMode(_ bool)                    {}
func (g *GPURenderer) CreateLabelOverlay() *fyne.Container { return container.NewWithoutLayout() }
