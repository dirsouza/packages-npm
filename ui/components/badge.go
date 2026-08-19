package components

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/dirsouza/packages-npm/ui/uitheme"
)

// Badge is a small pill-shaped label with a colored background.
// It communicates contextual information (e.g. version, status).
type Badge struct {
	widget.BaseWidget
	text      string
	bg        *canvas.Rectangle
	textObj   *canvas.Text
	container *fyne.Container
}

// NewBadge creates a Badge with the given text and background color.
func NewBadge(text string, bg color.Color) *Badge {
	b := &Badge{text: text}
	b.bg = canvas.NewRectangle(bg)
	b.bg.CornerRadius = 4

	b.textObj = canvas.NewText(text, uitheme.ColorForeground)
	b.textObj.TextSize = 11
	b.textObj.TextStyle = fyne.TextStyle{Bold: true}
	b.textObj.Alignment = fyne.TextAlignCenter

	b.container = container.NewStack(b.bg, container.NewPadded(b.textObj))
	b.ExtendBaseWidget(b)
	return b
}

// SetText updates the badge label.
func (b *Badge) SetText(text string) {
	b.text = text
	b.textObj.Text = text
	b.textObj.Refresh()
}

// SetColor updates the badge background color.
func (b *Badge) SetColor(c color.Color) {
	b.bg.FillColor = c
	b.bg.Refresh()
}

// CreateRenderer delegates to the inner container.
func (b *Badge) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.container)
}
