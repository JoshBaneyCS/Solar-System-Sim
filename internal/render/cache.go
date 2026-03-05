package render

import (
	"image/color"
	"sync"

	"fyne.io/fyne/v2/canvas"
)

// RenderCache pools canvas objects for performance
type RenderCache struct {
	circles     []*canvas.Circle
	lines       []*canvas.Line
	texts       []*canvas.Text
	circleIndex int
	lineIndex   int
	textIndex   int
	mu          sync.Mutex
}

func NewRenderCache() *RenderCache {
	return &RenderCache{
		circles: make([]*canvas.Circle, 0, 100),
		lines:   make([]*canvas.Line, 0, 5000),
		texts:   make([]*canvas.Text, 0, 50),
	}
}

func (rc *RenderCache) Reset() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.circleIndex = 0
	rc.lineIndex = 0
	rc.textIndex = 0
}

func (rc *RenderCache) GetCircle(col color.Color) *canvas.Circle {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.circleIndex < len(rc.circles) {
		circle := rc.circles[rc.circleIndex]
		circle.FillColor = col
		rc.circleIndex++
		return circle
	}

	circle := canvas.NewCircle(col)
	rc.circles = append(rc.circles, circle)
	rc.circleIndex++
	return circle
}

func (rc *RenderCache) GetLine(col color.Color) *canvas.Line {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.lineIndex < len(rc.lines) {
		line := rc.lines[rc.lineIndex]
		line.StrokeColor = col
		rc.lineIndex++
		return line
	}

	line := canvas.NewLine(col)
	rc.lines = append(rc.lines, line)
	rc.lineIndex++
	return line
}

func (rc *RenderCache) GetText(text string, col color.Color) *canvas.Text {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.textIndex < len(rc.texts) {
		textObj := rc.texts[rc.textIndex]
		textObj.Text = text
		textObj.Color = col
		rc.textIndex++
		return textObj
	}

	textObj := canvas.NewText(text, col)
	rc.texts = append(rc.texts, textObj)
	rc.textIndex++
	return textObj
}
