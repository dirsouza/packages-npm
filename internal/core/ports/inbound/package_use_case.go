package inbound

import (
	"github.com/dirsouza/packages-npm/internal/core/domain"
	"github.com/dirsouza/packages-npm/internal/core/ports/outbound"
)

// PackageDTO carries data between the UI and use-case layer.
// It avoids exposing domain internals to outer layers.
type PackageDTO struct {
	ID          int64
	DisplayName string
	PackageName string
	Version     string
}

// InstalledInfo carries the result of checking one package's installation state.
type InstalledInfo struct {
	PackageName      string
	InstalledVersion string // empty if not installed
	IsInstalled      bool
}

// PackageUseCase is the inbound port — the contract the UI depends on.
// Each method maps to a distinct use case (SRP + ISP).
type PackageUseCase interface {
	ListAll(order domain.SortOrder) ([]*domain.Package, error)
	Add(dto PackageDTO) (*domain.Package, error)
	Update(dto PackageDTO) error
	Delete(id int64) error
	InstallSelected(ids []int64, onProgress outbound.ProgressFn) error

	// UninstallSelected remove globalmente os pacotes selecionados via npm.
	// Apenas executa para os IDs fornecidos — a camada de UI é responsável
	// por filtrar somente os pacotes instalados antes de chamar este método.
	UninstallSelected(ids []int64, onProgress outbound.ProgressFn) error

	// CheckInstalled returns installation info for all known packages.
	// It does NOT block on failure — an error is returned instead of panicking.
	CheckInstalled() (map[string]InstalledInfo, error)

	// ExportBackup serializa todos os pacotes para um arquivo CSV no caminho indicado.
	ExportBackup(path string) error

	// ImportBackup restaura pacotes de um arquivo CSV.
	// ImportReplace apaga todos os dados antes de importar; ImportMerge apenas adiciona os ausentes.
	ImportBackup(path string, mode domain.ImportMode) (imported int, err error)
}
