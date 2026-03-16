package render

import (
	"image"
	"image/color"

	"fyne.io/fyne/v2/canvas"
)

// RenderCache pools canvas objects for performance.
// Only accessed from the single render goroutine — no mutex needed.
type RenderCache struct {
	circles     []*canvas.Circle
	lines       []*canvas.Line
	texts       []*canvas.Text
	images      []*canvas.Image
	circleIndex int
	lineIndex   int
	textIndex   int
	imageIndex  int
}

func NewRenderCache() *RenderCache {
	return &RenderCache{
		circles: make([]*canvas.Circle, 0, 100),
		lines:   make([]*canvas.Line, 0, 5000),
		texts:   make([]*canvas.Text, 0, 50),
		images:  make([]*canvas.Image, 0, 20),
	}
}

func (rc *RenderCache) Reset() {
	rc.circleIndex = 0
	rc.lineIndex = 0
	rc.textIndex = 0
	rc.imageIndex = 0
}

func (rc *RenderCache) GetCircle(col color.Color) *canvas.Circle {
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

// GetImage returns a pooled canvas.Image set to the given source image.
func (rc *RenderCache) GetImage(img image.Image) *canvas.Image {
	if rc.imageIndex < len(rc.images) {
		imgObj := rc.images[rc.imageIndex]
		imgObj.Image = img
		imgObj.Translucency = 0
		rc.imageIndex++
		return imgObj
	}

	imgObj := canvas.NewImageFromImage(img)
	imgObj.FillMode = canvas.ImageFillOriginal
	rc.images = append(rc.images, imgObj)
	rc.imageIndex++
	return imgObj
}
