//go:build !rust_render && !metal_render && !cuda_render && !rocm_render

package ffi

// GPUHardwareInfo contains GPU hardware details (stub for non-Rust builds).
type GPUHardwareInfo struct {
	Vendor     string
	DeviceName string
	Backend    string
	DeviceType string
	MaxTexture uint32
	Tier       uint8
}

// DetectGPUHardware returns nil when rust_render is not enabled.
func DetectGPUHardware() *GPUHardwareInfo {
	return nil
}
