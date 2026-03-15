package render

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"

	"solar-system-sim/internal/math3d"
	"solar-system-sim/internal/physics"
	"solar-system-sim/internal/viewport"
	"solar-system-sim/pkg/constants"
)

// BeltRenderer handles visual rendering of asteroid belt particles.
// These are NOT N-body simulated — positions are computed from Keplerian elements.
type BeltRenderer struct {
	particles []physics.BeltParticle
	cache     *RenderCache
}

// NewBeltRenderer creates a belt renderer with the given particle data.
func NewBeltRenderer(particles []physics.BeltParticle, cache *RenderCache) *BeltRenderer {
	return &BeltRenderer{
		particles: particles,
		cache:     cache,
	}
}

// Render computes belt particle positions and returns canvas objects.
func (br *BeltRenderer) Render(vp *viewport.ViewPort, simTime float64) []fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, 0, len(br.particles))

	vp.RLock()
	canvasWidth := vp.CanvasWidth
	canvasHeight := vp.CanvasHeight
	vp.RUnlock()

	// Precompute a palette of belt colors for variety
	beltColors := []color.RGBA{
		{140, 130, 115, 180}, // gray
		{160, 145, 120, 170}, // tan
		{120, 110, 100, 190}, // dark gray
		{170, 155, 130, 160}, // light brown
		{130, 120, 110, 175}, // medium gray
	}

	for i, p := range br.particles {
		pos := beltParticlePosition(p, simTime)
		x, y := vp.WorldToScreen(pos)

		// Skip off-screen particles
		if x < -10 || x > float32(canvasWidth)+10 || y < -10 || y > float32(canvasHeight)+10 {
			continue
		}

		c := beltColors[i%len(beltColors)]
		dot := br.cache.GetCircle(c)
		// Vary size: most are 1-2px, some are 3px
		size := float32(1)
		if i%3 == 0 {
			size = 2
		}
		if i%17 == 0 {
			size = 3
		}
		dot.Resize(fyne.NewSize(size, size))
		dot.Move(fyne.NewPos(x-size/2, y-size/2))
		objects = append(objects, dot)
	}

	return objects
}

// beltParticlePosition computes the 2D heliocentric position of a belt particle
// at the given simulation time using Keplerian mechanics.
func beltParticlePosition(p physics.BeltParticle, simTime float64) math3d.Vec3 {
	a := p.SemiMajorAxis * constants.AU
	e := p.Eccentricity

	// Mean motion: n = sqrt(GM/a^3)
	GM := constants.G * 1.989e30 // Sun mass
	n := math.Sqrt(GM / (a * a * a))

	// Mean anomaly
	M := p.InitialAnomaly + n*simTime
	M = math.Mod(M, 2*math.Pi)

	// Solve Kepler's equation: E - e*sin(E) = M (Newton's method, 5 iterations)
	E := M
	for k := 0; k < 5; k++ {
		E = E - (E-e*math.Sin(E)-M)/(1-e*math.Cos(E))
	}

	// True anomaly
	nu := 2 * math.Atan2(math.Sqrt(1+e)*math.Sin(E/2), math.Sqrt(1-e)*math.Cos(E/2))

	// Radial distance
	r := a * (1 - e*math.Cos(E))

	// Position in orbital plane
	xOrb := r * math.Cos(nu)
	yOrb := r * math.Sin(nu)

	// Apply inclination (simplified — no Omega/omega for visual particles)
	x := xOrb
	y := yOrb * math.Cos(p.Inclination)
	z := yOrb * math.Sin(p.Inclination)

	return math3d.Vec3{X: x, Y: y, Z: z}
}
