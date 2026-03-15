package ui

import (
	"runtime"
	"testing"
)

func TestDetectRuntime(t *testing.T) {
	ri := DetectRuntime()

	if ri.OS != runtime.GOOS {
		t.Errorf("expected OS %s, got %s", runtime.GOOS, ri.OS)
	}
	if ri.Arch != runtime.GOARCH {
		t.Errorf("expected Arch %s, got %s", runtime.GOARCH, ri.Arch)
	}
	if ri.NumCPU < 1 {
		t.Error("expected at least 1 CPU")
	}
	if ri.GoVersion == "" {
		t.Error("expected non-empty Go version")
	}
}

func TestDetectRuntime_AppleSilicon(t *testing.T) {
	ri := DetectRuntime()
	expected := runtime.GOOS == "darwin" && runtime.GOARCH == "arm64"
	if ri.IsAppleSilicon != expected {
		t.Errorf("IsAppleSilicon: expected %v, got %v", expected, ri.IsAppleSilicon)
	}
}

func TestRuntimeInfo_String(t *testing.T) {
	ri := DetectRuntime()
	s := ri.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
}

func TestRuntimeInfo_GPUBackend(t *testing.T) {
	ri := DetectRuntime()
	backend := ri.GPUBackend()
	if backend == "" {
		t.Error("expected non-empty GPU backend")
	}
}
