//go:build rust_render || metal_render || cuda_render || rocm_render

package render

import (
	"image"
	"image/color"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

	"solar-system-sim/internal/ffi"
	"solar-system-sim/internal/launch"
	"solar-system-sim/internal/math3d"
	"solar-system-sim/internal/physics"
	"solar-system-sim/internal/viewport"
)

// GPURenderer uses the Rust wgpu backend for rendering.
type GPURenderer struct {
	rust      *ffi.RustRenderer
	raster    *canvas.Raster
	simulator *physics.Simulator
	viewport  *viewport.ViewPort
	renderer  *Renderer // for SelectedBodies access
	width     uint32
	height    uint32
}

// resolveAssetDir finds the assets/textures directory.
func resolveAssetDir() string {
	// Try relative to working directory first (common for development)
	candidates := []string{
		filepath.Join("assets", "textures"),
	}
	// Try relative to executable
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "assets", "textures"),
			filepath.Join(dir, "..", "assets", "textures"),
		)
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

// NewGPURenderer creates a GPU renderer. Returns nil if GPU init fails.
func NewGPURenderer(sim *physics.Simulator, vp *viewport.ViewPort, r *Renderer, width, height uint32) *GPURenderer {
	assetDir := resolveAssetDir()
	rust := ffi.NewRustRendererWithTextures(width, height, assetDir)
	if rust == nil {
		return nil
	}

	gpu := &GPURenderer{
		rust:      rust,
		simulator: sim,
		viewport:  vp,
		renderer:  r,
		width:     width,
		height:    height,
	}

	gpu.raster = canvas.NewRaster(gpu.generateImage)
	return gpu
}

// Raster returns the Fyne canvas raster object for display.
func (g *GPURenderer) Raster() *canvas.Raster {
	return g.raster
}

// Resize updates the GPU render target size.
func (g *GPURenderer) Resize(width, height uint32) {
	if width == 0 || height == 0 {
		return
	}
	g.width = width
	g.height = height
	g.rust.Resize(width, height)
}

// Refresh triggers a raster redraw.
func (g *GPURenderer) Refresh() {
	g.raster.Refresh()
}

// SetRTMode enables or disables ray tracing.
func (g *GPURenderer) SetRTMode(enabled bool) {
	if g.rust != nil {
		g.rust.SetRTMode(enabled)
	}
}

// Free releases GPU resources.
func (g *GPURenderer) Free() {
	if g.rust != nil {
		g.rust.Free()
		g.rust = nil
	}
}

// colorToFloat64 converts a Go color to [r,g,b,a] in 0-1 range.
func colorToFloat64(c color.Color) [4]float64 {
	r, gr, b, a := c.RGBA()
	return [4]float64{
		float64(r) / 65535.0,
		float64(gr) / 65535.0,
		float64(b) / 65535.0,
		float64(a) / 65535.0,
	}
}

func (g *GPURenderer) generateImage(w, h int) image.Image {
	if g.rust == nil {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}

	planets := g.simulator.GetPlanetSnapshot()
	sun := g.simulator.GetSunSnapshot()

	g.simulator.RLock()
	showTrails := g.simulator.ShowTrails
	showSpacetime := g.simulator.ShowSpacetime
	g.simulator.RUnlock()

	// Update viewport size if changed
	if uint32(w) != g.width || uint32(h) != g.height {
		g.Resize(uint32(w), uint32(h))
	}

	// Set camera
	g.viewport.RLock()
	zoom := g.viewport.Zoom
	panX := g.viewport.PanX
	panY := g.viewport.PanY
	rotX := g.viewport.RotationX
	rotY := g.viewport.RotationY
	rotZ := g.viewport.RotationZ
	use3D := g.viewport.Use3D
	var followX, followY, followZ float64
	if g.viewport.FollowBody != nil {
		followX = g.viewport.FollowBody.Position.X
		followY = g.viewport.FollowBody.Position.Y
		followZ = g.viewport.FollowBody.Position.Z
	}
	g.viewport.RUnlock()

	g.rust.SetCamera(zoom, panX, panY, rotX, rotY, rotZ, use3D, followX, followY, followZ)

	// Set bodies
	n := len(planets)
	positions := make([]float64, n*3)
	colors := make([]float64, n*4)
	radii := make([]float64, n)

	for i, p := range planets {
		positions[i*3] = p.Position.X
		positions[i*3+1] = p.Position.Y
		positions[i*3+2] = p.Position.Z

		c := colorToFloat64(p.Color)
		colors[i*4] = c[0]
		colors[i*4+1] = c[1]
		colors[i*4+2] = c[2]
		colors[i*4+3] = c[3]

		radii[i] = p.Radius
	}

	sunPos := []float64{sun.Position.X, sun.Position.Y, sun.Position.Z}
	sc := colorToFloat64(sun.Color)
	sunColor := []float64{sc[0], sc[1], sc[2], sc[3]}

	if n > 0 {
		g.rust.SetBodies(uint32(n), positions, colors, radii, sunPos, sunColor, sun.Radius)
	}

	// Set trails (including launch trajectory as extra trail)
	launchTraj := g.renderer.LaunchTrajectory
	var earthPos math3d.Vec3
	if launchTraj != nil && len(planets) > 2 {
		earthPos = planets[2].Position
	}

	if showTrails || launchTraj != nil {
		trailCount := n
		if launchTraj != nil && len(launchTraj.Points) > 1 {
			trailCount++
		}

		trailLengths := make([]uint32, trailCount)
		var trailPositions []float64
		trailColors := make([]float64, trailCount*4)

		if showTrails {
			for i, p := range planets {
				trailLengths[i] = uint32(len(p.Trail))

				for _, tp := range p.Trail {
					trailPositions = append(trailPositions, tp.X, tp.Y, tp.Z)
				}

				c := colorToFloat64(p.Color)
				trailColors[i*4] = c[0]
				trailColors[i*4+1] = c[1]
				trailColors[i*4+2] = c[2]
				trailColors[i*4+3] = c[3]
			}
		} else {
			for i := 0; i < n; i++ {
				trailLengths[i] = 0
			}
		}

		// Add launch trajectory as the last trail
		if launchTraj != nil && len(launchTraj.Points) > 1 {
			idx := n
			trailLengths[idx] = uint32(len(launchTraj.Points))
			for _, pt := range launchTraj.Points {
				pos := pt.Position
				if launchTraj.Frame == launch.EarthCentered {
					pos = pos.Add(earthPos)
				}
				trailPositions = append(trailPositions, pos.X, pos.Y, pos.Z)
			}
			// Green color for trajectory
			trailColors[idx*4] = 0.0
			trailColors[idx*4+1] = 1.0
			trailColors[idx*4+2] = 0.5
			trailColors[idx*4+3] = 0.8
		}

		if len(trailPositions) > 0 {
			g.rust.SetTrails(uint32(trailCount), trailLengths, trailPositions, trailColors, true)
		} else {
			g.rust.SetTrails(uint32(trailCount), nil, nil, nil, false)
		}
	} else {
		g.rust.SetTrails(0, nil, nil, nil, false)
	}

	// Set spacetime
	if showSpacetime {
		// Include sun + all planets
		totalBodies := 1 + n
		masses := make([]float64, totalBodies)
		stPositions := make([]float64, totalBodies*3)

		masses[0] = sun.Mass
		stPositions[0] = sun.Position.X
		stPositions[1] = sun.Position.Y
		stPositions[2] = sun.Position.Z

		for i, p := range planets {
			masses[1+i] = p.Mass
			stPositions[(1+i)*3] = p.Position.X
			stPositions[(1+i)*3+1] = p.Position.Y
			stPositions[(1+i)*3+2] = p.Position.Z
		}

		g.rust.SetSpacetime(true, masses, stPositions, uint32(totalBodies))
	} else {
		g.rust.SetSpacetime(false, nil, nil, 0)
	}

	// Set distance line
	if len(g.renderer.SelectedBodies) == 2 {
		p1 := g.renderer.SelectedBodies[0].Position
		p2 := g.renderer.SelectedBodies[1].Position
		g.rust.SetDistanceLine(true, p1.X, p1.Y, p1.Z, p2.X, p2.Y, p2.Z)
	} else {
		g.rust.SetDistanceLine(false, 0, 0, 0, 0, 0, 0)
	}

	// Render frame
	pixels := g.rust.RenderFrame()
	if pixels == nil {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}

	// Create image from pixel data
	img := &image.RGBA{
		Pix:    pixels,
		Stride: int(g.width) * 4,
		Rect:   image.Rect(0, 0, int(g.width), int(g.height)),
	}

	return img
}

// CreateLabelOverlay returns text labels as a Fyne container for overlay on GPU raster.
func (g *GPURenderer) CreateLabelOverlay() *fyne.Container {
	return g.renderer.CreateLabelOverlay()
}
