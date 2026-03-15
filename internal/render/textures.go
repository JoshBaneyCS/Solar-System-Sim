package render

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
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
	skybox   image.Image
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

	// Dynamically discover all texture directories
	entries, dirErr := os.ReadDir(texDir)
	if dirErr != nil {
		return fmt.Errorf("read texture dir: %w", dirErr)
	}

	loaded := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		for _, ext := range []string{"albedo.jpg", "albedo.png"} {
			path := filepath.Join(texDir, name, ext)
			img, loadErr := loadImage(path)
			if loadErr != nil {
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

// LoadSkybox loads the skybox/milky_way.jpg texture for the background.
func (tm *TextureManager) LoadSkybox() {
	texDir, err := assets.ResolveTextureDir()
	if err != nil {
		return
	}
	for _, name := range []string{"milky_way.jpg", "milky_way.png"} {
		img, err := loadImage(filepath.Join(texDir, "skybox", name))
		if err == nil {
			tm.mu.Lock()
			tm.skybox = img
			tm.mu.Unlock()
			return
		}
	}
}

// GetSkybox returns the loaded skybox image, or nil.
func (tm *TextureManager) GetSkybox() image.Image {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.skybox
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

// GetIrregularImage returns an irregular (non-circular) image for an asteroid.
// Uses a seeded noise function to perturb the boundary into a lumpy potato shape.
func (tm *TextureManager) GetIrregularImage(name string, diameter int, seed int64) image.Image {
	if diameter < 2 {
		diameter = 2
	}
	lowerName := strings.ToLower(name)

	// Check cache first (reuse circles cache)
	cacheKey := fmt.Sprintf("%s_irreg_%d", lowerName, seed)
	tm.mu.RLock()
	if sizeMap, ok := tm.circles[cacheKey]; ok {
		if cached, ok := sizeMap[diameter]; ok {
			tm.mu.RUnlock()
			return cached
		}
	}
	tm.mu.RUnlock()

	// Generate procedural irregular asteroid
	img := makeIrregularImage(diameter, seed)

	tm.mu.Lock()
	if _, ok := tm.circles[cacheKey]; !ok {
		tm.circles[cacheKey] = make(map[int]*image.RGBA)
	}
	tm.circles[cacheKey][diameter] = img
	tm.mu.Unlock()

	return img
}

// makeIrregularImage creates a lumpy potato-shaped asteroid image.
func makeIrregularImage(diameter int, seed int64) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, diameter, diameter))
	radius := float64(diameter) / 2.0
	rng := newSimpleRNG(seed)

	// Generate 8 radial perturbation values for lumpy shape
	const nLobes = 8
	lobes := make([]float64, nLobes)
	for i := range lobes {
		lobes[i] = 0.65 + rng.Float64()*0.35 // 65% to 100% of radius
	}

	// Base gray color with slight variation
	baseR := uint8(100 + rng.Intn(80))
	baseG := uint8(90 + rng.Intn(70))
	baseB := uint8(80 + rng.Intn(60))

	for y := 0; y < diameter; y++ {
		for x := 0; x < diameter; x++ {
			dx := float64(x) - radius + 0.5
			dy := float64(y) - radius + 0.5
			dist := math.Sqrt(dx*dx + dy*dy)
			angle := math.Atan2(dy, dx)

			// Interpolate between lobes for smooth boundary
			t := (angle + math.Pi) / (2 * math.Pi) * float64(nLobes)
			idx := int(t) % nLobes
			nextIdx := (idx + 1) % nLobes
			frac := t - math.Floor(t)
			boundaryRadius := radius * (lobes[idx]*(1-frac) + lobes[nextIdx]*frac)

			if dist > boundaryRadius {
				continue
			}

			// Surface detail: slight color variation
			shade := 0.7 + 0.3*(1-dist/boundaryRadius)
			r := uint8(float64(baseR) * shade)
			g := uint8(float64(baseG) * shade)
			b := uint8(float64(baseB) * shade)
			dst.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	return dst
}

// simpleRNG is a simple deterministic random number generator.
type simpleRNG struct {
	state uint64
}

func newSimpleRNG(seed int64) *simpleRNG {
	return &simpleRNG{state: uint64(seed) ^ 0x5DEECE66D}
}

func (r *simpleRNG) next() uint64 {
	r.state = r.state*6364136223846793005 + 1442695040888963407
	return r.state
}

func (r *simpleRNG) Float64() float64 {
	return float64(r.next()>>11) / float64(1<<53)
}

func (r *simpleRNG) Intn(n int) int {
	return int(r.next() % uint64(n))
}

// Ensure draw package is importable (used by potential future compositing)
var _ = draw.Over
