package render

import (
	"image"
	"image/color"
	"math"

	"solar-system-sim/internal/math3d"
	"solar-system-sim/internal/physics"
	"solar-system-sim/internal/viewport"
	"solar-system-sim/pkg/constants"
)

// BeltRenderer handles visual rendering of asteroid belt particles.
// These are NOT N-body simulated — positions are computed from Keplerian elements.
// Renders particles to an image buffer instead of individual Fyne objects.
type BeltRenderer struct {
	particles []physics.BeltParticle
	img       *image.RGBA
	width     int
	height    int
}

// NewBeltRenderer creates a belt renderer with the given particle data.
func NewBeltRenderer(particles []physics.BeltParticle, cache *RenderCache) *BeltRenderer {
	return &BeltRenderer{
		particles: particles,
	}
}

// Precomputed belt colors (RGBA values)
var beltColors = []color.RGBA{
	{140, 130, 115, 180}, // gray
	{160, 145, 120, 170}, // tan
	{120, 110, 100, 190}, // dark gray
	{170, 155, 130, 160}, // light brown
	{130, 120, 110, 175}, // medium gray
}

// RenderToImage computes belt particle positions and draws them into an image buffer.
func (br *BeltRenderer) RenderToImage(vp *viewport.ViewPort, simTime float64, canvasWidth, canvasHeight float64) *image.RGBA {
	w := int(canvasWidth)
	h := int(canvasHeight)
	if w <= 0 || h <= 0 {
		return nil
	}

	// Resize buffer if needed
	if br.img == nil || br.width != w || br.height != h {
		br.img = image.NewRGBA(image.Rect(0, 0, w, h))
		br.width = w
		br.height = h
	} else {
		// Clear buffer
		for i := range br.img.Pix {
			br.img.Pix[i] = 0
		}
	}

	pix := br.img.Pix
	stride := br.img.Stride

	for i, p := range br.particles {
		pos := beltParticlePosition(p, simTime)
		x, y := vp.WorldToScreen(pos)

		// Skip off-screen particles
		ix := int(x)
		iy := int(y)
		if ix < 0 || ix >= w || iy < 0 || iy >= h {
			continue
		}

		c := beltColors[i%len(beltColors)]

		// Vary size: most are 1px, some are 2px
		size := 1
		if i%3 == 0 {
			size = 2
		}
		if i%17 == 0 {
			size = 3
		}

		// Draw dot directly into pixel buffer
		for dy := 0; dy < size; dy++ {
			py := iy + dy
			if py >= h {
				break
			}
			rowOff := py * stride
			for dx := 0; dx < size; dx++ {
				px := ix + dx
				if px >= w {
					break
				}
				off := rowOff + px*4
				// Simple alpha blend
				existA := pix[off+3]
				if existA == 0 {
					pix[off+0] = c.R
					pix[off+1] = c.G
					pix[off+2] = c.B
					pix[off+3] = c.A
				}
			}
		}
	}

	return br.img
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
