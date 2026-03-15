package render

import (
	"image"
	"math"
	"runtime"
	"sync"

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
// Uses direct Pix[] access and row parallelism for performance.
func (lm *LightingModel) ApplyDiffuseShading(src image.Image, planetPos math3d.Vec3) *image.RGBA {
	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	// Light direction: planet -> sun
	lightDir := lm.SunPosition.Sub(planetPos).Normalize()
	ambient := lm.AmbientLevel
	diffScale := 1.0 - ambient

	radius := float64(w) / 2.0

	// Try to get direct pixel access from source
	type pixReader interface {
		Pix() []uint8
		Stride() int
	}

	var srcPix []uint8
	var srcStride int
	var srcMinX, srcMinY int

	switch s := src.(type) {
	case *image.RGBA:
		srcPix = s.Pix
		srcStride = s.Stride
		srcMinX = s.Rect.Min.X
		srcMinY = s.Rect.Min.Y
	case *image.NRGBA:
		srcPix = s.Pix
		srcStride = s.Stride
		srcMinX = s.Rect.Min.X
		srcMinY = s.Rect.Min.Y
	}

	dstPix := dst.Pix
	dstStride := dst.Stride

	// Shade a range of rows
	shadeRows := func(yStart, yEnd int) {
		for y := yStart; y < yEnd; y++ {
			ny := (float64(y) - radius + 0.5) / radius
			nySq := ny * ny

			for x := 0; x < w; x++ {
				nx := (float64(x) - radius + 0.5) / radius
				distSq := nx*nx + nySq
				if distSq > 1.0 {
					continue
				}

				// Read source pixel
				var sr, sg, sb, sa uint8
				if srcPix != nil {
					off := (y+srcMinY-bounds.Min.Y)*srcStride + (x+srcMinX-bounds.Min.X)*4
					sr = srcPix[off+0]
					sg = srcPix[off+1]
					sb = srcPix[off+2]
					sa = srcPix[off+3]
				} else {
					r, g, b, a := src.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
					sr, sg, sb, sa = uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)
				}

				if sa == 0 {
					continue
				}

				nz := math.Sqrt(1.0 - distSq)

				// Compute diffuse intensity
				dot := nx*lightDir.X + (-ny)*lightDir.Y + nz*lightDir.Z
				intensity := ambient + diffScale*dot
				if intensity < ambient {
					intensity = ambient
				}
				if intensity > 1.0 {
					intensity = 1.0
				}

				// Write directly to dst Pix[]
				dstOff := y*dstStride + x*4
				dstPix[dstOff+0] = uint8(float64(sr) * intensity)
				dstPix[dstOff+1] = uint8(float64(sg) * intensity)
				dstPix[dstOff+2] = uint8(float64(sb) * intensity)
				dstPix[dstOff+3] = sa
			}
		}
	}

	// Parallelize for large images
	nCPU := runtime.NumCPU()
	if h > 100 && nCPU > 1 {
		var wg sync.WaitGroup
		bandHeight := (h + nCPU - 1) / nCPU
		for i := 0; i < nCPU; i++ {
			yStart := i * bandHeight
			yEnd := yStart + bandHeight
			if yEnd > h {
				yEnd = h
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				shadeRows(yStart, yEnd)
			}()
		}
		wg.Wait()
	} else {
		shadeRows(0, h)
	}

	return dst
}

// SunGlowImage creates a radial glow image for the Sun.
func SunGlowImage(diameter int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, diameter, diameter))
	center := float64(diameter) / 2.0
	pix := dst.Pix
	stride := dst.Stride

	for y := 0; y < diameter; y++ {
		dy := float64(y) - center + 0.5
		dySq := dy * dy
		rowOff := y * stride

		for x := 0; x < diameter; x++ {
			dx := float64(x) - center + 0.5
			distSq := (dx*dx + dySq) / (center * center)

			if distSq > 1.0 {
				continue
			}

			// Smooth radial falloff
			alpha := (1.0 - distSq) * 0.4
			if alpha < 0 {
				continue
			}

			off := rowOff + x*4
			pix[off+0] = 255
			pix[off+1] = 220
			pix[off+2] = 100
			pix[off+3] = uint8(alpha * 255)
		}
	}

	return dst
}
