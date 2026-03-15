package render

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"solar-system-sim/internal/assets"
)

// TextureManager loads and caches planet textures as circular images.
type TextureManager struct {
	mu       sync.RWMutex
	textures map[string]image.Image         // planet name -> raw texture
	circles  map[string]map[int]*image.RGBA // planet name -> size -> circular cutout
	loaded   bool
}

// NewTextureManager creates a new TextureManager.
func NewTextureManager() *TextureManager {
	return &TextureManager{
		textures: make(map[string]image.Image),
		circles:  make(map[string]map[int]*image.RGBA),
	}
}

// LoadAll loads planet textures from the assets directory.
func (tm *TextureManager) LoadAll() error {
	texDir, err := assets.ResolveTextureDir()
	if err != nil {
		return fmt.Errorf("texture dir: %w", err)
	}

	planetNames := []string{"sun", "mercury", "venus", "earth", "mars", "jupiter", "saturn", "uranus", "neptune"}
	loaded := 0

	for _, name := range planetNames {
		// Try common extensions
		for _, ext := range []string{"albedo.jpg", "albedo.png"} {
			path := filepath.Join(texDir, name, ext)
			img, err := loadImage(path)
			if err != nil {
				continue
			}
			tm.mu.Lock()
			tm.textures[name] = img
			tm.circles[name] = make(map[int]*image.RGBA)
			tm.mu.Unlock()
			loaded++
			break
		}
	}

	tm.mu.Lock()
	tm.loaded = true
	tm.mu.Unlock()

	if loaded == 0 {
		return fmt.Errorf("no textures loaded from %s", texDir)
	}
	return nil
}

// IsLoaded returns true if textures have been loaded.
func (tm *TextureManager) IsLoaded() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.loaded
}

// GetCircleImage returns a circular-masked texture for the named planet at the
// given diameter in pixels. Returns nil if no texture is loaded for this planet.
func (tm *TextureManager) GetCircleImage(name string, diameter int) image.Image {
	if diameter < 2 {
		diameter = 2
	}

	lowerName := strings.ToLower(name)

	tm.mu.RLock()
	raw, exists := tm.textures[lowerName]
	if !exists {
		tm.mu.RUnlock()
		return nil
	}
	if cached, ok := tm.circles[lowerName][diameter]; ok {
		tm.mu.RUnlock()
		return cached
	}
	tm.mu.RUnlock()

	// Generate circular cutout
	circle := makeCircularImage(raw, diameter)

	tm.mu.Lock()
	tm.circles[lowerName][diameter] = circle
	tm.mu.Unlock()

	return circle
}

// makeCircularImage resizes and applies a circular mask to an image.
func makeCircularImage(src image.Image, diameter int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, diameter, diameter))
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	radius := float64(diameter) / 2.0
	radiusSq := radius * radius

	for y := 0; y < diameter; y++ {
		for x := 0; x < diameter; x++ {
			// Check if pixel is within circle
			dx := float64(x) - radius + 0.5
			dy := float64(y) - radius + 0.5
			if dx*dx+dy*dy > radiusSq {
				continue
			}

			// Map to source texture coordinates
			srcX := int(float64(x) / float64(diameter) * float64(srcW))
			srcY := int(float64(y) / float64(diameter) * float64(srcH))
			if srcX >= srcW {
				srcX = srcW - 1
			}
			if srcY >= srcH {
				srcY = srcH - 1
			}

			dst.Set(x, y, src.At(srcBounds.Min.X+srcX, srcBounds.Min.Y+srcY))
		}
	}

	return dst
}

// loadImage loads an image from disk.
func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Decode(f)
	case ".png":
		return png.Decode(f)
	default:
		// Try generic decode
		img, _, err := image.Decode(f)
		return img, err
	}
}

// ClearCache clears all cached circular images (e.g., after lighting changes).
func (tm *TextureManager) ClearCache() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	for name := range tm.circles {
		tm.circles[name] = make(map[int]*image.RGBA)
	}
}

// GetRawTexture returns the raw source texture for a planet, or nil.
func (tm *TextureManager) GetRawTexture(name string) image.Image {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.textures[strings.ToLower(name)]
}

// Ensure draw package is importable (used by potential future compositing)
var _ = draw.Over
