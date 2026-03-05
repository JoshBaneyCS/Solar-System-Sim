package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
)

var bodies = []string{
	"mercury", "venus", "earth", "mars",
	"jupiter", "saturn", "uranus", "neptune", "sun",
}

func main() {
	dir := flag.String("dir", "assets", "Asset directory path")
	flag.Parse()

	var errors []string

	// Check texture directory structure
	texDir := filepath.Join(*dir, "textures")
	for _, body := range bodies {
		albedoJPG := filepath.Join(texDir, body, "albedo.jpg")
		albedoPNG := filepath.Join(texDir, body, "albedo.png")

		found := false
		for _, path := range []string{albedoJPG, albedoPNG} {
			if _, err := os.Stat(path); err == nil {
				found = true
				if errs := validateImage(path); len(errs) > 0 {
					errors = append(errors, errs...)
				}
			}
		}
		if !found {
			errors = append(errors, fmt.Sprintf("missing texture: %s/albedo.{jpg,png}", filepath.Join(texDir, body)))
		}
	}

	// Check models directory
	modelsDir := filepath.Join(*dir, "models")
	earthGLB := filepath.Join(modelsDir, "earth.glb")
	if _, err := os.Stat(earthGLB); err != nil {
		errors = append(errors, fmt.Sprintf("missing model: %s", earthGLB))
	} else {
		if errs := validateGLB(earthGLB); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

	// Check CREDITS.md
	credits := filepath.Join(*dir, "CREDITS.md")
	if _, err := os.Stat(credits); err != nil {
		errors = append(errors, fmt.Sprintf("missing: %s", credits))
	}

	// Check meshes (optional)
	meshDir := filepath.Join(*dir, "meshes")
	for _, name := range []string{"sphere_32.glb", "sphere_64.glb"} {
		path := filepath.Join(meshDir, name)
		if _, err := os.Stat(path); err != nil {
			fmt.Printf("INFO: optional mesh not found: %s (run 'make meshgen' to generate)\n", path)
		} else {
			if errs := validateGLB(path); len(errs) > 0 {
				errors = append(errors, errs...)
			}
		}
	}

	if len(errors) > 0 {
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", e)
		}
		os.Exit(1)
	}

	fmt.Printf("All assets validated successfully (%d body textures, models, credits)\n", len(bodies))
}

func validateImage(path string) []string {
	var errors []string

	f, err := os.Open(path)
	if err != nil {
		return []string{fmt.Sprintf("cannot open %s: %v", path, err)}
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return []string{fmt.Sprintf("cannot decode %s: %v", path, err)}
	}

	if cfg.Width < 512 {
		errors = append(errors, fmt.Sprintf("%s: width %d < 512 (too small)", path, cfg.Width))
	}
	if cfg.Width > 16384 {
		errors = append(errors, fmt.Sprintf("%s: width %d > 16384 (too large)", path, cfg.Width))
	}
	if cfg.Height < 256 {
		errors = append(errors, fmt.Sprintf("%s: height %d < 256 (too small)", path, cfg.Height))
	}

	return errors
}

func validateGLB(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return []string{fmt.Sprintf("cannot open %s: %v", path, err)}
	}
	defer f.Close()

	header := make([]byte, 4)
	if _, err := f.Read(header); err != nil {
		return []string{fmt.Sprintf("cannot read %s: %v", path, err)}
	}

	if string(header) != "glTF" {
		return []string{fmt.Sprintf("%s: invalid GLB header (expected 'glTF', got '%s')", path, string(header))}
	}

	return nil
}
