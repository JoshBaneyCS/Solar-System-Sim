package assets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAssetDir_Development(t *testing.T) {
	// Create a temp assets dir in the working directory
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	os.MkdirAll("assets/textures", 0755)

	dir, err := ResolveAssetDir()
	if err != nil {
		t.Fatalf("expected to find assets dir: %v", err)
	}
	if dir == "" {
		t.Fatal("expected non-empty dir")
	}
}

func TestResolveAssetDir_NotFound(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	_, err := ResolveAssetDir()
	if err == nil {
		t.Error("expected error when assets dir not found")
	}
}

func TestResolveTextureDir(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	os.MkdirAll("assets/textures", 0755)

	dir, err := ResolveTextureDir()
	if err != nil {
		t.Fatalf("expected to find textures dir: %v", err)
	}

	expected, _ := filepath.EvalSymlinks(filepath.Join(tmpDir, "assets", "textures"))
	abs, _ := filepath.EvalSymlinks(dir)
	if abs != expected {
		t.Errorf("expected %s, got %s", expected, abs)
	}
}
