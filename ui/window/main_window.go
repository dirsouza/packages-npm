package window

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/dirsouza/packages-npm/internal/core/domain"
	"github.com/dirsouza/packages-npm/internal/core/ports/inbound"
	"github.com/dirsouza/packages-npm/ui/components"
	"github.com/dirsouza/packages-npm/ui/viewmodel"
)

// MainWindow orchestrates the main window layout and user interactions.
// Presenter role: calls use cases and keeps the view in sync.
type MainWindow struct {
	window    fyne.Window
	useCase   inbound.PackageUseCase
	items     []viewmodel.PackageVM
	pkgList   *components.PackageList
	statusBar *widget.Label
	statsBar  *widget.Label
	sortOrder domain.SortOrder
	version   string
	build     string
}

// NewMainWindow constructs the presenter with its dependencies.
func NewMainWindow(w fyne.Window, useCase inbound.PackageUseCase, version, build string) *MainWindow {
	return &MainWindow{
		window:    w,
		useCase:   useCase,
		sortOrder: domain.SortByID,
		version:   version,
		build:     build,
	}
}

// Build assembles the window layout and loads initial data.
//
// Layout (top → bottom):
//
//	AppHeader        — branded title bar
//	Toolbar + Stats  — ações principais + contador (alinhado à direita)
//	Separator
//	SelectionBar     — "Selecionar todos" / "Desmarcar todos" (discreto)
//	PackageList      — lista scrollável de cards (ocupa o espaço restante)
//	Separator
//	StatusBar        — última mensagem de operação
func (m *MainWindow) Build() {
	m.statusBar = components.NewStatusBar("Verificando pacotes instalados...")
	m.statsBar = components.NewStatsBar(0, 0)

	m.pkgList = components.NewPackageList(nil, components.PackageListCallbacks{
		OnEdit:   m.onEdit,
		OnDelete: m.onDelete,
		OnToggle: m.onToggle,
	})

	toolbar := components.NewToolbar(m.onAdd, m.onInstall, m.onUninstall, m.onExport, m.onImport)
	topBar := container.NewBorder(nil, nil, toolbar, m.statsBar)

	selectionBar := components.NewSelectionBar(m.selectAll, m.deselectAll)
	sortSelector := m.buildSortSelector()
	controlBar := container.NewBorder(nil, nil, selectionBar, sortSelector)

	header := components.NewAppHeader()
	top := container.NewVBox(
		header,
		container.NewPadded(topBar),
		widget.NewSeparator(),
		container.NewPadded(controlBar),
		widget.NewSeparator(),
	)
	versionLabel := canvas.NewText(
		fmt.Sprintf("Versão: %s (Build: %s)", m.version, m.build),
		color.RGBA{R: 148, G: 163, B: 184, A: 255}, // slate-400 — discreto mas legível
	)
	versionLabel.TextSize = 11
	versionLabel.Alignment = fyne.TextAlignTrailing

	bottomBar := container.NewBorder(nil, nil, m.statusBar, versionLabel)
	bottom := container.NewVBox(widget.NewSeparator(), container.NewPadded(bottomBar))

	m.window.SetContent(container.NewBorder(top, bottom, nil, nil, m.pkgList))

	m.reload()
	go m.refreshInstallStatus()
}

// ── event handlers ────────────────────────────────────────────────────────────

func (m *MainWindow) onAdd() {
	components.ShowPackageForm("Adicionar Pacote", "", "", "", m.window,
		func(displayName, packageName, version string) {
			if _, err := m.useCase.Add(inbound.PackageDTO{
				DisplayName: displayName,
				PackageName: packageName,
				Version:     version,
			}); err != nil {
				components.ShowError(err, m.window)
				return
			}
			m.setStatus("✅  Pacote \"" + displayName + "\" adicionado.")
			m.reload()
		})
}

func (m *MainWindow) onEdit(vm viewmodel.PackageVM) {
	components.ShowPackageForm("Editar Pacote", vm.DisplayName, vm.PackageName, vm.Version, m.window,
		func(displayName, packageName, version string) {
			if err := m.useCase.Update(inbound.PackageDTO{
				ID:          vm.ID,
				DisplayName: displayName,
				PackageName: packageName,
				Version:     version,
			}); err != nil {
				components.ShowError(err, m.window)
				return
			}
			m.setStatus("✏️  Pacote \"" + displayName + "\" atualizado.")
			m.reload()
		})
}

func (m *MainWindow) onDelete(vm viewmodel.PackageVM) {
	components.ShowConfirm(
		"Remover Pacote",
		"Deseja remover \""+vm.DisplayName+"\"?\nEsta ação não pode ser desfeita.",
		m.window,
		func() {
			if err := m.useCase.Delete(vm.ID); err != nil {
				components.ShowError(err, m.window)
				return
			}
			m.setStatus("🗑️  Pacote \"" + vm.DisplayName + "\" removido.")
			m.reload()
		},
	)
}

func (m *MainWindow) onToggle(id int64, selected bool) {
	for i := range m.items {
		if m.items[i].ID == id {
			m.items[i].Selected = selected
			break
		}
	}
	m.updateStats()
}

func (m *MainWindow) onInstall() {
	selected := m.selectedIDs()
	if len(selected) == 0 {
		components.ShowInfo("Nenhum pacote selecionado",
			"Marque ao menos um pacote antes de instalar.", m.window)
		return
	}

	progressDlg := components.NewProgressDialog("Instalando Pacotes", "instalação", m.window)
	var failures []string

	go func() {
		err := m.useCase.InstallSelected(selected, func(target string, done, total int, installErr error) {
			if installErr != nil {
				failures = append(failures, target)
			}
			fyne.Do(func() {
				progressDlg.Update(target, done, total, installErr)
			})
		})
		if err != nil {
			fyne.Do(func() {
				progressDlg.Close()
				components.ShowError(err, m.window)
			})
			return
		}
		fyne.Do(func() {
			progressDlg.Finish(m.window, failures)
			m.setStatus("⬇️  Instalação concluída.")
			m.deselectAll()
		})
		go m.refreshInstallStatus()
	}()
}

func (m *MainWindow) onUninstall() {
	// Filtra apenas pacotes selecionados E instalados — não faz sentido
	// tentar desinstalar um pacote que não está presente no sistema.
	var ids []int64
	var names []string
	for i := range m.items {
		if m.items[i].Selected && m.items[i].InstallStatus == viewmodel.StatusInstalled {
			ids = append(ids, m.items[i].ID)
			names = append(names, m.items[i].PackageName)
		}
	}

	if len(ids) == 0 {
		components.ShowInfo(
			"Nenhum pacote elegível",
			"Selecione ao menos um pacote que esteja instalado para desinstalar.",
			m.window,
		)
		return
	}

	msg := fmt.Sprintf("Deseja desinstalar %d pacote(s) globalmente?\n\n", len(names))
	for _, n := range names {
		msg += "  • " + n + "\n"
	}

	components.ShowConfirm("Confirmar Desinstalação", msg, m.window, func() {
		progressDlg := components.NewProgressDialog("Desinstalando Pacotes", "Desinstalando", m.window)
		var failures []string

		go func() {
			err := m.useCase.UninstallSelected(ids, func(target string, done, total int, opErr error) {
				if opErr != nil {
					failures = append(failures, target)
				}
				fyne.Do(func() {
					progressDlg.Update(target, done, total, opErr)
				})
			})
			if err != nil {
				fyne.Do(func() {
					progressDlg.Close()
					components.ShowError(err, m.window)
				})
				return
			}
			fyne.Do(func() {
				progressDlg.Finish(m.window, failures)
				m.setStatus("🗑️  Desinstalação concluída.")
				m.deselectAll()
			})
			go m.refreshInstallStatus()
		}()
	})
}

// ── backup handlers ───────────────────────────────────────────────────────────

func (m *MainWindow) onExport() {
	dlg := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		path := writer.URI().Path()
		writer.Close() // fechar antes: o adapter usa os.WriteFile

		if exportErr := m.useCase.ExportBackup(path); exportErr != nil {
			components.ShowError(exportErr, m.window)
			return
		}
		components.ShowInfo("Backup exportado",
			fmt.Sprintf("Backup salvo com sucesso em:\n%s", path), m.window)
		m.setStatus("💾  Backup exportado.")
	}, m.window)

	dlg.SetFileName("packages-npm-backup.csv")
	dlg.SetFilter(storage.NewExtensionFileFilter([]string{".csv"}))
	dlg.Show()
}

func (m *MainWindow) onImport() {
	dlg := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		path := reader.URI().Path()
		reader.Close()

		// Pergunta o modo de importação antes de prosseguir.
		dialog.ShowCustomConfirm(
			"Modo de Importação",
			"Substituir tudo", "Mesclar",
			widget.NewLabel("Como deseja restaurar o backup?\n\n"+
				"• Substituir tudo — apaga os pacotes atuais e importa o backup completo.\n"+
				"• Mesclar — mantém os atuais e adiciona apenas os ausentes."),
			func(replace bool) {
				mode := domain.ImportMerge
				if replace {
					mode = domain.ImportReplace
				}
				m.setStatus("⏳  Importando backup...")
				go func() {
					imported, importErr := m.useCase.ImportBackup(path, mode)
					fyne.Do(func() {
						if importErr != nil {
							components.ShowError(importErr, m.window)
							m.setStatus("⚠️  Falha ao importar backup.")
							return
						}
						components.ShowInfo("Backup importado",
							fmt.Sprintf("%d pacote(s) importado(s) com sucesso.", imported), m.window)
						m.setStatus(fmt.Sprintf("📦  %d pacote(s) importado(s).", imported))
						m.reload()
						go m.refreshInstallStatus()
					})
				}()
			},
			m.window,
		)
	}, m.window)

	dlg.SetFilter(storage.NewExtensionFileFilter([]string{".csv"}))
	dlg.Show()
}

func (m *MainWindow) reload() {
	packages, err := m.useCase.ListAll(m.sortOrder)
	if err != nil {
		components.ShowError(err, m.window)
		return
	}
	prevSelected := m.selectedIDSet()

	// Preserve selection and installation status across reloads.
	prevStatus := make(map[int64]viewmodel.PackageVM, len(m.items))
	for i := range m.items {
		prevStatus[m.items[i].ID] = m.items[i]
	}

	m.items = viewmodel.FromDomainList(packages)
	for i := range m.items {
		id := m.items[i].ID
		m.items[i].Selected = prevSelected[id]
		if prev, ok := prevStatus[id]; ok {
			m.items[i].InstallStatus = prev.InstallStatus
			m.items[i].InstalledVersion = prev.InstalledVersion
		}
	}
	m.pkgList.SetItems(m.items)
	m.updateStats()
}

// refreshInstallStatus queries npm for globally installed packages and updates
// each card's badge. Designed to run in a goroutine — all UI updates are
// dispatched to the main loop with fyne.Do.
func (m *MainWindow) refreshInstallStatus() {
	installed, err := m.useCase.CheckInstalled()
	if err != nil {
		fyne.Do(func() {
			m.setStatus("⚠️  Não foi possível verificar pacotes: " + err.Error())
		})
		return
	}

	fyne.Do(func() {
		for i := range m.items {
			info, found := installed[m.items[i].PackageName]
			if !found {
				m.items[i].InstallStatus = viewmodel.StatusMissing
				continue
			}
			m.items[i].ResolveInstallStatus(info.InstalledVersion, info.IsInstalled)
		}

		m.pkgList.SetItems(m.items)
		m.setStatus("✅  Status de instalação atualizado.")
	})
}

func (m *MainWindow) updateStats() {
	components.UpdateStatsBar(m.statsBar, len(m.items), len(m.selectedIDs()))
}

func (m *MainWindow) selectedIDs() []int64 {
	ids := make([]int64, 0)
	for i := range m.items {
		if m.items[i].Selected {
			ids = append(ids, m.items[i].ID)
		}
	}
	return ids
}

func (m *MainWindow) selectedIDSet() map[int64]bool {
	set := make(map[int64]bool, len(m.items))
	for i := range m.items {
		if m.items[i].Selected {
			set[m.items[i].ID] = true
		}
	}
	return set
}

func (m *MainWindow) setStatus(text string) { m.statusBar.SetText(text) }

// ── sort helper ───────────────────────────────────────────────────────────────

// buildSortSelector retorna um widget.Select com as opções de ordenação.
// Ao mudar, atualiza m.sortOrder e recarrega a lista imediatamente.
func (m *MainWindow) buildSortSelector() *fyne.Container {
	options := []string{"ID", "Nome"}
	sel := widget.NewSelect(options, func(chosen string) {
		if chosen == "ID" {
			m.sortOrder = domain.SortByID
		} else {
			m.sortOrder = domain.SortByDisplayName
		}
		m.reload()
	})
	sel.SetSelected("ID") // default

	label := canvas.NewText("Ordenar por:", color.White)
	label.TextSize = 13
	return container.NewHBox(label, sel)
}

func (m *MainWindow) selectAll() {
	for i := range m.items {
		m.items[i].Selected = true
	}
	m.pkgList.SetItems(m.items)
	m.updateStats()
}

func (m *MainWindow) deselectAll() {
	for i := range m.items {
		m.items[i].Selected = false
	}
	m.pkgList.SetItems(m.items)
	m.updateStats()
}
