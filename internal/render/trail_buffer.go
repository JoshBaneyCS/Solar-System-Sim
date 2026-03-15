package render

import (
	"image"
	"image/color"

	"solar-system-sim/internal/math3d"
	"solar-system-sim/internal/physics"
	"solar-system-sim/internal/viewport"
)

// TrailBuffer renders orbital trails into a single image buffer
// instead of creating thousands of individual Fyne line objects.
type TrailBuffer struct {
	img    *image.RGBA
	width  int
	height int
}

// NewTrailBuffer creates a trail buffer.
func NewTrailBuffer() *TrailBuffer {
	return &TrailBuffer{}
}

// Render draws all planet trails into a single image and returns it.
func (tb *TrailBuffer) Render(planets []physics.Body, vp *viewport.ViewPort, canvasWidth, canvasHeight float64) *image.RGBA {
	w := int(canvasWidth)
	h := int(canvasHeight)
	if w <= 0 || h <= 0 {
		return nil
	}

	// Resize buffer if needed
	if tb.img == nil || tb.width != w || tb.height != h {
		tb.img = image.NewRGBA(image.Rect(0, 0, w, h))
		tb.width = w
		tb.height = h
	} else {
		// Clear buffer — zero out all pixels
		for i := range tb.img.Pix {
			tb.img.Pix[i] = 0
		}
	}

	for _, planet := range planets {
		if len(planet.Trail) < 2 {
			continue
		}

		// Downsample: max 200 segments (same as original)
		step := 1
		if len(planet.Trail) > 200 {
			step = len(planet.Trail) / 200
		}

		// Extract base color
		var cr, cg, cb uint8
		if c, ok := planet.Color.(color.RGBA); ok {
			cr, cg, cb = c.R, c.G, c.B
		} else {
			r, g, b, _ := planet.Color.RGBA()
			cr, cg, cb = uint8(r>>8), uint8(g>>8), uint8(b>>8)
		}

		trailLen := float64(len(planet.Trail))

		for j := 0; j < len(planet.Trail)-step; j += step {
			i0 := j - step
			if i0 < 0 {
				i0 = 0
			}
			i1 := j
			i2 := j + step
			i3 := j + 2*step
			if i3 >= len(planet.Trail) {
				i3 = len(planet.Trail) - 1
			}
			p0 := planet.Trail[i0]
			p1 := planet.Trail[i1]
			p2 := planet.Trail[i2]
			p3 := planet.Trail[i3]

			alpha := uint8(float64(j) / trailLen * 255)

			const nSub = 4
			prevX, prevY := vp.WorldToScreen(p1)
			for k := 1; k <= nSub; k++ {
				t := float64(k) / float64(nSub)
				pt := math3d.CatmullRom(p0, p1, p2, p3, t)
				curX, curY := vp.WorldToScreen(pt)

				// Draw if either endpoint is on screen
				if isOnScreenF(prevX, prevY, canvasWidth, canvasHeight) ||
					isOnScreenF(curX, curY, canvasWidth, canvasHeight) {
					tb.drawLine(int(prevX), int(prevY), int(curX), int(curY), cr, cg, cb, alpha)
				}
				prevX, prevY = curX, curY
			}
		}
	}

	return tb.img
}

// drawLine uses Bresenham's algorithm to draw a line directly into the pixel buffer.
func (tb *TrailBuffer) drawLine(x0, y0, x1, y1 int, r, g, b, a uint8) {
	dx := x1 - x0
	dy := y1 - y0
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}

	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}

	err := dx - dy

	w := tb.width
	h := tb.height
	stride := tb.img.Stride
	pix := tb.img.Pix

	for {
		// Plot pixel with alpha blending
		if x0 >= 0 && x0 < w && y0 >= 0 && y0 < h {
			off := y0*stride + x0*4
			// Alpha blend: existing pixel + new pixel
			existA := pix[off+3]
			if existA == 0 {
				pix[off+0] = r
				pix[off+1] = g
				pix[off+2] = b
				pix[off+3] = a
			} else {
				// Simple over blending
				srcA := uint16(a)
				dstA := uint16(existA)
				outA := srcA + dstA*(255-srcA)/255
				if outA > 0 {
					pix[off+0] = uint8((uint16(r)*srcA + uint16(pix[off+0])*dstA*(255-srcA)/255) / outA)
					pix[off+1] = uint8((uint16(g)*srcA + uint16(pix[off+1])*dstA*(255-srcA)/255) / outA)
					pix[off+2] = uint8((uint16(b)*srcA + uint16(pix[off+2])*dstA*(255-srcA)/255) / outA)
					pix[off+3] = uint8(outA)
				}
			}
		}

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func isOnScreenF(x, y float32, canvasWidth, canvasHeight float64) bool {
	margin := float32(50)
	return x >= -margin && x <= float32(canvasWidth)+margin &&
		y >= -margin && y <= float32(canvasHeight)+margin
}
