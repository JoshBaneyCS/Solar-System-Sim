package render

import (
	"image"
	"image/color"
	"math"

	"solar-system-sim/internal/math3d"
)

// LightingModel computes Lambertian diffuse shading from the Sun.
type LightingModel struct {
	SunPosition  math3d.Vec3
	AmbientLevel float64 // minimum brightness for dark side (0.0-1.0)
}

// NewLightingModel creates a lighting model with the given sun position.
func NewLightingModel(sunPos math3d.Vec3) *LightingModel {
	return &LightingModel{
		SunPosition:  sunPos,
		AmbientLevel: 0.15,
	}
}

// ApplyDiffuseShading applies Lambertian shading to a circular planet image.
// The light direction is from planetPos toward SunPosition.
// The image is assumed to be a circle (transparent outside).
func (lm *LightingModel) ApplyDiffuseShading(src image.Image, planetPos math3d.Vec3) *image.RGBA {
	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	// Light direction: planet -> sun
	lightDir := lm.SunPosition.Sub(planetPos).Normalize()

	radius := float64(w) / 2.0
	radiusSq := radius * radius

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			srcColor := src.At(bounds.Min.X+x, bounds.Min.Y+y)
			r, g, b, a := srcColor.RGBA()
			if a == 0 {
				continue
			}

			// Map pixel to unit sphere normal
			nx := (float64(x) - radius + 0.5) / radius
			ny := (float64(y) - radius + 0.5) / radius
			distSq := nx*nx + ny*ny
			if distSq > 1.0 {
				continue
			}
			nz := math.Sqrt(1.0 - distSq)

			// Surface normal in view space (assume camera looks along -Z)
			normal := math3d.Vec3{X: nx, Y: -ny, Z: nz}

			// Compute diffuse intensity
			dot := normal.Dot(lightDir)
			intensity := lm.AmbientLevel + (1.0-lm.AmbientLevel)*math.Max(0, dot)
			if intensity > 1.0 {
				intensity = 1.0
			}

			// Apply shading
			dst.Set(x, y, color.RGBA{
				R: uint8(float64(r>>8) * intensity),
				G: uint8(float64(g>>8) * intensity),
				B: uint8(float64(b>>8) * intensity),
				A: uint8(a >> 8),
			})
		}
	}

	return dst
}

// SunGlowImage creates a radial glow image for the Sun.
func SunGlowImage(diameter int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, diameter, diameter))
	center := float64(diameter) / 2.0

	for y := 0; y < diameter; y++ {
		for x := 0; x < diameter; x++ {
			dx := float64(x) - center + 0.5
			dy := float64(y) - center + 0.5
			dist := math.Sqrt(dx*dx+dy*dy) / center

			if dist > 1.0 {
				continue
			}

			// Smooth radial falloff
			alpha := (1.0 - dist*dist) * 0.4
			if alpha < 0 {
				alpha = 0
			}

			dst.Set(x, y, color.RGBA{
				R: 255,
				G: 220,
				B: 100,
				A: uint8(alpha * 255),
			})
		}
	}

	return dst
}
