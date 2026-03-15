//go:build rust_render

package ui

import (
	"solar-system-sim/internal/ffi"
	"solar-system-sim/internal/render"
)

// initGPU attempts to create a GPU renderer. Returns nil on failure (graceful fallback).
func (a *App) initGPU() *render.GPURenderer {
	return render.NewGPURenderer(a.simulator, a.viewport, a.renderer, 800, 600)
}

// detectGPUInfo probes GPU hardware via the Rust backend and populates RuntimeInfo.
func (a *App) detectGPUInfo() {
	info := ffi.DetectGPUHardware()
	if info == nil {
		return
	}
	a.runtimeInfo.GPUVendor = info.Vendor
	a.runtimeInfo.GPUDevice = info.DeviceName
	a.runtimeInfo.GPUBackendName = info.Backend
	a.runtimeInfo.GPUDeviceType = info.DeviceType
	a.runtimeInfo.GPUMaxTexture = info.MaxTexture
	switch info.Tier {
	case 2:
		a.runtimeInfo.GPUTier = "High"
	case 1:
		a.runtimeInfo.GPUTier = "Medium"
	default:
		a.runtimeInfo.GPUTier = "Low"
	}
}
