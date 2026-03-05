package main

import (
	"image/color"
	"math"
	"sync"

	"fyne.io/fyne/v2"
)

// SpacetimeRenderer handles visualization of gravitational field as a warped fabric
type SpacetimeRenderer struct {
	gridResolution int
	cache          SpacetimeCache
	mu             sync.RWMutex
}

// SpacetimeCache stores computed data to avoid recalculation
type SpacetimeCache struct {
	lastZoom       float64
	lastPanX       float64
	lastPanY       float64
	potentials     [][]float64
	needsUpdate    bool
}

const (
	defaultGridResolution      = 80
	displacementFactor         = 50.0
	maxInfluenceDistance       = 10.0 * AU
	cacheInvalidationThreshold = 0.05
	minGridResolution          = 40
	maxGridResolution          = 120
	c                          = 299792458.0 // Speed of light m/s
	curvatureScaleFactor       = 1e11        // Visual scaling for GR curvature
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
		return minGridResolution // 40 for wide view
	} else if zoom < 2.0 {
		return 80 // Medium detail
	} else if zoom < 10.0 {
		return 100 // High detail when zoomed
	}
	return maxGridResolution // 120 when very close
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
	vp *ViewPort,
	planets []Body,
	sun Body,
) [][]float64 {
	vp.mu.RLock()
	zoom := vp.Zoom
	panX := vp.PanX
	panY := vp.PanY
	canvasWidth := vp.CanvasWidth
	canvasHeight := vp.CanvasHeight
	vp.mu.RUnlock()

	// Check if we can use cached values
	if !s.ShouldUpdateCache(zoom, panX, panY) {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return s.cache.potentials
	}

	// Update grid resolution based on zoom
	s.gridResolution = s.GetAdaptiveResolution(zoom)

	potentials := make([][]float64, s.gridResolution)
	for i := range potentials {
		potentials[i] = make([]float64, s.gridResolution)
	}

	displayScale := defaultDisplayScale * zoom

	// Calculate world-space bounds of viewport
	worldWidth := canvasWidth / displayScale * AU
	worldHeight := canvasHeight / displayScale * AU

	viewportCenterX := panX * AU
	viewportCenterY := panY * AU

	// Grid spacing in world coordinates
	gridSpacingX := worldWidth / float64(s.gridResolution-1)
	gridSpacingY := worldHeight / float64(s.gridResolution-1)

	// Zoom-dependent influence distance - include more planets when zoomed in
	effectiveInfluence := maxInfluenceDistance
	if zoom > 2.0 {
		effectiveInfluence = maxInfluenceDistance * zoom
	}

	// Calculate potential at each grid point
	for i := 0; i < s.gridResolution; i++ {
		for j := 0; j < s.gridResolution; j++ {
			// Convert grid indices to world coordinates
			worldX := viewportCenterX - worldWidth/2 + float64(i)*gridSpacingX
			worldY := viewportCenterY - worldHeight/2 + float64(j)*gridSpacingY

			// Calculate General Relativity metric perturbation (spacetime curvature)
			// Uses weak-field Einstein metric: h₀₀ = 2φ/c² = 2GM/(c²r)
			// This represents actual spacetime curvature and time dilation
			curvature := 0.0

			// Sun contribution (dominant)
			rSunX := sun.Position.X - worldX
			rSunY := sun.Position.Y - worldY
			rSun := math.Sqrt(rSunX*rSunX + rSunY*rSunY)
			if rSun > 1e6 { // Avoid division by zero
				// Metric perturbation h₀₀ = 2GM/(c²r)
				h00_sun := 2 * G * sun.Mass / (c * c * rSun)
				curvature += h00_sun
			}

			// Planet contributions (superposition valid in weak field)
			for _, planet := range planets {
				rPlanetX := planet.Position.X - worldX
				rPlanetY := planet.Position.Y - worldY
				rPlanet := math.Sqrt(rPlanetX*rPlanetX + rPlanetY*rPlanetY)

				// Only include if within influence distance and not too close
				if rPlanet > 1e6 && rPlanet < effectiveInfluence {
					// Metric perturbation for this planet
					h00_planet := 2 * G * planet.Mass / (c * c * rPlanet)
					curvature += h00_planet
				}
			}

			// Scale for visualization
			// curvature is dimensionless (h₀₀ ~ 10⁻⁸ at 1 AU from Sun)
			// Scale by 10¹¹ to get visible displacement in pixels
			potentials[i][j] = curvature * curvatureScaleFactor
		}
	}

	// Update cache
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

	// Find min and max
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

	// Avoid division by zero
	if maxPotential == minPotential {
		return potentials, minPotential, maxPotential
	}

	normalized := make([][]float64, len(potentials))
	for i := range potentials {
		normalized[i] = make([]float64, len(potentials[i]))
		for j := range potentials[i] {
			normalized[i][j] = (potentials[i][j] - minPotential) / (maxPotential - minPotential)
		}
	}

	return normalized, minPotential, maxPotential
}

// InterpolateColor maps normalized potential (0-1) to color gradient
// Deep space (0) = Blue → Weak field = Purple → Medium = Red → Strong (1) = Orange
func (s *SpacetimeRenderer) InterpolateColor(normalizedPotential float64) color.Color {
	// Invert so stronger fields (more negative potential) = warmer colors
	t := 1.0 - normalizedPotential

	// Clamp to [0, 1]
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}

	// Four-color gradient: Blue → Purple → Red → Orange
	if t < 0.33 {
		// Blue to Purple
		ratio := t / 0.33
		r := uint8(100 * ratio)
		g := uint8(0)
		b := uint8(50 + 100*ratio)
		return color.RGBA{r, g, b, 180}
	} else if t < 0.66 {
		// Purple to Red
		ratio := (t - 0.33) / 0.33
		r := uint8(100 + 100*ratio)
		g := uint8(0)
		b := uint8(150 - 150*ratio)
		return color.RGBA{r, g, b, 180}
	} else {
		// Red to Orange
		ratio := (t - 0.66) / 0.34
		r := uint8(200 + 55*ratio)
		g := uint8(100 * ratio)
		b := uint8(0)
		return color.RGBA{r, g, b, 180}
	}
}

// RenderGrid generates the warped spacetime grid visualization
func (s *SpacetimeRenderer) RenderGrid(
	renderCache *RenderCache,
	vp *ViewPort,
	planets []Body,
	sun Body,
) []fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, 0)

	// Calculate potential field
	potentials := s.CalculatePotentialField(vp, planets, sun)
	if len(potentials) == 0 {
		return objects
	}

	// Normalize potentials to 0-1 range
	normalized, _, _ := s.NormalizePotentials(potentials)

	vp.mu.RLock()
	canvasWidth := vp.CanvasWidth
	canvasHeight := vp.CanvasHeight
	zoom := vp.Zoom
	vp.mu.RUnlock()

	// Zoom-dependent displacement - more dramatic when zoomed in
	effectiveDisplacement := displacementFactor * (1.0 + zoom/2.0)

	// Grid dimensions
	rows := len(normalized)
	cols := len(normalized[0])

	// Calculate screen positions for grid vertices
	gridSpacingX := float32(canvasWidth) / float32(rows-1)
	gridSpacingY := float32(canvasHeight) / float32(cols-1)

	// Draw horizontal lines
	for j := 0; j < cols; j++ {
		for i := 0; i < rows-1; i++ {
			x1 := float32(i) * gridSpacingX
			y1 := float32(j) * gridSpacingY
			x2 := float32(i+1) * gridSpacingX
			y2 := float32(j) * gridSpacingY

			// Apply displacement based on potential
			displacement1 := float32(normalized[i][j] * effectiveDisplacement)
			displacement2 := float32(normalized[i+1][j] * effectiveDisplacement)

			y1 += displacement1
			y2 += displacement2

			// Color based on average potential
			avgPotential := (normalized[i][j] + normalized[i+1][j]) / 2
			lineColor := s.InterpolateColor(avgPotential)

			line := renderCache.GetLine(lineColor)
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

			// Apply displacement based on potential
			displacement1 := float32(normalized[i][j] * effectiveDisplacement)
			displacement2 := float32(normalized[i][j+1] * effectiveDisplacement)

			y1 += displacement1
			y2 += displacement2

			// Color based on average potential
			avgPotential := (normalized[i][j] + normalized[i][j+1]) / 2
			lineColor := s.InterpolateColor(avgPotential)

			line := renderCache.GetLine(lineColor)
			line.Position1 = fyne.NewPos(x1, y1)
			line.Position2 = fyne.NewPos(x2, y2)
			line.StrokeWidth = 1

			objects = append(objects, line)
		}
	}

	return objects
}
