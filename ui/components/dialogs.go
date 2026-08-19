package components

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ShowPackageForm shows a styled form dialog for adding or editing a package.
func ShowPackageForm(
	title string,
	displayName, packageName, version string,
	parent fyne.Window,
	onConfirm func(displayName, packageName, version string),
) {
	displayEntry := widget.NewEntry()
	displayEntry.SetPlaceHolder("Ex: TypeScript")
	displayEntry.SetText(displayName)

	packageEntry := widget.NewEntry()
	packageEntry.SetPlaceHolder("Ex: typescript  ou  @nestjs/cli")
	packageEntry.SetText(packageName)

	versionEntry := widget.NewEntry()
	versionEntry.SetPlaceHolder("Ex: 5.0.0 — deixe vazio para instalar a versão mais recente")
	versionEntry.SetText(version)

	items := []*widget.FormItem{
		{Text: "Nome de Exibição", Widget: displayEntry, HintText: "Obrigatório"},
		{Text: "Pacote npm", Widget: packageEntry, HintText: "Obrigatório"},
		{Text: "Versão", Widget: versionEntry, HintText: "Opcional — omita para usar a latest"},
	}

	dialog.ShowForm(title, "Salvar", "Cancelar", items, func(confirmed bool) {
		if !confirmed {
			return
		}
		if displayEntry.Text == "" || packageEntry.Text == "" {
			dialog.ShowError(errRequiredFields, parent)
			return
		}
		onConfirm(displayEntry.Text, packageEntry.Text, versionEntry.Text)
	}, parent)
}

// ShowError forwards to dialog.ShowError.
func ShowError(err error, parent fyne.Window) { dialog.ShowError(err, parent) }

// ShowInfo forwards to dialog.ShowInformation.
func ShowInfo(title, message string, parent fyne.Window) {
	dialog.ShowInformation(title, message, parent)
}

// ShowConfirm shows a yes/no confirmation dialog.
func ShowConfirm(title, message string, parent fyne.Window, onConfirm func()) {
	dialog.ShowConfirm(title, message, func(ok bool) {
		if ok {
			onConfirm()
		}
	}, parent)
}

// NewToolbar returns a styled top action bar.
// Agrupamento visual: [Adicionar] | [Instalar] [Desinstalar] | [Exportar] [Importar].
func NewToolbar(onAdd, onInstall, onUninstall, onExport, onImport func()) *fyne.Container {
	addBtn := widget.NewButtonWithIcon("Adicionar", theme.ContentAddIcon(), onAdd)
	addBtn.Importance = widget.HighImportance

	installBtn := widget.NewButtonWithIcon("Instalar", theme.DownloadIcon(), onInstall)
	installBtn.Importance = widget.SuccessImportance

	uninstallBtn := widget.NewButtonWithIcon("Desinstalar", theme.DeleteIcon(), onUninstall)
	uninstallBtn.Importance = widget.DangerImportance

	exportBtn := widget.NewButtonWithIcon("Exportar", theme.DocumentSaveIcon(), onExport)
	exportBtn.Importance = widget.LowImportance

	importBtn := widget.NewButtonWithIcon("Importar", theme.FolderOpenIcon(), onImport)
	importBtn.Importance = widget.LowImportance

	sep1 := widget.NewSeparator()
	sep2 := widget.NewSeparator()

	return container.NewHBox(addBtn, sep1, installBtn, uninstallBtn, sep2, exportBtn, importBtn)
}

// NewSelectionBar retorna controles para selecionar/desmarcar todos os pacotes.
// Fica entre o toolbar principal e a lista, com botões discretos (LowImportance).
func NewSelectionBar(onSelectAll, onDeselectAll func()) *fyne.Container {
	selAll := widget.NewButton("Selecionar todos", onSelectAll)
	selAll.Importance = widget.LowImportance

	deselAll := widget.NewButton("Desmarcar todos", onDeselectAll)
	deselAll.Importance = widget.LowImportance

	return container.NewHBox(selAll, deselAll)
}

// NewStatsBar returns a label showing package count and selection count.
func NewStatsBar(total, selected int) *widget.Label {
	label := widget.NewLabel(statsText(total, selected))
	label.Alignment = fyne.TextAlignTrailing
	return label
}

// UpdateStatsBar refreshes the stats label.
func UpdateStatsBar(bar *widget.Label, total, selected int) {
	bar.SetText(statsText(total, selected))
}

// NewStatusBar returns a label used as a bottom status message bar.
func NewStatusBar(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Alignment = fyne.TextAlignLeading
	return label
}

func statsText(total, selected int) string {
	return fmt.Sprintf("%d pacote(s)  ·  %d selecionado(s)", total, selected)
}

var errRequiredFields = fmt.Errorf("nome de exibição e nome do pacote são obrigatórios")
