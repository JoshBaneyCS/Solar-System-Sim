package render

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"solar-system-sim/internal/launch"
	"solar-system-sim/internal/math3d"
	"solar-system-sim/internal/physics"
	"solar-system-sim/internal/spacetime"
	"solar-system-sim/internal/viewport"
	"solar-system-sim/pkg/constants"
)

// Renderer handles all rendering operations
type Renderer struct {
	Simulator         *physics.Simulator
	Viewport          *viewport.ViewPort
	Cache             *RenderCache
	SpacetimeRenderer *spacetime.SpacetimeRenderer
	Textures          *TextureManager

	// Distance measurement
	SelectedBodies []*physics.Body

	// Launch trajectory overlay
	LaunchTrajectory *launch.Trajectory
	LaunchEarthPos   math3d.Vec3

	// Launch vehicle position for animated marker
	LaunchVehiclePos *math3d.Vec3

	// Display options
	ShowLabels bool
	ShowBelt   bool

	// Belt renderer
	BeltRenderer *BeltRenderer

	// Trail buffer for image-based trail rendering
	trailBuffer *TrailBuffer

	// Persistent scene container to avoid per-frame allocation
	sceneContainer *fyne.Container

	// Lighting cache
	lightingCache    map[string]*image.RGBA // name+size -> shaded image
	lightingCacheMu  sync.Mutex
	lastSunPos       math3d.Vec3
	sunGlowCache     *image.RGBA
	sunGlowCacheSize int
}

func NewRenderer(sim *physics.Simulator, vp *viewport.ViewPort) *Renderer {
	cache := NewRenderCache()
	r := &Renderer{
		Simulator:         sim,
		Viewport:          vp,
		Cache:             cache,
		SpacetimeRenderer: spacetime.NewSpacetimeRenderer(),
		Textures:          NewTextureManager(),
		SelectedBodies:    make([]*physics.Body, 0, 2),
		ShowLabels:        true,
		ShowBelt:          true,
		BeltRenderer:      NewBeltRenderer(physics.GenerateBeltParticles(1500), cache),
		trailBuffer:       NewTrailBuffer(),
		lightingCache:     make(map[string]*image.RGBA),
	}

	// Load textures asynchronously so startup isn't blocked
	go func() {
		if err := r.Textures.LoadAll(); err != nil {
			log.Printf("Textures: %v (using solid colors)", err)
		}
		r.Textures.LoadSkybox()
	}()

	return r
}

// CreateCanvas renders the simulation state to a canvas using pre-fetched snapshot data.
// This avoids acquiring any simulator locks from the render path.
func (r *Renderer) CreateCanvasFromSnapshot(planets []physics.Body, sun physics.Body, showTrails, showSpacetime bool, simTime float64) *fyne.Container {
	r.Cache.Reset()

	// Single lock acquisition for the entire frame — eliminates 3600+ RLock cycles
	snap := r.Viewport.TakeSnapshot()
	canvasWidth := snap.CanvasWidth
	canvasHeight := snap.CanvasHeight

	objects := []fyne.CanvasObject{}

	if skybox := r.Textures.GetSkybox(); skybox != nil {
		skyboxImg := canvas.NewImageFromImage(skybox)
		skyboxImg.Resize(fyne.NewSize(float32(canvasWidth), float32(canvasHeight)))
		skyboxImg.Move(fyne.NewPos(0, 0))
		skyboxImg.FillMode = canvas.ImageFillStretch
		objects = append(objects, skyboxImg)
	} else {
		bg := canvas.NewRectangle(color.RGBA{5, 5, 15, 255})
		bg.Resize(fyne.NewSize(float32(canvasWidth), float32(canvasHeight)))
		bg.Move(fyne.NewPos(0, 0))
		objects = append(objects, bg)
	}

	if showSpacetime {
		spacetimeObjects := r.SpacetimeRenderer.RenderGrid(r.Cache, r.Viewport, planets, sun)
		objects = append(objects, spacetimeObjects...)
	}

	// Render belt and trail buffers in parallel — they write to independent images
	var beltImg, trailImg *image.RGBA
	var wg sync.WaitGroup

	if r.ShowBelt && r.BeltRenderer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			beltImg = r.BeltRenderer.RenderToImage(snap, simTime, canvasWidth, canvasHeight)
		}()
	}

	if showTrails {
		wg.Add(1)
		go func() {
			defer wg.Done()
			trailImg = r.trailBuffer.Render(planets, snap, canvasWidth, canvasHeight)
		}()
	}

	wg.Wait()

	if beltImg != nil {
		imgObj := r.Cache.GetImage(beltImg)
		imgObj.FillMode = canvas.ImageFillOriginal
		imgObj.Resize(fyne.NewSize(float32(canvasWidth), float32(canvasHeight)))
		imgObj.Move(fyne.NewPos(0, 0))
		objects = append(objects, imgObj)
	}
	if trailImg != nil {
		imgObj := r.Cache.GetImage(trailImg)
		imgObj.FillMode = canvas.ImageFillOriginal
		imgObj.Resize(fyne.NewSize(float32(canvasWidth), float32(canvasHeight)))
		imgObj.Move(fyne.NewPos(0, 0))
		objects = append(objects, imgObj)
	}

	// Create lighting model
	lighting := NewLightingModel(sun.Position)

	// Invalidate shading cache if sun moved significantly
	r.lightingCacheMu.Lock()
	sunDist := sun.Position.Sub(r.lastSunPos).Magnitude()
	if sunDist > 1e9 { // ~0.007 AU
		r.lightingCache = make(map[string]*image.RGBA)
		r.lastSunPos = sun.Position
	}
	r.lightingCacheMu.Unlock()

	// Render Sun
	sunX, sunY := snap.WorldToScreen(sun.Position)
	if r.isOnScreen(sunX, sunY, canvasWidth, canvasHeight) {
		sunRadius := float32(sun.Radius)

		// Sun glow
		glowDiam := int(sunRadius * 4)
		if glowDiam > 4 {
			glowImg := r.getSunGlow(glowDiam)
			if glowImg != nil {
				glow := r.Cache.GetImage(glowImg)
				glow.Resize(fyne.NewSize(float32(glowDiam), float32(glowDiam)))
				glow.Move(fyne.NewPos(sunX-float32(glowDiam)/2, sunY-float32(glowDiam)/2))
				objects = append(objects, glow)
			}
		}

		// Sun body — use texture if available
		sunDiam := int(sunRadius * 2)
		sunImg := r.Textures.GetCircleImage("sun", sunDiam)
		if sunImg != nil {
			sunObj := r.Cache.GetImage(sunImg)
			sunObj.Resize(fyne.NewSize(sunRadius*2, sunRadius*2))
			sunObj.Move(fyne.NewPos(sunX-sunRadius, sunY-sunRadius))
			objects = append(objects, sunObj)
		} else {
			sunCircle := r.Cache.GetCircle(sun.Color)
			sunCircle.Resize(fyne.NewSize(sunRadius*2, sunRadius*2))
			sunCircle.Move(fyne.NewPos(sunX-sunRadius, sunY-sunRadius))
			objects = append(objects, sunCircle)
		}

		if r.ShowLabels {
			sunLabel := r.Cache.GetText("Sun", color.White)
			sunLabel.TextSize = 10
			sunLabel.Move(fyne.NewPos(sunX+sunRadius+5, sunY-5))
			objects = append(objects, sunLabel)
		}
	}

	// Render planets, moons, comets, and asteroids
	displayScale := snap.DisplayScale
	for _, planet := range planets {
		x, y := snap.WorldToScreen(planet.Position)

		if r.isOnScreen(x, y, canvasWidth, canvasHeight) {
			// Compute display radius: use physical radius when zoomed in enough
			planetRadius := float32(planet.Radius)
			if planet.PhysicalRadius > 0 {
				physPx := float32(planet.PhysicalRadius / constants.AU * displayScale)
				if physPx > planetRadius {
					planetRadius = physPx
				}
			}
			if planetRadius > 5000 {
				planetRadius = 5000
			}
			diameter := int(planetRadius * 2)

			rendered := false

			// Asteroids: use irregular shape
			if planet.Type == physics.BodyTypeAsteroid && diameter >= 4 {
				seed := int64(len(planet.Name)) * 7919
				irregImg := r.Textures.GetIrregularImage(planet.Name, diameter, seed)
				if irregImg != nil {
					imgObj := r.Cache.GetImage(irregImg)
					imgObj.Resize(fyne.NewSize(planetRadius*2, planetRadius*2))
					imgObj.Move(fyne.NewPos(x-planetRadius, y-planetRadius))
					objects = append(objects, imgObj)
					rendered = true
				}
			}

			// Regular textured rendering for planets, moons, dwarf planets
			if !rendered && r.Textures.IsLoaded() && diameter >= 4 {
				shadedImg := r.getShadedPlanetImage(planet.Name, diameter, lighting, planet.Position)
				if shadedImg != nil {
					imgObj := r.Cache.GetImage(shadedImg)
					imgObj.Resize(fyne.NewSize(planetRadius*2, planetRadius*2))
					imgObj.Move(fyne.NewPos(x-planetRadius, y-planetRadius))
					objects = append(objects, imgObj)
					rendered = true
				}
			}

			if !rendered {
				circle := r.Cache.GetCircle(planet.Color)
				circle.Resize(fyne.NewSize(planetRadius*2, planetRadius*2))
				circle.Move(fyne.NewPos(x-planetRadius, y-planetRadius))
				objects = append(objects, circle)
			}

			// Comet tail: gradient wedge away from Sun
			if planet.Type == physics.BodyTypeComet && planetRadius >= 2 {
				objects = append(objects, r.renderCometTail(planet, sun, x, y, canvasWidth, canvasHeight)...)
			}

			if r.ShowLabels {
				label := r.Cache.GetText(planet.Name, color.White)
				label.TextSize = 10
				label.Move(fyne.NewPos(x+planetRadius+3, y-5))
				objects = append(objects, label)
			}
		}
	}

	// Render launch trajectory
	if r.LaunchTrajectory != nil && len(r.LaunchTrajectory.Points) > 1 {
		objects = append(objects, r.renderTrajectory(&snap, canvasWidth, canvasHeight)...)
	}

	// Render launch vehicle marker
	if r.LaunchVehiclePos != nil {
		vx, vy := snap.WorldToScreen(*r.LaunchVehiclePos)
		if r.isOnScreen(vx, vy, canvasWidth, canvasHeight) {
			marker := r.Cache.GetCircle(color.RGBA{0, 255, 200, 255})
			marker.Resize(fyne.NewSize(8, 8))
			marker.Move(fyne.NewPos(vx-4, vy-4))
			objects = append(objects, marker)
		}
	}

	if len(r.SelectedBodies) == 2 {
		pos1 := r.SelectedBodies[0].Position
		pos2 := r.SelectedBodies[1].Position

		x1, y1 := snap.WorldToScreen(pos1)
		x2, y2 := snap.WorldToScreen(pos2)

		distLine := r.Cache.GetLine(color.RGBA{255, 255, 0, 180})
		distLine.Position1 = fyne.NewPos(x1, y1)
		distLine.Position2 = fyne.NewPos(x2, y2)
		distLine.StrokeWidth = 2
		objects = append(objects, distLine)

		dist := pos2.Sub(pos1).Magnitude()
		distAU := dist / constants.AU
		distKm := dist / 1000
		lightMinutes := dist / (299792458.0 * 60)

		midX := (x1 + x2) / 2
		midY := (y1 + y2) / 2

		distText := fmt.Sprintf("%.3f AU\n%.2e km\n%.2f light-min", distAU, distKm, lightMinutes)
		distLabel := r.Cache.GetText(distText, color.RGBA{255, 255, 0, 255})
		distLabel.TextSize = 12
		distLabel.Alignment = fyne.TextAlignCenter
		distLabel.Move(fyne.NewPos(midX-50, midY-30))
		objects = append(objects, distLabel)
	}

	if r.sceneContainer == nil {
		r.sceneContainer = container.NewWithoutLayout(objects...)
	} else {
		r.sceneContainer.Objects = objects
	}
	return r.sceneContainer
}

// getShadedPlanetImage returns a cached or freshly-shaded planet texture.
func (r *Renderer) getShadedPlanetImage(name string, diameter int, lighting *LightingModel, planetPos math3d.Vec3) image.Image {
	key := fmt.Sprintf("%s_%d", strings.ToLower(name), diameter)

	r.lightingCacheMu.Lock()
	if cached, ok := r.lightingCache[key]; ok {
		r.lightingCacheMu.Unlock()
		return cached
	}
	r.lightingCacheMu.Unlock()

	baseImg := r.Textures.GetCircleImage(name, diameter)
	if baseImg == nil {
		return nil
	}

	shaded := lighting.ApplyDiffuseShading(baseImg, planetPos)

	r.lightingCacheMu.Lock()
	r.lightingCache[key] = shaded
	r.lightingCacheMu.Unlock()

	return shaded
}

// getSunGlow returns a cached or freshly-generated sun glow image.
func (r *Renderer) getSunGlow(diameter int) image.Image {
	if r.sunGlowCache != nil && r.sunGlowCacheSize == diameter {
		return r.sunGlowCache
	}
	r.sunGlowCache = SunGlowImage(diameter)
	r.sunGlowCacheSize = diameter
	return r.sunGlowCache
}

// renderTrajectory draws the launch trajectory as colored line segments.
func (r *Renderer) renderTrajectory(snap *viewport.Snapshot, canvasWidth, canvasHeight float64) []fyne.CanvasObject {
	traj := r.LaunchTrajectory
	if traj == nil || len(traj.Points) < 2 {
		return nil
	}

	var objects []fyne.CanvasObject

	points := traj.Points
	isEarthCentered := traj.Frame == launch.EarthCentered

	step := 1
	if len(points) > 500 {
		step = len(points) / 500
	}

	trajectoryColor := color.RGBA{0, 255, 128, 200}

	for j := 0; j < len(points)-step; j += step {
		p1 := points[j].Position
		p2 := points[j+step].Position

		if isEarthCentered {
			p1 = p1.Add(r.LaunchEarthPos)
			p2 = p2.Add(r.LaunchEarthPos)
		}

		x1, y1 := snap.WorldToScreen(p1)
		x2, y2 := snap.WorldToScreen(p2)

		if r.isOnScreen(x1, y1, canvasWidth, canvasHeight) ||
			r.isOnScreen(x2, y2, canvasWidth, canvasHeight) {
			progress := float64(j) / float64(len(points))
			c := color.RGBA{
				R: uint8(progress * 255),
				G: uint8((1 - progress*0.5) * 255),
				B: uint8((1 - progress) * 128),
				A: trajectoryColor.A,
			}

			line := r.Cache.GetLine(c)
			line.Position1 = fyne.NewPos(x1, y1)
			line.Position2 = fyne.NewPos(x2, y2)
			line.StrokeWidth = 2
			objects = append(objects, line)
		}
	}

	return objects
}

// CreateLabelOverlay returns only text labels as a container (for GPU render mode overlay).
func (r *Renderer) CreateLabelOverlay() *fyne.Container {
	r.Cache.Reset()

	planets := r.Simulator.GetPlanetSnapshot()
	sun := r.Simulator.GetSunSnapshot()

	snap := r.Viewport.TakeSnapshot()
	canvasWidth := snap.CanvasWidth
	canvasHeight := snap.CanvasHeight

	objects := []fyne.CanvasObject{}

	if r.ShowLabels {
		sunX, sunY := snap.WorldToScreen(sun.Position)
		if r.isOnScreen(sunX, sunY, canvasWidth, canvasHeight) {
			sunRadius := float32(sun.Radius)
			sunLabel := r.Cache.GetText("Sun", color.White)
			sunLabel.TextSize = 10
			sunLabel.Move(fyne.NewPos(sunX+sunRadius+5, sunY-5))
			objects = append(objects, sunLabel)
		}

		for _, planet := range planets {
			x, y := snap.WorldToScreen(planet.Position)
			if r.isOnScreen(x, y, canvasWidth, canvasHeight) {
				planetRadius := float32(planet.Radius)
				label := r.Cache.GetText(planet.Name, color.White)
				label.TextSize = 10
				label.Move(fyne.NewPos(x+planetRadius+3, y-5))
				objects = append(objects, label)
			}
		}
	}

	if len(r.SelectedBodies) == 2 {
		pos1 := r.SelectedBodies[0].Position
		pos2 := r.SelectedBodies[1].Position

		x1, y1 := snap.WorldToScreen(pos1)
		x2, y2 := snap.WorldToScreen(pos2)

		dist := pos2.Sub(pos1).Magnitude()
		distAU := dist / constants.AU
		distKm := dist / 1000
		lightMinutes := dist / (299792458.0 * 60)

		midX := (x1 + x2) / 2
		midY := (y1 + y2) / 2

		distText := fmt.Sprintf("%.3f AU\n%.2e km\n%.2f light-min", distAU, distKm, lightMinutes)
		distLabel := r.Cache.GetText(distText, color.RGBA{255, 255, 0, 255})
		distLabel.TextSize = 12
		distLabel.Alignment = fyne.TextAlignCenter
		distLabel.Move(fyne.NewPos(midX-50, midY-30))
		objects = append(objects, distLabel)
	}

	return container.NewWithoutLayout(objects...)
}

// renderCometTail draws a gradient tail pointing away from the Sun.
func (r *Renderer) renderCometTail(comet physics.Body, sun physics.Body, cx, cy float32, canvasWidth, canvasHeight float64) []fyne.CanvasObject {
	// Direction away from Sun
	dx := comet.Position.X - sun.Position.X
	dy := comet.Position.Y - sun.Position.Y
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 1e6 {
		return nil
	}
	nx := dx / dist
	ny := dy / dist

	// Tail length in pixels (scaled by distance from Sun — closer = longer tail)
	distAU := dist / constants.AU
	tailLenPx := float32(80.0 / (distAU + 0.5))
	if tailLenPx < 10 {
		tailLenPx = 10
	}
	if tailLenPx > 200 {
		tailLenPx = 200
	}

	var objects []fyne.CanvasObject
	segments := 8
	for i := 0; i < segments; i++ {
		t0 := float32(i) / float32(segments)
		t1 := float32(i+1) / float32(segments)

		x1 := cx + float32(nx)*tailLenPx*t0
		y1 := cy + float32(ny)*tailLenPx*t0
		x2 := cx + float32(nx)*tailLenPx*t1
		y2 := cy + float32(ny)*tailLenPx*t1

		alpha := uint8(float32(180) * (1 - t0))
		tailColor := color.RGBA{180, 210, 255, alpha}

		line := r.Cache.GetLine(tailColor)
		line.Position1 = fyne.NewPos(x1, y1)
		line.Position2 = fyne.NewPos(x2, y2)
		line.StrokeWidth = float32(3) * (1 - t0)
		if line.StrokeWidth < 1 {
			line.StrokeWidth = 1
		}
		objects = append(objects, line)
	}
	return objects
}

func (r *Renderer) isOnScreen(x, y float32, width, height float64) bool {
	margin := float32(100)
	return x >= -margin && x <= float32(width)+margin &&
		y >= -margin && y <= float32(height)+margin
}

// Ensure math is imported (used by lighting angle threshold)
var _ = math.Pi
