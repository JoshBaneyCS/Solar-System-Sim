//go:build rust_render

package ffi

/*
#cgo LDFLAGS: -L${SRCDIR}/../../crates/render_core/target/release -lrender_core -Wl,-rpath,${SRCDIR}/../../crates/render_core/target/release
#cgo LDFLAGS: -framework QuartzCore -framework Metal -framework Foundation -framework CoreGraphics
#include "../../crates/render_core/include/render_core.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// RustRenderer wraps the Rust render_core GPU renderer handle.
type RustRenderer struct {
	handle *C.Renderer
	width  uint32
	height uint32
}

// NewRustRenderer creates a new GPU renderer. Returns nil on failure.
func NewRustRenderer(width, height uint32) *RustRenderer {
	handle := C.render_create(C.uint32_t(width), C.uint32_t(height))
	if handle == nil {
		return nil
	}
	return &RustRenderer{handle: handle, width: width, height: height}
}

// SetCamera updates camera parameters.
func (r *RustRenderer) SetCamera(zoom, panX, panY, rotX, rotY, rotZ float64, use3D bool, followX, followY, followZ float64) {
	u3d := C.uint8_t(0)
	if use3D {
		u3d = 1
	}
	C.render_set_camera(r.handle,
		C.double(zoom), C.double(panX), C.double(panY),
		C.double(rotX), C.double(rotY), C.double(rotZ),
		u3d,
		C.double(followX), C.double(followY), C.double(followZ),
	)
}

// SetBodies sets planet body data for rendering.
// positions: flat [x0,y0,z0, x1,y1,z1, ...] in meters
// colors: flat [r0,g0,b0,a0, ...] in 0-1 range
// radii: display radius in pixels per body
// sunPos: [x,y,z], sunColor: [r,g,b,a], sunRadius: pixels
func (r *RustRenderer) SetBodies(nBodies uint32, positions, colors, radii []float64, sunPos, sunColor []float64, sunRadius float64) {
	C.render_set_bodies(r.handle,
		C.uint32_t(nBodies),
		(*C.double)(unsafe.Pointer(&positions[0])),
		(*C.double)(unsafe.Pointer(&colors[0])),
		(*C.double)(unsafe.Pointer(&radii[0])),
		(*C.double)(unsafe.Pointer(&sunPos[0])),
		(*C.double)(unsafe.Pointer(&sunColor[0])),
		C.double(sunRadius),
	)
}

// SetTrails sets trail rendering data.
// trailLengths: number of trail points per body
// trailPositions: flat [x,y,z,...] for all trail points concatenated
// trailColors: flat [r,g,b,a] per body
func (r *RustRenderer) SetTrails(nBodies uint32, trailLengths []uint32, trailPositions, trailColors []float64, showTrails bool) {
	st := C.uint8_t(0)
	if showTrails {
		st = 1
	}
	if len(trailLengths) == 0 {
		C.render_set_trails(r.handle, C.uint32_t(nBodies), nil, nil, nil, st)
		return
	}
	C.render_set_trails(r.handle,
		C.uint32_t(nBodies),
		(*C.uint32_t)(unsafe.Pointer(&trailLengths[0])),
		(*C.double)(unsafe.Pointer(&trailPositions[0])),
		(*C.double)(unsafe.Pointer(&trailColors[0])),
		st,
	)
}

// SetSpacetime sets spacetime grid visualization data.
// masses and positions include sun as first element.
func (r *RustRenderer) SetSpacetime(showSpacetime bool, masses []float64, positions []float64, nBodies uint32) {
	ss := C.uint8_t(0)
	if showSpacetime {
		ss = 1
	}
	if len(masses) == 0 {
		C.render_set_spacetime(r.handle, ss, nil, nil, C.uint32_t(0))
		return
	}
	C.render_set_spacetime(r.handle,
		ss,
		(*C.double)(unsafe.Pointer(&masses[0])),
		(*C.double)(unsafe.Pointer(&positions[0])),
		C.uint32_t(nBodies),
	)
}

// SetDistanceLine sets or clears the distance measurement line.
func (r *RustRenderer) SetDistanceLine(hasLine bool, x1, y1, z1, x2, y2, z2 float64) {
	hl := C.uint8_t(0)
	if hasLine {
		hl = 1
	}
	C.render_set_distance_line(r.handle, hl,
		C.double(x1), C.double(y1), C.double(z1),
		C.double(x2), C.double(y2), C.double(z2),
	)
}

// RenderFrame renders a frame and returns the RGBA pixel data.
// The returned slice references Rust-owned memory valid until the next RenderFrame or Free call.
func (r *RustRenderer) RenderFrame() []byte {
	ptr := C.render_frame(r.handle)
	if ptr == nil {
		return nil
	}
	size := int(r.width) * int(r.height) * 4
	return unsafe.Slice((*byte)(unsafe.Pointer(ptr)), size)
}

// Resize changes the render target size.
func (r *RustRenderer) Resize(width, height uint32) {
	r.width = width
	r.height = height
	C.render_resize(r.handle, C.uint32_t(width), C.uint32_t(height))
}

// Free releases the GPU renderer.
func (r *RustRenderer) Free() {
	if r.handle != nil {
		C.render_free(r.handle)
		r.handle = nil
	}
}
