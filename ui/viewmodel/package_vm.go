package viewmodel

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/dirsouza/packages-npm/internal/core/domain"
)

// InstallStatus represents the installation state of a package.
type InstallStatus uint8

const (
	StatusUnknown   InstallStatus = iota // ainda não verificado
	StatusMissing                        // não instalado globalmente
	StatusInstalled                      // instalado na versão correta (ou any, se sem versão)
	StatusMismatch                       // instalado, mas em versão diferente
)

// PackageVM is a flat, UI-friendly representation of a domain.Package.
// It decouples the UI from the domain model (Bridge pattern).
type PackageVM struct {
	ID               int64
	DisplayName      string
	PackageName      string
	Version          string
	Selected         bool
	InstallStatus    InstallStatus
	InstalledVersion string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// FromDomain maps a domain entity to a view model.
func FromDomain(p *domain.Package) PackageVM {
	return PackageVM{
		ID:            p.ID,
		DisplayName:   p.DisplayName,
		PackageName:   p.Name.Value(),
		Version:       p.Version.Value(),
		InstallStatus: StatusUnknown,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

// FromDomainList maps a slice of domain entities to view models.
func FromDomainList(packages []*domain.Package) []PackageVM {
	vms := make([]PackageVM, 0, len(packages))
	for _, p := range packages {
		vms = append(vms, FromDomain(p))
	}
	return vms
}

// StatusLabel returns the human-readable installation status for display inline
// next to the package name. Color mapping is handled by the UI component.
func (vm PackageVM) StatusLabel() string {
	switch vm.InstallStatus {
	case StatusInstalled:
		return "instalado - versão: " + vm.InstalledVersion
	case StatusMismatch:
		return "instalado - versão: " + vm.InstalledVersion + " (esperado: " + vm.VersionLabel() + ")"
	case StatusMissing:
		return "não instalado"
	default: // StatusUnknown
		return "verificando..."
	}
}

// StatusColorName maps InstallStatus to a Fyne theme color name.
func (vm PackageVM) StatusColorName() fyne.ThemeColorName {
	switch vm.InstallStatus {
	case StatusInstalled:
		return theme.ColorNameSuccess
	case StatusMismatch:
		return theme.ColorNameWarning
	case StatusMissing, StatusUnknown:
		return theme.ColorNamePlaceHolder
	default:
		return theme.ColorNamePlaceHolder
	}
}

// VersionLabel returns a human-readable version label for the defined version.
func (vm PackageVM) VersionLabel() string {
	if vm.Version == "" {
		return "latest"
	}
	return vm.Version
}

// HasVersion reports whether a specific version is pinned.
func (vm PackageVM) HasVersion() bool { return vm.Version != "" }

// ResolveInstallStatus computes the InstallStatus based on what is installed
// versus what is expected. Call this after receiving CheckInstalled results.
func (vm *PackageVM) ResolveInstallStatus(installedVersion string, isInstalled bool) {
	if !isInstalled {
		vm.InstallStatus = StatusMissing
		vm.InstalledVersion = ""
		return
	}
	vm.InstalledVersion = installedVersion
	switch {
	case !vm.HasVersion():
		// No version pinned → any installed version is "correct".
		vm.InstallStatus = StatusInstalled
	case vm.Version == installedVersion:
		vm.InstallStatus = StatusInstalled
	default:
		vm.InstallStatus = StatusMismatch
	}
}
