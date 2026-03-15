package render

import (
	"image"
	"image/color"
	"testing"

	"solar-system-sim/internal/math3d"
)

func TestLightingModel_LitSideBrighter(t *testing.T) {
	// Create a white circle image
	size := 50
	src := image.NewRGBA(image.Rect(0, 0, size, size))
	center := float64(size) / 2.0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - center + 0.5
			dy := float64(y) - center + 0.5
			if dx*dx+dy*dy <= center*center {
				src.Set(x, y, color.RGBA{200, 200, 200, 255})
			}
		}
	}

	// Light from the right (positive X)
	lm := NewLightingModel(math3d.Vec3{X: 1e11, Y: 0, Z: 0})
	planetPos := math3d.Vec3{X: 0, Y: 0, Z: 0}

	result := lm.ApplyDiffuseShading(src, planetPos)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Right side (lit) should be brighter than left side (dark)
	litR, _, _, _ := result.At(size-5, size/2).RGBA()
	darkR, _, _, _ := result.At(5, size/2).RGBA()

	if litR <= darkR {
		t.Errorf("lit side (%d) should be brighter than dark side (%d)", litR>>8, darkR>>8)
	}
}

func TestLightingModel_AmbientLevel(t *testing.T) {
	lm := NewLightingModel(math3d.Vec3{X: 1e11, Y: 0, Z: 0})
	if lm.AmbientLevel != 0.15 {
		t.Errorf("expected ambient 0.15, got %f", lm.AmbientLevel)
	}
}

func TestSunGlowImage(t *testing.T) {
	glow := SunGlowImage(40)
	if glow == nil {
		t.Fatal("expected non-nil glow image")
	}

	bounds := glow.Bounds()
	if bounds.Dx() != 40 || bounds.Dy() != 40 {
		t.Errorf("expected 40x40, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Center should have some alpha
	_, _, _, a := glow.At(20, 20).RGBA()
	if a == 0 {
		t.Error("center of glow should have some alpha")
	}

	// Far corner should be transparent
	_, _, _, a = glow.At(0, 0).RGBA()
	if a != 0 {
		t.Error("corner of glow should be transparent")
	}
}
