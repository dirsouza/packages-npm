package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/dirsouza/packages-npm/ui/uitheme"
)

// NewAppHeader returns a branded header bar with app icon, title and subtitle.
// The background is drawn with canvas.Rectangle to use the surface color.
func NewAppHeader() fyne.CanvasObject {
	bg := canvas.NewRectangle(uitheme.ColorSurface)

	icon := widget.NewIcon(theme.StorageIcon())

	title := canvas.NewText("Gerenciador de Pacotes NPM", uitheme.ColorForeground)
	title.TextSize = 18
	title.TextStyle = fyne.TextStyle{Bold: true}

	subtitle := canvas.NewText("Instale pacotes globais de forma visual e organizada", uitheme.ColorMuted)
	subtitle.TextSize = 12

	textBlock := container.NewVBox(
		container.NewWithoutLayout(title),
		container.NewWithoutLayout(subtitle),
	)

	row := container.NewBorder(nil, nil, container.NewPadded(icon), nil,
		container.NewPadded(textBlock),
	)

	separator := widget.NewSeparator()

	content := container.NewVBox(container.NewPadded(row), separator)

	return container.NewStack(bg, content)
}
