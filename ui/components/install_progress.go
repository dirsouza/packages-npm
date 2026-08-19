package components

import (
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ProgressDialog é um diálogo reutilizável para operações com progresso
// (instalação e desinstalação). Recebe título e verbo de ação via construtor.
type ProgressDialog struct {
	dlg       dialog.Dialog
	progress  *widget.ProgressBar
	status    *widget.Label
	log       *widget.Label
	actionMsg string // "Instalando" ou "Desinstalando"
}

// NewProgressDialog cria e exibe um diálogo de progresso genérico.
func NewProgressDialog(title, actionMsg string, parent fyne.Window) *ProgressDialog {
	progress := widget.NewProgressBar()
	status := widget.NewLabel("Iniciando " + actionMsg + "...")
	log := widget.NewLabel("")
	log.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(status, progress, log)
	dlg := dialog.NewCustomWithoutButtons(title, content, parent)
	dlg.Resize(fyne.NewSize(480, 200))
	dlg.Show()

	return &ProgressDialog{dlg: dlg, progress: progress, status: status, log: log, actionMsg: actionMsg}
}

// Update avança a barra de progresso e atualiza o texto de status.
func (d *ProgressDialog) Update(pkgTarget string, done, total int, opErr error) {
	d.progress.SetValue(float64(done) / float64(total))
	d.status.SetText(fmt.Sprintf("%s %s... (%d/%d)", d.actionMsg, pkgTarget, done, total))

	if opErr != nil {
		d.log.SetText(d.log.Text + fmt.Sprintf("❌ %s\n", pkgTarget))
	} else {
		d.log.SetText(d.log.Text + fmt.Sprintf("✅ %s\n", pkgTarget))
	}
}

func (d *ProgressDialog) Close() {
	d.dlg.Hide()
}

// Finish fecha o diálogo e exibe mensagem de conclusão.
func (d *ProgressDialog) Finish(parent fyne.Window, failures []string) {
	d.dlg.Hide()

	if len(failures) == 0 {
		ShowInfo("Concluído", "Operação concluída com sucesso! 🎉", parent)
		return
	}

	msg := fmt.Sprintf("%d pacote(s) com falha:\n", len(failures))
	for _, f := range failures {
		msg += "• " + f + "\n"
	}
	dialog.ShowError(errors.New(msg), parent)
}
