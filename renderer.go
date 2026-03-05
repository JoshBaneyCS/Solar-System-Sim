package main

import (
	"fmt"
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// RenderCache pools canvas objects for performance
type RenderCache struct {
	circles     []*canvas.Circle
	lines       []*canvas.Line
	texts       []*canvas.Text
	circleIndex int
	lineIndex   int
	textIndex   int
	mu          sync.Mutex
}

func NewRenderCache() *RenderCache {
	return &RenderCache{
		circles: make([]*canvas.Circle, 0, 100),
		lines:   make([]*canvas.Line, 0, 5000),
		texts:   make([]*canvas.Text, 0, 50),
	}
}

func (rc *RenderCache) Reset() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.circleIndex = 0
	rc.lineIndex = 0
	rc.textIndex = 0
}

func (rc *RenderCache) GetCircle(col color.Color) *canvas.Circle {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.circleIndex < len(rc.circles) {
		circle := rc.circles[rc.circleIndex]
		circle.FillColor = col
		rc.circleIndex++
		return circle
	}

	circle := canvas.NewCircle(col)
	rc.circles = append(rc.circles, circle)
	rc.circleIndex++
	return circle
}

func (rc *RenderCache) GetLine(col color.Color) *canvas.Line {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.lineIndex < len(rc.lines) {
		line := rc.lines[rc.lineIndex]
		line.StrokeColor = col
		rc.lineIndex++
		return line
	}

	line := canvas.NewLine(col)
	rc.lines = append(rc.lines, line)
	rc.lineIndex++
	return line
}

func (rc *RenderCache) GetText(text string, col color.Color) *canvas.Text {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.textIndex < len(rc.texts) {
		textObj := rc.texts[rc.textIndex]
		textObj.Text = text
		textObj.Color = col
		rc.textIndex++
		return textObj
	}

	textObj := canvas.NewText(text, col)
	rc.texts = append(rc.texts, textObj)
	rc.textIndex++
	return textObj
}

// Renderer handles all rendering operations
type Renderer struct {
	simulator         *Simulator
	viewport          *ViewPort
	cache             *RenderCache
	spacetimeRenderer *SpacetimeRenderer

	// Distance measurement
	selectedBodies []*Body
}

func NewRenderer(sim *Simulator, vp *ViewPort) *Renderer {
	return &Renderer{
		simulator:         sim,
		viewport:          vp,
		cache:             NewRenderCache(),
		spacetimeRenderer: NewSpacetimeRenderer(),
		selectedBodies:    make([]*Body, 0, 2),
	}
}

// CreateCanvas renders the simulation state to a canvas
func (r *Renderer) CreateCanvas() *fyne.Container {
	// Reset render cache for object reuse
	r.cache.Reset()

	// Get thread-safe snapshots of simulation data
	planets := r.simulator.GetPlanetSnapshot()
	sun := r.simulator.GetSunSnapshot()

	// Get simulation state
	r.simulator.mu.RLock()
	showTrails := r.simulator.ShowTrails
	showSpacetime := r.simulator.ShowSpacetime
	r.simulator.mu.RUnlock()

	// Get viewport dimensions
	r.viewport.mu.RLock()
	canvasWidth := r.viewport.CanvasWidth
	canvasHeight := r.viewport.CanvasHeight
	r.viewport.mu.RUnlock()

	// Create objects for rendering
	objects := []fyne.CanvasObject{}

	// Background - sized to match viewport
	bg := canvas.NewRectangle(color.RGBA{5, 5, 15, 255})
	bg.Resize(fyne.NewSize(float32(canvasWidth), float32(canvasHeight)))
	bg.Move(fyne.NewPos(0, 0))
	objects = append(objects, bg)

	// Render spacetime fabric if enabled
	if showSpacetime {
		spacetimeObjects := r.spacetimeRenderer.RenderGrid(r.cache, r.viewport, planets, sun)
		objects = append(objects, spacetimeObjects...)
	}

	// Render trails with object pooling and adaptive resolution
	if showTrails {
		for _, planet := range planets {
			if len(planet.Trail) > 1 {
				// Adaptive resolution - skip trail points if too many
				step := 1
				if len(planet.Trail) > 200 {
					step = len(planet.Trail) / 200
				}

				for j := 0; j < len(planet.Trail)-step; j += step {
					x1, y1 := r.viewport.WorldToScreen(planet.Trail[j])
					x2, y2 := r.viewport.WorldToScreen(planet.Trail[j+step])

					// Only render if on screen
					if r.isOnScreen(x1, y1, canvasWidth, canvasHeight) ||
						r.isOnScreen(x2, y2, canvasWidth, canvasHeight) {
						// Fade older trail segments
						alpha := uint8(float64(j) / float64(len(planet.Trail)) * 255)
						lineColor := planet.Color
						if c, ok := planet.Color.(color.RGBA); ok {
							lineColor = color.RGBA{c.R, c.G, c.B, alpha}
						}

						line := r.cache.GetLine(lineColor)
						line.Position1 = fyne.NewPos(x1, y1)
						line.Position2 = fyne.NewPos(x2, y2)
						line.StrokeWidth = 1
						objects = append(objects, line)
					}
				}
			}
		}
	}

	// Render Sun
	sunX, sunY := r.viewport.WorldToScreen(sun.Position)
	if r.isOnScreen(sunX, sunY, canvasWidth, canvasHeight) {
		sunRadius := float32(sun.Radius)
		sunCircle := r.cache.GetCircle(sun.Color)
		sunCircle.Resize(fyne.NewSize(sunRadius*2, sunRadius*2))
		sunCircle.Move(fyne.NewPos(sunX-sunRadius, sunY-sunRadius))
		objects = append(objects, sunCircle)

		sunLabel := r.cache.GetText("Sun", color.White)
		sunLabel.TextSize = 10
		sunLabel.Move(fyne.NewPos(sunX+sunRadius+5, sunY-5))
		objects = append(objects, sunLabel)
	}

	// Render planets
	for _, planet := range planets {
		x, y := r.viewport.WorldToScreen(planet.Position)

		// Only render if on screen (with some margin for labels)
		if r.isOnScreen(x, y, canvasWidth, canvasHeight) {
			planetRadius := float32(planet.Radius)

			circle := r.cache.GetCircle(planet.Color)
			circle.Resize(fyne.NewSize(planetRadius*2, planetRadius*2))
			circle.Move(fyne.NewPos(x-planetRadius, y-planetRadius))
			objects = append(objects, circle)

			label := r.cache.GetText(planet.Name, color.White)
			label.TextSize = 10
			label.Move(fyne.NewPos(x+planetRadius+3, y-5))
			objects = append(objects, label)
		}
	}

	// Render distance measurement line if 2 bodies are selected
	if len(r.selectedBodies) == 2 {
		pos1 := r.selectedBodies[0].Position
		pos2 := r.selectedBodies[1].Position

		x1, y1 := r.viewport.WorldToScreen(pos1)
		x2, y2 := r.viewport.WorldToScreen(pos2)

		// Draw line between bodies
		distLine := r.cache.GetLine(color.RGBA{255, 255, 0, 180})
		distLine.Position1 = fyne.NewPos(x1, y1)
		distLine.Position2 = fyne.NewPos(x2, y2)
		distLine.StrokeWidth = 2
		objects = append(objects, distLine)

		// Calculate distance
		dist := pos2.Sub(pos1).Magnitude()
		distAU := dist / AU
		distKm := dist / 1000
		lightMinutes := dist / (299792458.0 * 60)

		// Show distance label at midpoint
		midX := (x1 + x2) / 2
		midY := (y1 + y2) / 2

		distText := fmt.Sprintf("%.3f AU\n%.2e km\n%.2f light-min", distAU, distKm, lightMinutes)
		distLabel := r.cache.GetText(distText, color.RGBA{255, 255, 0, 255})
		distLabel.TextSize = 12
		distLabel.Alignment = fyne.TextAlignCenter
		distLabel.Move(fyne.NewPos(midX-50, midY-30))
		objects = append(objects, distLabel)
	}

	return container.NewWithoutLayout(objects...)
}

// isOnScreen checks if a point is visible on the canvas (with margin)
func (r *Renderer) isOnScreen(x, y float32, width, height float64) bool {
	margin := float32(100) // Allow some off-screen rendering for labels
	return x >= -margin && x <= float32(width)+margin &&
		y >= -margin && y <= float32(height)+margin
}
