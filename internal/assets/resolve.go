package assets

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveAssetDir finds the assets directory by searching common locations
// relative to the executable and working directory.
func ResolveAssetDir() (string, error) {
	candidates := []string{}

	// 1. Working directory (development)
	candidates = append(candidates, "assets")

	// 2. Relative to executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "assets"),
			filepath.Join(exeDir, "..", "assets"),
			// macOS .app bundle: Contents/MacOS/../Resources/assets
			filepath.Join(exeDir, "..", "Resources", "assets"),
		)
	}

	for _, candidate := range candidates {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			return abs, nil
		}
	}

	return "", fmt.Errorf("assets directory not found; searched: %v", candidates)
}

// ResolveTextureDir returns the path to the textures subdirectory.
func ResolveTextureDir() (string, error) {
	assetDir, err := ResolveAssetDir()
	if err != nil {
		return "", err
	}
	texDir := filepath.Join(assetDir, "textures")
	if info, err := os.Stat(texDir); err == nil && info.IsDir() {
		return texDir, nil
	}
	return "", fmt.Errorf("textures directory not found at %s", texDir)
}
