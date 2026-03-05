package viewport

import (
	"math"
	"sync"

	"solar-system-sim/internal/math3d"
	"solar-system-sim/internal/physics"
	"solar-system-sim/pkg/constants"
)

const (
	// Default canvas size - will be updated dynamically
	DefaultCanvasWidth  = 800
	DefaultCanvasHeight = 600
	// Default zoom level (pixels per AU)
	DefaultDisplayScale = 100.0
)

// ViewPort handles camera/view transformation
type ViewPort struct {
	Zoom          float64 // Zoom level multiplier (1.0 = default)
	PanX, PanY    float64 // Pan offset in AU
	CanvasWidth   float64
	CanvasHeight  float64
	FollowBody    *physics.Body // If not nil, center camera on this body

	// 3D support
	RotationX float64 // Pitch (rotation around X-axis)
	RotationY float64 // Yaw (rotation around Y-axis)
	RotationZ float64 // Roll (rotation around Z-axis)
	Use3D     bool    // Enable 3D rendering
	mu        sync.RWMutex
}

// RLock acquires a read lock on the viewport mutex.
func (vp *ViewPort) RLock() { vp.mu.RLock() }

// RUnlock releases the read lock on the viewport mutex.
func (vp *ViewPort) RUnlock() { vp.mu.RUnlock() }

// Lock acquires a write lock on the viewport mutex.
func (vp *ViewPort) Lock() { vp.mu.Lock() }

// Unlock releases the write lock on the viewport mutex.
func (vp *ViewPort) Unlock() { vp.mu.Unlock() }

func NewViewPort() *ViewPort {
	return &ViewPort{
		Zoom:         1.0,
		PanX:         0,
		PanY:         0,
		CanvasWidth:  DefaultCanvasWidth,
		CanvasHeight: DefaultCanvasHeight,
		FollowBody:   nil,
		RotationX:    math.Pi / 6, // 30° initial pitch for better 3D view
		RotationY:    0,
		RotationZ:    0,
		Use3D:        false,
	}
}

func (vp *ViewPort) GetDisplayScale() float64 {
	vp.mu.RLock()
	defer vp.mu.RUnlock()
	return DefaultDisplayScale * vp.Zoom
}

func (vp *ViewPort) SetZoom(zoom float64) {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	if zoom < 0.01 {
		zoom = 0.01
	}
	if zoom > 100.0 {
		zoom = 100.0
	}
	vp.Zoom = zoom
}

func (vp *ViewPort) SetPan(panX, panY float64) {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	vp.PanX = panX
	vp.PanY = panY
}

func (vp *ViewPort) AdjustPan(deltaX, deltaY float64) {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	vp.PanX += deltaX
	vp.PanY += deltaY
}

// UpdateCanvasSize updates the viewport canvas dimensions
func (vp *ViewPort) UpdateCanvasSize(width, height float64) {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	vp.CanvasWidth = width
	vp.CanvasHeight = height
}

// AutoFit calculates zoom to fit all bodies on screen
func (vp *ViewPort) AutoFit(bodies []physics.Body, sun physics.Body) {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	if len(bodies) == 0 {
		return
	}

	minX, maxX := bodies[0].Position.X, bodies[0].Position.X
	minY, maxY := bodies[0].Position.Y, bodies[0].Position.Y

	for _, body := range bodies {
		if body.Position.X < minX {
			minX = body.Position.X
		}
		if body.Position.X > maxX {
			maxX = body.Position.X
		}
		if body.Position.Y < minY {
			minY = body.Position.Y
		}
		if body.Position.Y > maxY {
			maxY = body.Position.Y
		}
	}

	rangeX := (maxX - minX) * 1.1
	rangeY := (maxY - minY) * 1.1

	scaleX := (vp.CanvasWidth * 0.9) / (rangeX / constants.AU)
	scaleY := (vp.CanvasHeight * 0.9) / (rangeY / constants.AU)

	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	vp.Zoom = scale / DefaultDisplayScale
	vp.PanX = 0
	vp.PanY = 0
}

// WorldToScreen converts a 3D world position to 2D screen coordinates
func (vp *ViewPort) WorldToScreen(pos math3d.Vec3) (float32, float32) {
	vp.mu.RLock()
	displayScale := DefaultDisplayScale * vp.Zoom
	panX := vp.PanX
	panY := vp.PanY
	canvasWidth := vp.CanvasWidth
	canvasHeight := vp.CanvasHeight
	followBody := vp.FollowBody

	rotX := vp.RotationX
	rotY := vp.RotationY
	rotZ := vp.RotationZ
	use3D := vp.Use3D
	vp.mu.RUnlock()

	worldPos := pos
	if use3D {
		if rotX != 0 {
			cosX := math.Cos(rotX)
			sinX := math.Sin(rotX)
			y := worldPos.Y*cosX - worldPos.Z*sinX
			z := worldPos.Y*sinX + worldPos.Z*cosX
			worldPos.Y = y
			worldPos.Z = z
		}

		if rotY != 0 {
			cosY := math.Cos(rotY)
			sinY := math.Sin(rotY)
			x := worldPos.X*cosY + worldPos.Z*sinY
			z := -worldPos.X*sinY + worldPos.Z*cosY
			worldPos.X = x
			worldPos.Z = z
		}

		if rotZ != 0 {
			cosZ := math.Cos(rotZ)
			sinZ := math.Sin(rotZ)
			x := worldPos.X*cosZ - worldPos.Y*sinZ
			y := worldPos.X*sinZ + worldPos.Y*cosZ
			worldPos.X = x
			worldPos.Y = y
		}
	}

	centerOffsetX := 0.0
	centerOffsetY := 0.0
	if followBody != nil {
		centerOffsetX = followBody.Position.X
		centerOffsetY = followBody.Position.Y
	}

	x := float32((worldPos.X-centerOffsetX)/constants.AU*displayScale - panX*displayScale + canvasWidth/2)
	y := float32((worldPos.Y-centerOffsetY)/constants.AU*displayScale - panY*displayScale + canvasHeight/2)

	if use3D {
		x -= float32(worldPos.Z / constants.AU * displayScale * 0.5)
		y -= float32(worldPos.Z / constants.AU * displayScale * 0.8)
	}

	return x, y
}
