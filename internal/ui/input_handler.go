package ui

import (
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"solar-system-sim/internal/viewport"
)

// InteractiveCanvas wraps the simulation canvas with mouse and keyboard input handling.
type InteractiveCanvas struct {
	widget.BaseWidget

	content  fyne.CanvasObject
	viewport *viewport.ViewPort

	// Drag state
	dragging     bool
	lastDragPos  fyne.Position
	shiftPressed bool

	// Focus state
	focused bool

	// Sensitivity settings
	OrbitSensitivity float64
	PanSensitivity   float64
	ZoomFactor       float64
	KeyMoveSpeed     float64
}

// NewInteractiveCanvas creates an interactive canvas with camera controls.
func NewInteractiveCanvas(content fyne.CanvasObject, vp *viewport.ViewPort) *InteractiveCanvas {
	ic := &InteractiveCanvas{
		content:          content,
		viewport:         vp,
		OrbitSensitivity: 0.01,
		PanSensitivity:   0.002,
		ZoomFactor:       1.15,
		KeyMoveSpeed:     0.1,
	}
	ic.ExtendBaseWidget(ic)
	return ic
}

// CreateRenderer implements fyne.Widget.
func (ic *InteractiveCanvas) CreateRenderer() fyne.WidgetRenderer {
	return &interactiveCanvasRenderer{canvas: ic}
}

// --- Scrollable (zoom) ---

func (ic *InteractiveCanvas) Scrolled(ev *fyne.ScrollEvent) {
	delta := float64(ev.Scrolled.DY) / 10.0
	factor := math.Pow(ic.ZoomFactor, delta)
	ic.viewport.AdjustZoom(factor)
}

// --- Draggable (orbit / pan) ---

func (ic *InteractiveCanvas) Dragged(ev *fyne.DragEvent) {
	dx := float64(ev.Dragged.DX)
	dy := float64(ev.Dragged.DY)

	if ic.shiftPressed {
		// Shift+drag: pan
		ic.viewport.RLock()
		zoom := ic.viewport.Zoom
		ic.viewport.RUnlock()
		panScale := ic.PanSensitivity / zoom
		ic.viewport.AdjustPan(-dx*panScale, -dy*panScale)
	} else {
		// Normal drag: orbit
		ic.viewport.Lock()
		ic.viewport.Use3D = true
		ic.viewport.RotationY += dx * ic.OrbitSensitivity
		ic.viewport.RotationX += dy * ic.OrbitSensitivity
		// Clamp pitch to avoid flipping
		if ic.viewport.RotationX > math.Pi/2 {
			ic.viewport.RotationX = math.Pi / 2
		}
		if ic.viewport.RotationX < -math.Pi/2 {
			ic.viewport.RotationX = -math.Pi / 2
		}
		ic.viewport.Unlock()
	}
}

func (ic *InteractiveCanvas) DragEnd() {
	ic.dragging = false
}

// --- Focusable (keyboard) ---

func (ic *InteractiveCanvas) FocusGained() {
	ic.focused = true
}

func (ic *InteractiveCanvas) FocusLost() {
	ic.focused = false
}

func (ic *InteractiveCanvas) TypedRune(r rune) {
	// Not used — we handle key names
}

func (ic *InteractiveCanvas) TypedKey(ev *fyne.KeyEvent) {
	ic.viewport.RLock()
	zoom := ic.viewport.Zoom
	ic.viewport.RUnlock()

	moveStep := ic.KeyMoveSpeed / zoom

	switch ev.Name {
	case fyne.KeyW:
		ic.viewport.AdjustPan(0, -moveStep)
	case fyne.KeyS:
		ic.viewport.AdjustPan(0, moveStep)
	case fyne.KeyA:
		ic.viewport.AdjustPan(-moveStep, 0)
	case fyne.KeyD:
		ic.viewport.AdjustPan(moveStep, 0)
	case fyne.KeyR:
		ic.viewport.AdjustZoom(ic.ZoomFactor)
	case fyne.KeyF:
		ic.viewport.AdjustZoom(1.0 / ic.ZoomFactor)
	case fyne.KeyQ:
		ic.viewport.Lock()
		ic.viewport.RotationZ -= 0.05
		ic.viewport.Unlock()
	case fyne.KeyE:
		ic.viewport.Lock()
		ic.viewport.RotationZ += 0.05
		ic.viewport.Unlock()
	}
}

// --- Desktop mouse events for shift detection ---

func (ic *InteractiveCanvas) MouseDown(ev *desktop.MouseEvent) {
	// Request focus on click
	if c := fyne.CurrentApp().Driver().CanvasForObject(ic); c != nil {
		c.Focus(ic)
	}
	ic.shiftPressed = ev.Modifier&fyne.KeyModifierShift != 0
}

func (ic *InteractiveCanvas) MouseUp(ev *desktop.MouseEvent) {
	ic.shiftPressed = false
}

// --- Tapped (request focus) ---

func (ic *InteractiveCanvas) Tapped(ev *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(ic); c != nil {
		c.Focus(ic)
	}
}

func (ic *InteractiveCanvas) TappedSecondary(ev *fyne.PointEvent) {}

// --- Renderer ---

type interactiveCanvasRenderer struct {
	canvas *InteractiveCanvas
}

func (r *interactiveCanvasRenderer) Layout(size fyne.Size) {
	r.canvas.content.Resize(size)
	r.canvas.content.Move(fyne.NewPos(0, 0))
}

func (r *interactiveCanvasRenderer) MinSize() fyne.Size {
	return fyne.NewSize(400, 300)
}

func (r *interactiveCanvasRenderer) Refresh() {
	r.canvas.content.Refresh()
}

func (r *interactiveCanvasRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.canvas.content}
}

func (r *interactiveCanvasRenderer) Destroy() {}
