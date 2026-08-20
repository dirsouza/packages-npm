package ui

import (
	"fmt"

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

	// Binários gerados pelo `fyne package` carregam os metadados do
	// FyneApp.toml (ID preenchido); eles têm precedência sobre os valores
	// injetados via -ldflags, usados apenas em builds locais (make build).
	if m := a.Metadata(); m.ID != "" {
		version = m.Version
		build = fmt.Sprintf("%05d", m.Build)
	}

	w := a.NewWindow(appTitle)
	w.Resize(fyne.NewSize(820, 560))
	w.SetFixedSize(false)

	window.NewMainWindow(w, useCase, version, build).Build()

	w.ShowAndRun()
}
