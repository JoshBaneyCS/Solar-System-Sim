package render

import (
	"fmt"
	"image/color"

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

	// Distance measurement
	SelectedBodies []*physics.Body

	// Launch trajectory overlay
	LaunchTrajectory *launch.Trajectory
	LaunchEarthPos   math3d.Vec3
}

func NewRenderer(sim *physics.Simulator, vp *viewport.ViewPort) *Renderer {
	return &Renderer{
		Simulator:         sim,
		Viewport:          vp,
		Cache:             NewRenderCache(),
		SpacetimeRenderer: spacetime.NewSpacetimeRenderer(),
		SelectedBodies:    make([]*physics.Body, 0, 2),
	}
}

// CreateCanvas renders the simulation state to a canvas
func (r *Renderer) CreateCanvas() *fyne.Container {
	r.Cache.Reset()

	planets := r.Simulator.GetPlanetSnapshot()
	sun := r.Simulator.GetSunSnapshot()

	r.Simulator.RLock()
	showTrails := r.Simulator.ShowTrails
	showSpacetime := r.Simulator.ShowSpacetime
	r.Simulator.RUnlock()

	r.Viewport.RLock()
	canvasWidth := r.Viewport.CanvasWidth
	canvasHeight := r.Viewport.CanvasHeight
	r.Viewport.RUnlock()

	objects := []fyne.CanvasObject{}

	bg := canvas.NewRectangle(color.RGBA{5, 5, 15, 255})
	bg.Resize(fyne.NewSize(float32(canvasWidth), float32(canvasHeight)))
	bg.Move(fyne.NewPos(0, 0))
	objects = append(objects, bg)

	if showSpacetime {
		spacetimeObjects := r.SpacetimeRenderer.RenderGrid(r.Cache, r.Viewport, planets, sun)
		objects = append(objects, spacetimeObjects...)
	}

	if showTrails {
		for _, planet := range planets {
			if len(planet.Trail) > 1 {
				step := 1
				if len(planet.Trail) > 200 {
					step = len(planet.Trail) / 200
				}

				for j := 0; j < len(planet.Trail)-step; j += step {
					// Gather 4 control points for Catmull-Rom (clamped at edges)
					i0 := j - step
					if i0 < 0 {
						i0 = 0
					}
					i1 := j
					i2 := j + step
					i3 := j + 2*step
					if i3 >= len(planet.Trail) {
						i3 = len(planet.Trail) - 1
					}
					p0 := planet.Trail[i0]
					p1 := planet.Trail[i1]
					p2 := planet.Trail[i2]
					p3 := planet.Trail[i3]

					alpha := uint8(float64(j) / float64(len(planet.Trail)) * 255)
					lineColor := planet.Color
					if c, ok := planet.Color.(color.RGBA); ok {
						lineColor = color.RGBA{c.R, c.G, c.B, alpha}
					}

					// Subdivide segment into 4 interpolated sub-segments
					const nSub = 4
					prevX, prevY := r.Viewport.WorldToScreen(p1)
					for k := 1; k <= nSub; k++ {
						t := float64(k) / float64(nSub)
						pt := math3d.CatmullRom(p0, p1, p2, p3, t)
						curX, curY := r.Viewport.WorldToScreen(pt)

						if r.isOnScreen(prevX, prevY, canvasWidth, canvasHeight) ||
							r.isOnScreen(curX, curY, canvasWidth, canvasHeight) {
							line := r.Cache.GetLine(lineColor)
							line.Position1 = fyne.NewPos(prevX, prevY)
							line.Position2 = fyne.NewPos(curX, curY)
							line.StrokeWidth = 1
							objects = append(objects, line)
						}
						prevX, prevY = curX, curY
					}
				}
			}
		}
	}

	sunX, sunY := r.Viewport.WorldToScreen(sun.Position)
	if r.isOnScreen(sunX, sunY, canvasWidth, canvasHeight) {
		sunRadius := float32(sun.Radius)
		sunCircle := r.Cache.GetCircle(sun.Color)
		sunCircle.Resize(fyne.NewSize(sunRadius*2, sunRadius*2))
		sunCircle.Move(fyne.NewPos(sunX-sunRadius, sunY-sunRadius))
		objects = append(objects, sunCircle)

		sunLabel := r.Cache.GetText("Sun", color.White)
		sunLabel.TextSize = 10
		sunLabel.Move(fyne.NewPos(sunX+sunRadius+5, sunY-5))
		objects = append(objects, sunLabel)
	}

	for _, planet := range planets {
		x, y := r.Viewport.WorldToScreen(planet.Position)

		if r.isOnScreen(x, y, canvasWidth, canvasHeight) {
			planetRadius := float32(planet.Radius)

			circle := r.Cache.GetCircle(planet.Color)
			circle.Resize(fyne.NewSize(planetRadius*2, planetRadius*2))
			circle.Move(fyne.NewPos(x-planetRadius, y-planetRadius))
			objects = append(objects, circle)

			label := r.Cache.GetText(planet.Name, color.White)
			label.TextSize = 10
			label.Move(fyne.NewPos(x+planetRadius+3, y-5))
			objects = append(objects, label)
		}
	}

	// Render launch trajectory
	if r.LaunchTrajectory != nil && len(r.LaunchTrajectory.Points) > 1 {
		objects = append(objects, r.renderTrajectory(canvasWidth, canvasHeight)...)
	}

	if len(r.SelectedBodies) == 2 {
		pos1 := r.SelectedBodies[0].Position
		pos2 := r.SelectedBodies[1].Position

		x1, y1 := r.Viewport.WorldToScreen(pos1)
		x2, y2 := r.Viewport.WorldToScreen(pos2)

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

	return container.NewWithoutLayout(objects...)
}

// renderTrajectory draws the launch trajectory as colored line segments.
func (r *Renderer) renderTrajectory(canvasWidth, canvasHeight float64) []fyne.CanvasObject {
	traj := r.LaunchTrajectory
	if traj == nil || len(traj.Points) < 2 {
		return nil
	}

	var objects []fyne.CanvasObject

	// Convert Earth-centered trajectory to heliocentric for rendering
	points := traj.Points
	isEarthCentered := traj.Frame == launch.EarthCentered

	step := 1
	if len(points) > 500 {
		step = len(points) / 500
	}

	trajectoryColor := color.RGBA{0, 255, 128, 200} // green

	for j := 0; j < len(points)-step; j += step {
		p1 := points[j].Position
		p2 := points[j+step].Position

		if isEarthCentered {
			p1 = p1.Add(r.LaunchEarthPos)
			p2 = p2.Add(r.LaunchEarthPos)
		}

		x1, y1 := r.Viewport.WorldToScreen(p1)
		x2, y2 := r.Viewport.WorldToScreen(p2)

		if r.isOnScreen(x1, y1, canvasWidth, canvasHeight) ||
			r.isOnScreen(x2, y2, canvasWidth, canvasHeight) {
			// Color gradient: green at start -> orange at end
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

	r.Viewport.RLock()
	canvasWidth := r.Viewport.CanvasWidth
	canvasHeight := r.Viewport.CanvasHeight
	r.Viewport.RUnlock()

	objects := []fyne.CanvasObject{}

	sunX, sunY := r.Viewport.WorldToScreen(sun.Position)
	if r.isOnScreen(sunX, sunY, canvasWidth, canvasHeight) {
		sunRadius := float32(sun.Radius)
		sunLabel := r.Cache.GetText("Sun", color.White)
		sunLabel.TextSize = 10
		sunLabel.Move(fyne.NewPos(sunX+sunRadius+5, sunY-5))
		objects = append(objects, sunLabel)
	}

	for _, planet := range planets {
		x, y := r.Viewport.WorldToScreen(planet.Position)
		if r.isOnScreen(x, y, canvasWidth, canvasHeight) {
			planetRadius := float32(planet.Radius)
			label := r.Cache.GetText(planet.Name, color.White)
			label.TextSize = 10
			label.Move(fyne.NewPos(x+planetRadius+3, y-5))
			objects = append(objects, label)
		}
	}

	if len(r.SelectedBodies) == 2 {
		pos1 := r.SelectedBodies[0].Position
		pos2 := r.SelectedBodies[1].Position

		x1, y1 := r.Viewport.WorldToScreen(pos1)
		x2, y2 := r.Viewport.WorldToScreen(pos2)

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

func (r *Renderer) isOnScreen(x, y float32, width, height float64) bool {
	margin := float32(100)
	return x >= -margin && x <= float32(width)+margin &&
		y >= -margin && y <= float32(height)+margin
}
