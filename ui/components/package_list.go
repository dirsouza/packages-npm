package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/dirsouza/packages-npm/ui/uitheme"
	"github.com/dirsouza/packages-npm/ui/viewmodel"
)

// PackageListCallbacks groups all interaction callbacks for a PackageList.
type PackageListCallbacks struct {
	OnEdit   func(vm viewmodel.PackageVM)
	OnDelete func(vm viewmodel.PackageVM)
	OnToggle func(id int64, selected bool)
}

// PackageList is a scrollable list of package cards.
// Uses VBox+Scroll with widget.Card per item for a richer visual hierarchy
// compared to the default widget.List flat rows.
type PackageList struct {
	widget.BaseWidget
	items     []viewmodel.PackageVM
	callbacks PackageListCallbacks
	vbox      *fyne.Container
	scroll    *container.Scroll
}

// NewPackageList constructs a PackageList with initial items and callbacks.
func NewPackageList(items []viewmodel.PackageVM, cb PackageListCallbacks) *PackageList {
	pl := &PackageList{items: items, callbacks: cb}
	pl.vbox = container.NewVBox()
	pl.scroll = container.NewVScroll(pl.vbox)
	pl.ExtendBaseWidget(pl)
	pl.rebuildCards()
	return pl
}

// SetItems replaces the displayed items and re-renders the card list.
func (pl *PackageList) SetItems(items []viewmodel.PackageVM) {
	pl.items = items
	pl.rebuildCards()
}

// CreateRenderer delegates to the scroll container.
func (pl *PackageList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(pl.scroll)
}

// rebuildCards clears and reconstructs one card per package item.
func (pl *PackageList) rebuildCards() {
	objects := make([]fyne.CanvasObject, 0, len(pl.items))
	for i := range pl.items {
		objects = append(objects, pl.buildCard(pl.items[i]))
	}
	pl.vbox.Objects = objects
	pl.vbox.Refresh()
}

// buildCard constructs a compact, styled row-card for the given package view model.
//
// Layout:
//
//	┌──────────────────────────────────────────────────────────┐
//	│ [☑]  DisplayName (bold)                  [v] [✏] [🗑]  │
//	│       package-name  ·  instalado - versão: 14.1.1        │
//	└──────────────────────────────────────────────────────────┘
func (pl *PackageList) buildCard(vm viewmodel.PackageVM) fyne.CanvasObject {
	check := widget.NewCheck("", nil)
	check.Checked = vm.Selected
	check.OnChanged = func(checked bool) { pl.callbacks.OnToggle(vm.ID, checked) }

	// Três segmentos numa única widget: display name (linha 1),
	// package name + status label inline (linha 2).
	textWidget := widget.NewRichText(
		&widget.TextSegment{
			Text:  vm.DisplayName,
			Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Bold: true}, Inline: false},
		},
		&widget.TextSegment{
			Text:  vm.PackageName,
			Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Monospace: true}, Inline: true},
		},
		&widget.TextSegment{
			Text:  "  ·  " + vm.StatusLabel(),
			Style: widget.RichTextStyle{ColorName: vm.StatusColorName(), Inline: true},
		},
	)

	// Badge mostra a versão definida ou "latest" — sem alteração de cor por status.
	badgeColor := uitheme.ColorCard
	if vm.HasVersion() {
		badgeColor = uitheme.ColorPrimary
	}
	versionBadge := NewBadge(vm.VersionLabel(), badgeColor)

	editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
		pl.callbacks.OnEdit(vm)
	})
	editBtn.Importance = widget.LowImportance

	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		pl.callbacks.OnDelete(vm)
	})
	deleteBtn.Importance = widget.LowImportance

	actions := container.NewHBox(versionBadge, editBtn, deleteBtn)
	row := container.NewBorder(nil, nil, check, actions, textWidget)

	bg := canvas.NewRectangle(uitheme.ColorCard)
	bg.CornerRadius = 6

	return container.NewVBox(container.NewStack(bg, container.NewPadded(row)))
}
