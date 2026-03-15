package ui

import (
	"fmt"
	"runtime"
)

// RuntimeInfo contains detected hardware and platform information.
type RuntimeInfo struct {
	OS             string
	Arch           string
	IsAppleSilicon bool
	NumCPU         int
	GoVersion      string

	// GPU info (populated when rust_render is active)
	GPUVendor      string
	GPUDevice      string
	GPUBackendName string
	GPUDeviceType  string
	GPUMaxTexture  uint32
	GPUTier        string // "High", "Medium", "Low"
}

// DetectRuntime gathers system information at startup.
func DetectRuntime() RuntimeInfo {
	return RuntimeInfo{
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		IsAppleSilicon: runtime.GOOS == "darwin" && runtime.GOARCH == "arm64",
		NumCPU:         runtime.NumCPU(),
		GoVersion:      runtime.Version(),
	}
}

// String returns a human-readable summary.
func (ri RuntimeInfo) String() string {
	silicon := ""
	if ri.IsAppleSilicon {
		silicon = " (Apple Silicon)"
	}
	s := fmt.Sprintf("OS: %s/%s%s | CPUs: %d | Go: %s",
		ri.OS, ri.Arch, silicon, ri.NumCPU, ri.GoVersion)
	if ri.GPUDevice != "" {
		s += fmt.Sprintf(" | GPU: %s (%s, %s)", ri.GPUDevice, ri.GPUVendor, ri.GPUBackendName)
	}
	return s
}

// GPUBackend returns the actual GPU backend if detected, otherwise infers from OS.
func (ri RuntimeInfo) GPUBackend() string {
	if ri.GPUBackendName != "" {
		return ri.GPUBackendName
	}
	switch ri.OS {
	case "darwin":
		return "Metal"
	case "linux":
		return "Vulkan/OpenGL"
	case "windows":
		return "DirectX/Vulkan"
	default:
		return "Software"
	}
}
