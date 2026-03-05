package main

import (
	"math"
	"sync"
)

const (
	// Default canvas size - will be updated dynamically
	defaultCanvasWidth  = 800
	defaultCanvasHeight = 600
	// Default zoom level (pixels per AU)
	defaultDisplayScale = 100.0
)

// ViewPort handles camera/view transformation
type ViewPort struct {
	Zoom          float64 // Zoom level multiplier (1.0 = default)
	PanX, PanY    float64 // Pan offset in AU
	CanvasWidth   float64
	CanvasHeight  float64
	FollowBody    *Body // If not nil, center camera on this body

	// 3D support
	RotationX float64 // Pitch (rotation around X-axis)
	RotationY float64 // Yaw (rotation around Y-axis)
	RotationZ float64 // Roll (rotation around Z-axis)
	Use3D     bool    // Enable 3D rendering
	mu        sync.RWMutex
}

func NewViewPort() *ViewPort {
	return &ViewPort{
		Zoom:         1.0,
		PanX:         0,
		PanY:         0,
		CanvasWidth:  defaultCanvasWidth,
		CanvasHeight: defaultCanvasHeight,
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
	return defaultDisplayScale * vp.Zoom
}

func (vp *ViewPort) SetZoom(zoom float64) {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	// Clamp zoom between 0.01x and 100x
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
func (vp *ViewPort) AutoFit(bodies []Body, sun Body) {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	if len(bodies) == 0 {
		return
	}

	// Find bounding box
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

	// Add some padding (10%)
	rangeX := (maxX - minX) * 1.1
	rangeY := (maxY - minY) * 1.1

	// Calculate zoom to fit
	scaleX := (vp.CanvasWidth * 0.9) / (rangeX / AU)
	scaleY := (vp.CanvasHeight * 0.9) / (rangeY / AU)

	// Use the smaller scale to ensure everything fits
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	vp.Zoom = scale / defaultDisplayScale
	vp.PanX = 0
	vp.PanY = 0
}

// WorldToScreen converts a 3D world position to 2D screen coordinates
func (vp *ViewPort) WorldToScreen(pos Vec3) (float32, float32) {
	vp.mu.RLock()
	displayScale := vp.GetDisplayScale()
	panX := vp.PanX
	panY := vp.PanY
	canvasWidth := vp.CanvasWidth
	canvasHeight := vp.CanvasHeight
	followBody := vp.FollowBody

	// Handle 3D rotation if enabled
	rotX := vp.RotationX
	rotY := vp.RotationY
	rotZ := vp.RotationZ
	use3D := vp.Use3D
	vp.mu.RUnlock()

	// Apply 3D rotation if enabled
	worldPos := pos
	if use3D {
		// Apply rotation matrices (Euler angles)
		// Rotate around X axis (pitch)
		if rotX != 0 {
			cosX := math.Cos(rotX)
			sinX := math.Sin(rotX)
			y := worldPos.Y*cosX - worldPos.Z*sinX
			z := worldPos.Y*sinX + worldPos.Z*cosX
			worldPos.Y = y
			worldPos.Z = z
		}

		// Rotate around Y axis (yaw)
		if rotY != 0 {
			cosY := math.Cos(rotY)
			sinY := math.Sin(rotY)
			x := worldPos.X*cosY + worldPos.Z*sinY
			z := -worldPos.X*sinY + worldPos.Z*cosY
			worldPos.X = x
			worldPos.Z = z
		}

		// Rotate around Z axis (roll)
		if rotZ != 0 {
			cosZ := math.Cos(rotZ)
			sinZ := math.Sin(rotZ)
			x := worldPos.X*cosZ - worldPos.Y*sinZ
			y := worldPos.X*sinZ + worldPos.Y*cosZ
			worldPos.X = x
			worldPos.Y = y
		}
	}

	// Apply follow mode offset
	centerOffsetX := 0.0
	centerOffsetY := 0.0
	if followBody != nil {
		centerOffsetX = followBody.Position.X
		centerOffsetY = followBody.Position.Y
	}

	// Convert meters to pixels with zoom and pan
	// Pan is in AU, so convert to meters first
	x := float32((worldPos.X-centerOffsetX)/AU*displayScale - panX*displayScale + canvasWidth/2)
	y := float32((worldPos.Y-centerOffsetY)/AU*displayScale - panY*displayScale + canvasHeight/2)

	// Add isometric effect for 3D visualization (Z becomes depth)
	if use3D {
		x -= float32(worldPos.Z / AU * displayScale * 0.5)
		y -= float32(worldPos.Z / AU * displayScale * 0.8)
	}

	return x, y
}
