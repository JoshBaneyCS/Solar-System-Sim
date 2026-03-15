package viewport

import (
	"math"
	"testing"

	"solar-system-sim/internal/math3d"
	"solar-system-sim/internal/physics"
	"solar-system-sim/pkg/constants"
)

func TestNewViewPort(t *testing.T) {
	vp := NewViewPort()
	if vp.Zoom != 1.0 {
		t.Errorf("expected zoom 1.0, got %f", vp.Zoom)
	}
	if vp.PanX != 0 || vp.PanY != 0 {
		t.Error("expected zero pan")
	}
	if vp.Use3D {
		t.Error("expected 3D disabled by default")
	}
}

func TestSetZoom_Clamping(t *testing.T) {
	vp := NewViewPort()
	vp.SetZoom(0.001)
	if vp.Zoom != 0.01 {
		t.Errorf("expected min clamp 0.01, got %f", vp.Zoom)
	}
	vp.SetZoom(2e7)
	if vp.Zoom != 1e7 {
		t.Errorf("expected max clamp 1e7, got %f", vp.Zoom)
	}
}

func TestAdjustZoom(t *testing.T) {
	vp := NewViewPort()
	vp.AdjustZoom(2.0)
	if vp.Zoom != 2.0 {
		t.Errorf("expected zoom 2.0, got %f", vp.Zoom)
	}
	vp.AdjustZoom(0.5)
	if vp.Zoom != 1.0 {
		t.Errorf("expected zoom 1.0, got %f", vp.Zoom)
	}
}

func TestAdjustZoom_Clamping(t *testing.T) {
	vp := NewViewPort()
	vp.Zoom = 0.02
	vp.AdjustZoom(0.1) // 0.02 * 0.1 = 0.002
	if vp.Zoom != 0.01 {
		t.Errorf("expected min clamp 0.01, got %f", vp.Zoom)
	}
}

func TestAdjustPan(t *testing.T) {
	vp := NewViewPort()
	vp.AdjustPan(1.0, 2.0)
	if vp.PanX != 1.0 || vp.PanY != 2.0 {
		t.Errorf("expected pan (1.0, 2.0), got (%f, %f)", vp.PanX, vp.PanY)
	}
}

func TestAdjustRotation(t *testing.T) {
	vp := NewViewPort()
	initial := vp.RotationX
	vp.AdjustRotation(0.1, 0.2)
	if math.Abs(vp.RotationX-(initial+0.1)) > 1e-10 {
		t.Error("rotation X not adjusted")
	}
	if math.Abs(vp.RotationY-0.2) > 1e-10 {
		t.Error("rotation Y not adjusted")
	}
}

func TestWorldToScreen_Origin(t *testing.T) {
	vp := NewViewPort()
	vp.CanvasWidth = 800
	vp.CanvasHeight = 600
	vp.Use3D = false

	x, y := vp.WorldToScreen(math3d.Vec3{X: 0, Y: 0, Z: 0})
	if math.Abs(float64(x)-400) > 1 || math.Abs(float64(y)-300) > 1 {
		t.Errorf("origin should map to center (400,300), got (%f,%f)", x, y)
	}
}

func TestWorldToScreen_ZoomDoubles(t *testing.T) {
	vp := NewViewPort()
	vp.CanvasWidth = 800
	vp.CanvasHeight = 600
	vp.Use3D = false

	pos := math3d.Vec3{X: constants.AU, Y: 0, Z: 0}

	vp.SetZoom(1.0)
	x1, _ := vp.WorldToScreen(pos)

	vp.SetZoom(2.0)
	x2, _ := vp.WorldToScreen(pos)

	// At zoom 2x, the pixel distance from center should be double
	center := float32(400)
	dist1 := x1 - center
	dist2 := x2 - center

	ratio := float64(dist2) / float64(dist1)
	if math.Abs(ratio-2.0) > 0.01 {
		t.Errorf("zoom 2x should double pixel distance, ratio was %f", ratio)
	}
}

func TestAutoFit(t *testing.T) {
	vp := NewViewPort()
	vp.CanvasWidth = 800
	vp.CanvasHeight = 600

	bodies := []physics.Body{
		{Position: math3d.Vec3{X: -constants.AU, Y: 0, Z: 0}},
		{Position: math3d.Vec3{X: constants.AU, Y: 0, Z: 0}},
	}
	sun := physics.Body{Position: math3d.Vec3{}}

	vp.AutoFit(bodies, sun)

	if vp.Zoom <= 0 {
		t.Error("expected positive zoom after auto-fit")
	}
	if vp.PanX != 0 || vp.PanY != 0 {
		t.Error("expected zero pan after auto-fit")
	}
}
