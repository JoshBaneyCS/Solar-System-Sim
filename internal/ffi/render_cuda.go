//go:build cuda_render

package ffi

/*
#cgo LDFLAGS: -L${SRCDIR}/../../native_gpu/cuda -lnative_render_cuda -Wl,-rpath,${SRCDIR}/../../native_gpu/cuda
#cgo LDFLAGS: -lcudart
#include "../../native_gpu/common/native_render.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// GPUHardwareInfo contains GPU hardware details detected by the CUDA backend.
type GPUHardwareInfo struct {
	Vendor     string
	DeviceName string
	Backend    string
	DeviceType string
	MaxTexture uint32
	Tier       uint8
}

func DetectGPUHardware() *GPUHardwareInfo {
	info := C.render_get_hardware_info()
	if info == nil {
		return nil
	}
	defer C.render_free_hardware_info(info)
	result := &GPUHardwareInfo{
		MaxTexture: uint32(info.max_texture_size),
		Tier:       uint8(info.tier),
	}
	if info.vendor != nil {
		result.Vendor = C.GoString(info.vendor)
	}
	if info.device_name != nil {
		result.DeviceName = C.GoString(info.device_name)
	}
	if info.backend != nil {
		result.Backend = C.GoString(info.backend)
	}
	if info.device_type != nil {
		result.DeviceType = C.GoString(info.device_type)
	}
	return result
}

type RustRenderer struct {
	handle *C.Renderer
	width  uint32
	height uint32
}

func NewRustRenderer(width, height uint32) *RustRenderer {
	handle := C.render_create(C.uint32_t(width), C.uint32_t(height))
	if handle == nil {
		return nil
	}
	return &RustRenderer{handle: handle, width: width, height: height}
}

func NewRustRendererWithTextures(width, height uint32, assetDir string) *RustRenderer {
	var handle *C.Renderer
	if assetDir == "" {
		handle = C.render_create(C.uint32_t(width), C.uint32_t(height))
	} else {
		cDir := C.CString(assetDir)
		defer C.free(unsafe.Pointer(cDir))
		handle = C.render_create_with_textures(C.uint32_t(width), C.uint32_t(height), cDir)
	}
	if handle == nil {
		return nil
	}
	return &RustRenderer{handle: handle, width: width, height: height}
}

func (r *RustRenderer) SetCamera(zoom, panX, panY, rotX, rotY, rotZ float64, use3D bool, followX, followY, followZ float64) {
	u3d := C.uint8_t(0)
	if use3D {
		u3d = 1
	}
	C.render_set_camera(r.handle, C.double(zoom), C.double(panX), C.double(panY),
		C.double(rotX), C.double(rotY), C.double(rotZ), u3d,
		C.double(followX), C.double(followY), C.double(followZ))
}

func (r *RustRenderer) SetBodies(nBodies uint32, positions, colors, radii []float64, sunPos, sunColor []float64, sunRadius float64) {
	C.render_set_bodies(r.handle, C.uint32_t(nBodies),
		(*C.double)(unsafe.Pointer(&positions[0])),
		(*C.double)(unsafe.Pointer(&colors[0])),
		(*C.double)(unsafe.Pointer(&radii[0])),
		(*C.double)(unsafe.Pointer(&sunPos[0])),
		(*C.double)(unsafe.Pointer(&sunColor[0])),
		C.double(sunRadius))
}

func (r *RustRenderer) SetTrails(nBodies uint32, trailLengths []uint32, trailPositions, trailColors []float64, showTrails bool) {
	st := C.uint8_t(0)
	if showTrails {
		st = 1
	}
	if len(trailLengths) == 0 {
		C.render_set_trails(r.handle, C.uint32_t(nBodies), nil, nil, nil, st)
		return
	}
	C.render_set_trails(r.handle, C.uint32_t(nBodies),
		(*C.uint32_t)(unsafe.Pointer(&trailLengths[0])),
		(*C.double)(unsafe.Pointer(&trailPositions[0])),
		(*C.double)(unsafe.Pointer(&trailColors[0])), st)
}

func (r *RustRenderer) SetSpacetime(showSpacetime bool, masses []float64, positions []float64, nBodies uint32) {
	ss := C.uint8_t(0)
	if showSpacetime {
		ss = 1
	}
	if len(masses) == 0 {
		C.render_set_spacetime(r.handle, ss, nil, nil, C.uint32_t(0))
		return
	}
	C.render_set_spacetime(r.handle, ss,
		(*C.double)(unsafe.Pointer(&masses[0])),
		(*C.double)(unsafe.Pointer(&positions[0])),
		C.uint32_t(nBodies))
}

func (r *RustRenderer) SetDistanceLine(hasLine bool, x1, y1, z1, x2, y2, z2 float64) {
	hl := C.uint8_t(0)
	if hasLine {
		hl = 1
	}
	C.render_set_distance_line(r.handle, hl,
		C.double(x1), C.double(y1), C.double(z1),
		C.double(x2), C.double(y2), C.double(z2))
}

func (r *RustRenderer) SetRTMode(enabled bool) {
	e := C.uint8_t(0)
	if enabled {
		e = 1
	}
	C.render_set_rt_mode(r.handle, e)
}

func (r *RustRenderer) SetRTQuality(samplesPerFrame, maxBounces uint32) {
	C.render_set_rt_quality(r.handle, C.uint32_t(samplesPerFrame), C.uint32_t(maxBounces))
}

func (r *RustRenderer) RenderFrame() []byte {
	ptr := C.render_frame(r.handle)
	if ptr == nil {
		return nil
	}
	size := int(r.width) * int(r.height) * 4
	return unsafe.Slice((*byte)(unsafe.Pointer(ptr)), size)
}

func (r *RustRenderer) Resize(width, height uint32) {
	r.width = width
	r.height = height
	C.render_resize(r.handle, C.uint32_t(width), C.uint32_t(height))
}

func (r *RustRenderer) Free() {
	if r.handle != nil {
		C.render_free(r.handle)
		r.handle = nil
	}
}
