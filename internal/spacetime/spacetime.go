package spacetime

import (
	"image/color"
	"math"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

	"solar-system-sim/internal/physics"
	"solar-system-sim/internal/viewport"
	"solar-system-sim/pkg/constants"
)

// LineProvider provides pooled line canvas objects for rendering.
// This interface breaks the circular dependency between spacetime and render packages.
type LineProvider interface {
	GetLine(col color.Color) *canvas.Line
}

// SpacetimeRenderer handles visualization of gravitational field as a warped fabric
type SpacetimeRenderer struct {
	gridResolution int
	cache          SpacetimeCache
	mu             sync.RWMutex
}

// SpacetimeCache stores computed data to avoid recalculation
type SpacetimeCache struct {
	lastZoom    float64
	lastPanX    float64
	lastPanY    float64
	potentials  [][]float64
	needsUpdate bool
}

const (
	defaultGridResolution      = 80
	displacementFactor         = 50.0
	maxInfluenceDistance       = 10.0 * constants.AU
	cacheInvalidationThreshold = 0.05
	minGridResolution    = 40
	maxGridResolution    = 120
	baseCurvatureFactor  = 1e11
)

func NewSpacetimeRenderer() *SpacetimeRenderer {
	return &SpacetimeRenderer{
		gridResolution: defaultGridResolution,
		cache: SpacetimeCache{
			needsUpdate: true,
		},
	}
}

// GetAdaptiveResolution adjusts grid detail based on zoom level
func (s *SpacetimeRenderer) GetAdaptiveResolution(zoom float64) int {
	if zoom < 0.5 {
		return minGridResolution
	} else if zoom < 2.0 {
		return 80
	} else if zoom < 10.0 {
		return 100
	} else if zoom > 20.0 {
		return 150
	}
	return maxGridResolution
}

// ShouldUpdateCache checks if viewport changed enough to warrant recalculation
func (s *SpacetimeRenderer) ShouldUpdateCache(zoom, panX, panY float64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.cache.needsUpdate {
		return true
	}

	zoomDiff := math.Abs(zoom-s.cache.lastZoom) / zoom
	panXDiff := math.Abs(panX - s.cache.lastPanX)
	panYDiff := math.Abs(panY - s.cache.lastPanY)

	return zoomDiff > cacheInvalidationThreshold ||
		panXDiff > cacheInvalidationThreshold ||
		panYDiff > cacheInvalidationThreshold
}

// CalculatePotentialField computes gravitational potential at grid points
func (s *SpacetimeRenderer) CalculatePotentialField(
	vp *viewport.ViewPort,
	planets []physics.Body,
	sun physics.Body,
) [][]float64 {
	vp.RLock()
	zoom := vp.Zoom
	panX := vp.PanX
	panY := vp.PanY
	canvasWidth := vp.CanvasWidth
	canvasHeight := vp.CanvasHeight
	vp.RUnlock()

	if !s.ShouldUpdateCache(zoom, panX, panY) {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return s.cache.potentials
	}

	s.gridResolution = s.GetAdaptiveResolution(zoom)

	potentials := make([][]float64, s.gridResolution)
	for i := range potentials {
		potentials[i] = make([]float64, s.gridResolution)
	}

	displayScale := viewport.DefaultDisplayScale * zoom

	worldWidth := canvasWidth / displayScale * constants.AU
	worldHeight := canvasHeight / displayScale * constants.AU

	viewportCenterX := panX * constants.AU
	viewportCenterY := panY * constants.AU

	gridSpacingX := worldWidth / float64(s.gridResolution-1)
	gridSpacingY := worldHeight / float64(s.gridResolution-1)

	effectiveInfluence := maxInfluenceDistance
	if zoom > 2.0 {
		effectiveInfluence = maxInfluenceDistance * zoom
	}

	for i := 0; i < s.gridResolution; i++ {
		for j := 0; j < s.gridResolution; j++ {
			worldX := viewportCenterX - worldWidth/2 + float64(i)*gridSpacingX
			worldY := viewportCenterY - worldHeight/2 + float64(j)*gridSpacingY

			curvature := 0.0

			rSunX := sun.Position.X - worldX
			rSunY := sun.Position.Y - worldY
			rSun := math.Sqrt(rSunX*rSunX + rSunY*rSunY)
			if rSun > 1e6 {
				h00_sun := 2 * constants.G * sun.Mass / (constants.C * constants.C * rSun)
				curvature += h00_sun
			}

			for _, planet := range planets {
				rPlanetX := planet.Position.X - worldX
				rPlanetY := planet.Position.Y - worldY
				rPlanet := math.Sqrt(rPlanetX*rPlanetX + rPlanetY*rPlanetY)

				if rPlanet > 1e6 && rPlanet < effectiveInfluence {
					h00_planet := 2 * constants.G * planet.Mass / (constants.C * constants.C * rPlanet)
					curvature += h00_planet
				}
			}

			// Log-scale normalization reveals planetary warping alongside the Sun's
			scaleFactor := baseCurvatureFactor * (1.0 + zoom*0.5)
			potentials[i][j] = math.Log1p(curvature * scaleFactor)
		}
	}

	s.mu.Lock()
	s.cache.lastZoom = zoom
	s.cache.lastPanX = panX
	s.cache.lastPanY = panY
	s.cache.potentials = potentials
	s.cache.needsUpdate = false
	s.mu.Unlock()

	return potentials
}

// NormalizePotentials converts potential field to 0-1 range for visualization
func (s *SpacetimeRenderer) NormalizePotentials(potentials [][]float64) ([][]float64, float64, float64) {
	if len(potentials) == 0 || len(potentials[0]) == 0 {
		return potentials, 0, 0
	}

	minPotential := potentials[0][0]
	maxPotential := potentials[0][0]

	for i := range potentials {
		for j := range potentials[i] {
			if potentials[i][j] < minPotential {
				minPotential = potentials[i][j]
			}
			if potentials[i][j] > maxPotential {
				maxPotential = potentials[i][j]
			}
		}
	}

	if maxPotential == minPotential {
		return potentials, minPotential, maxPotential
	}

	normalized := make([][]float64, len(potentials))
	pRange := maxPotential - minPotential
	for i := range potentials {
		normalized[i] = make([]float64, len(potentials[i]))
		for j := range potentials[i] {
			v := (potentials[i][j] - minPotential) / pRange
			// Gamma correction to reveal smaller planetary contributions
			normalized[i][j] = math.Pow(v, 0.4)
		}
	}

	return normalized, minPotential, maxPotential
}

// InterpolateColor maps normalized potential (0-1) to color gradient
func (s *SpacetimeRenderer) InterpolateColor(normalizedPotential float64) color.Color {
	t := 1.0 - normalizedPotential

	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}

	if t < 0.33 {
		ratio := t / 0.33
		r := uint8(100 * ratio)
		g := uint8(0)
		b := uint8(50 + 100*ratio)
		return color.RGBA{r, g, b, 180}
	} else if t < 0.66 {
		ratio := (t - 0.33) / 0.33
		r := uint8(100 + 100*ratio)
		g := uint8(0)
		b := uint8(150 - 150*ratio)
		return color.RGBA{r, g, b, 180}
	} else {
		ratio := (t - 0.66) / 0.34
		r := uint8(200 + 55*ratio)
		g := uint8(100 * ratio)
		b := uint8(0)
		return color.RGBA{r, g, b, 180}
	}
}

// RenderGrid generates the warped spacetime grid visualization.
// It accepts a LineProvider to obtain pooled line objects, avoiding circular dependency with render.
func (s *SpacetimeRenderer) RenderGrid(
	lp LineProvider,
	vp *viewport.ViewPort,
	planets []physics.Body,
	sun physics.Body,
) []fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, 0)

	potentials := s.CalculatePotentialField(vp, planets, sun)
	if len(potentials) == 0 {
		return objects
	}

	normalized, _, _ := s.NormalizePotentials(potentials)

	vp.RLock()
	canvasWidth := vp.CanvasWidth
	canvasHeight := vp.CanvasHeight
	zoom := vp.Zoom
	vp.RUnlock()

	effectiveDisplacement := displacementFactor * (1.0 + zoom/2.0)

	rows := len(normalized)
	cols := len(normalized[0])

	gridSpacingX := float32(canvasWidth) / float32(rows-1)
	gridSpacingY := float32(canvasHeight) / float32(cols-1)

	// Draw horizontal lines
	for j := 0; j < cols; j++ {
		for i := 0; i < rows-1; i++ {
			x1 := float32(i) * gridSpacingX
			y1 := float32(j) * gridSpacingY
			x2 := float32(i+1) * gridSpacingX
			y2 := float32(j) * gridSpacingY

			displacement1 := float32(normalized[i][j] * effectiveDisplacement)
			displacement2 := float32(normalized[i+1][j] * effectiveDisplacement)

			y1 += displacement1
			y2 += displacement2

			avgPotential := (normalized[i][j] + normalized[i+1][j]) / 2
			lineColor := s.InterpolateColor(avgPotential)

			line := lp.GetLine(lineColor)
			line.Position1 = fyne.NewPos(x1, y1)
			line.Position2 = fyne.NewPos(x2, y2)
			line.StrokeWidth = 1

			objects = append(objects, line)
		}
	}

	// Draw vertical lines
	for i := 0; i < rows; i++ {
		for j := 0; j < cols-1; j++ {
			x1 := float32(i) * gridSpacingX
			y1 := float32(j) * gridSpacingY
			x2 := float32(i) * gridSpacingX
			y2 := float32(j+1) * gridSpacingY

			displacement1 := float32(normalized[i][j] * effectiveDisplacement)
			displacement2 := float32(normalized[i][j+1] * effectiveDisplacement)

			y1 += displacement1
			y2 += displacement2

			avgPotential := (normalized[i][j] + normalized[i][j+1]) / 2
			lineColor := s.InterpolateColor(avgPotential)

			line := lp.GetLine(lineColor)
			line.Position1 = fyne.NewPos(x1, y1)
			line.Position2 = fyne.NewPos(x2, y2)
			line.StrokeWidth = 1

			objects = append(objects, line)
		}
	}

	return objects
}
