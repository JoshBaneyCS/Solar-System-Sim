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
	return fmt.Sprintf("OS: %s/%s%s | CPUs: %d | Go: %s",
		ri.OS, ri.Arch, silicon, ri.NumCPU, ri.GoVersion)
}

// GPUBackend returns a description of the expected GPU backend.
func (ri RuntimeInfo) GPUBackend() string {
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
