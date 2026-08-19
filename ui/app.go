package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/dirsouza/packages-npm/internal/core/ports/inbound"
	"github.com/dirsouza/packages-npm/ui/uitheme"
	"github.com/dirsouza/packages-npm/ui/window"
)

const (
	appID    = "io.github.packages-npm"
	appTitle = "Gerenciador de Pacotes NPM"
)

// Run is the composition root of the UI layer.
// It applies the custom theme, wires the Fyne application and main window,
// then starts the event loop.
func Run(useCase inbound.PackageUseCase, version, build string) {
	a := app.NewWithID(appID)
	a.Settings().SetTheme(&uitheme.DarkTheme{})

	w := a.NewWindow(appTitle)
	w.Resize(fyne.NewSize(820, 560))
	w.SetFixedSize(false)

	window.NewMainWindow(w, useCase, version, build).Build()

	w.ShowAndRun()
}
