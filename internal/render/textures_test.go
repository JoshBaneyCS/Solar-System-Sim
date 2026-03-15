package render

import (
	"image"
	"image/color"
	"testing"
)

func TestMakeCircularImage(t *testing.T) {
	// Create a 10x10 red image
	src := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			src.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}

	result := makeCircularImage(src, 20)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	bounds := result.Bounds()
	if bounds.Dx() != 20 || bounds.Dy() != 20 {
		t.Errorf("expected 20x20, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Center pixel should be non-transparent (inside circle)
	_, _, _, a := result.At(10, 10).RGBA()
	if a == 0 {
		t.Error("center pixel should be non-transparent")
	}

	// Corner pixel should be transparent (outside circle)
	_, _, _, a = result.At(0, 0).RGBA()
	if a != 0 {
		t.Error("corner pixel should be transparent")
	}
}

func TestNewTextureManager(t *testing.T) {
	tm := NewTextureManager()
	if tm == nil {
		t.Fatal("expected non-nil TextureManager")
	}
	if tm.IsLoaded() {
		t.Error("should not be loaded initially")
	}
}

func TestTextureManager_GetCircleImage_NoTexture(t *testing.T) {
	tm := NewTextureManager()
	img := tm.GetCircleImage("nonexistent", 20)
	if img != nil {
		t.Error("expected nil for missing texture")
	}
}

func TestTextureManager_GetCircleImage_MinDiameter(t *testing.T) {
	tm := NewTextureManager()
	img := tm.GetCircleImage("test", 0)
	if img != nil {
		t.Error("expected nil for missing texture even with small diameter")
	}
}
